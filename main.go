package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pulsa-backend/database"
	"pulsa-backend/handler"
	"pulsa-backend/internal/config"
	"pulsa-backend/middleware"
	"pulsa-backend/repository"
	"pulsa-backend/service"
	"pulsa-backend/util"
	"pulsa-backend/worker"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	util.InitLogger(cfg.AppEnv)

	db, err := database.Connect(cfg)
	if err != nil {
		util.Log.Fatal("[MAIN] DB connect failed: ", err)
	}
	defer db.Close()

	// --- Repository layer ---
	agentRepo := repository.NewAgentRepository(db)
	balanceHandler := handler.NewBalanceHandler(agentRepo)
	adminRepo := repository.NewAdminRepository(db)
	productRepo := repository.NewProductRepository(db)
	productHandler := handler.NewProductHandler(productRepo)
	txRepo := repository.NewTransactionRepository(db)
	depositRepo := repository.NewDepositRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)
	profileHandler := handler.NewProfileHandler(agentRepo)

	// --- Service layer ---
	callbackSvc := service.NewCallbackService(db, agentRepo, txRepo,
		cfg.CallbackTimeout, cfg.CallbackMaxRetry)
	authSvc := service.NewAuthService(agentRepo, adminRepo, []byte(cfg.JWTSecret), cfg.JWTExpireHours)
	topupSvc := service.NewTopupService(db, agentRepo, productRepo, txRepo, ledgerRepo, callbackSvc)
	adminSvc := service.NewAdminService(db, agentRepo, depositRepo, ledgerRepo)

	// --- Worker pool ---
	pool := worker.NewPool(cfg.WorkerCount, cfg.QueueSize, topupSvc)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	// --- Handler layer ---
	authHandler := handler.NewAuthHandler(authSvc)
	topupHandler := handler.NewTopupHandler(topupSvc, pool)
	historyHandler := handler.NewHistoryHandler(txRepo, depositRepo)
	adminHandler := handler.NewAdminHandler(adminSvc, agentRepo, txRepo)

	// --- Router ---
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.RequestLogger())
	r.Use(middleware.Recovery())
	r.Use(middleware.NewRateLimiter(10, 20).Middleware()) // 10 rps, burst 20 per IP

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "queue": pool.QueueSize()})
	})

	// --- PUBLIC ---
	r.POST("/api/agent/login", authHandler.AgentLogin)
	r.POST("/api/admin/login", authHandler.AdminLogin)

	// --- AGENT (protected) ---
	agent := r.Group("/api")
	agent.Use(middleware.AuthAgent([]byte(cfg.JWTSecret)))
	{
		agent.POST("/topup", topupHandler.Topup)
		agent.GET("/balance", balanceHandler.MyBalance) // ← BARU: cek saldo sendiri
		agent.GET("/products", productHandler.ListForAgent)
		agent.GET("/products/:id", productHandler.GetForAgent)
		agent.GET("/history/transactions", historyHandler.Transactions)
		agent.GET("/history/deposits", historyHandler.Deposits)
		agent.PATCH("/pin", profileHandler.UpdatePin)
	}

	// --- ADMIN (protected) ---
	admin := r.Group("/api/admin")
	admin.Use(middleware.AuthAdmin([]byte(cfg.JWTSecret)))
	{
		admin.POST("/deposit", adminHandler.Deposit)
		admin.GET("/agents", adminHandler.ListAgents)
		admin.GET("/transactions", adminHandler.ListTransactions)
		admin.PATCH("/agents/:id/pin", adminHandler.SetAgentPIN)
		admin.GET("/products", productHandler.ListForAdmin)
		admin.POST("/products", productHandler.Create)
		admin.PATCH("/products/:id", productHandler.Update)
		admin.DELETE("/products/:id", productHandler.Deactivate)
	}

	// --- Graceful shutdown ---
	srv := &http.Server{
		Addr:         net.JoinHostPort(cfg.AppHost, cfg.AppPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		util.Log.Infof("[MAIN] server starting on %s (env=%s)", srv.Addr, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			util.Log.Fatal("[MAIN] server error: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	util.Log.Info("[MAIN] shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	cancel()
	util.Log.Info("[MAIN] shutdown complete")
}
