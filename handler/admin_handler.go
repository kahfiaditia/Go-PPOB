package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"

	"pulsa-backend/repository"
	"pulsa-backend/service"
	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	adminSvc  *service.AdminService
	agentRepo *repository.AgentRepository
	txRepo    *repository.TransactionRepository
}

func NewAdminHandler(as *service.AdminService, ar *repository.AgentRepository,
	tr *repository.TransactionRepository) *AdminHandler {
	return &AdminHandler{adminSvc: as, agentRepo: ar, txRepo: tr}
}

type depositReq struct {
	AgentID   int64   `json:"agent_id" binding:"required"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	Reference string  `json:"reference"`
	Note      string  `json:"note"`
}

func (h *AdminHandler) Deposit(c *gin.Context) {
	adminID := c.GetInt64("admin_id")
	var req depositReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	d, err := h.adminSvc.Deposit(c.Request.Context(), adminID, req.AgentID,
		req.Amount, req.Reference, req.Note)
	if err != nil {
		util.BadRequest(c, err.Error())
		return
	}
	util.Created(c, "deposit success", d)
}

func (h *AdminHandler) ListAgents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	agents, err := h.agentRepo.List(c.Request.Context(), limit, offset)
	if err != nil {
		util.InternalError(c, "failed to list agents")
		return
	}
	util.OK(c, "agents", agents)
}

func (h *AdminHandler) ListTransactions(c *gin.Context) {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	txs, err := h.txRepo.ListAll(c.Request.Context(), status, limit, offset)
	if err != nil {
		util.InternalError(c, "failed to list transactions")
		return
	}
	util.OK(c, "transactions", txs)
}

func (h *AdminHandler) SetAgentPIN(c *gin.Context) {
	if c.GetString("admin_role") != "superadmin" {
		util.Forbidden(c, "superadmin access only")
		return
	}

	agentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || agentID <= 0 {
		util.BadRequest(c, "invalid agent id")
		return
	}

	var req struct {
		PIN string `json:"pin" binding:"required,len=6,numeric"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, "PIN harus berupa 6 angka")
		return
	}
	if _, err := h.agentRepo.FindByID(c.Request.Context(), agentID); err != nil {
		if err == repository.ErrNotFound {
			util.NotFound(c, "agent not found")
			return
		}
		util.InternalError(c, "failed to load agent")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.PIN), bcrypt.DefaultCost)
	if err != nil {
		util.Log.Errorf("[ADMIN] PIN hash error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to secure PIN")
		return
	}
	if err := h.agentRepo.UpdatePin(c.Request.Context(), agentID, string(hash)); err != nil {
		util.Log.Errorf("[ADMIN] PIN update error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to update PIN")
		return
	}
	util.OK(c, "agent PIN updated", gin.H{"agent_id": agentID})
}

func (h *AdminHandler) RotateAgentCallbackSecret(c *gin.Context) {
	if c.GetString("admin_role") != "superadmin" {
		util.Forbidden(c, "superadmin access only")
		return
	}

	agentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || agentID <= 0 {
		util.BadRequest(c, "invalid agent id")
		return
	}

	randomSecret := make([]byte, 32)
	if _, err := rand.Read(randomSecret); err != nil {
		util.Log.Errorf("[ADMIN] callback secret generation error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to generate callback secret")
		return
	}
	secret := base64.RawURLEncoding.EncodeToString(randomSecret)

	if err := h.agentRepo.SetCallbackSecret(c.Request.Context(), agentID, secret); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			util.NotFound(c, "agent not found")
			return
		}
		util.Log.Errorf("[ADMIN] callback secret update error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to update callback secret")
		return
	}

	util.OK(c, "callback secret rotated; store it securely because it will not be shown again",
		gin.H{"agent_id": agentID, "callback_secret": secret})
}
