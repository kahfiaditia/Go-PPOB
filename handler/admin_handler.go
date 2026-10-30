package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"

	"pulsa-backend/model"
	"pulsa-backend/repository"
	"pulsa-backend/service"
	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	adminSvc      *service.AdminService
	agentRepo     *repository.AgentRepository
	agentUserRepo *repository.AgentUserRepository
	txRepo        *repository.TransactionRepository
}

func NewAdminHandler(as *service.AdminService, ar *repository.AgentRepository,
	aur *repository.AgentUserRepository, tr *repository.TransactionRepository) *AdminHandler {
	return &AdminHandler{adminSvc: as, agentRepo: ar, agentUserRepo: aur, txRepo: tr}
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

func (h *AdminHandler) CreateAgent(c *gin.Context) {
	var req struct {
		Username          string `json:"username" binding:"required,min=7,max=50"`
		Email             string `json:"email" binding:"required,email,max=100"`
		Password          string `json:"password" binding:"required,min=6"`
		Name              string `json:"name" binding:"required,max=100"`
		Phone             string `json:"phone" binding:"required,max=20"`
		Address           string `json:"address" binding:"required,max=255"`
		BankName          string `json:"bank_name" binding:"required,max=100"`
		BankAccountNumber string `json:"bank_account_number" binding:"required,max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	if _, _, err := h.agentUserRepo.FindByEmail(c.Request.Context(), req.Email); err == nil {
		util.BadRequest(c, "email already exists")
		return
	} else if !errors.Is(err, repository.ErrNotFound) {
		util.Log.Errorf("[ADMIN] employee email lookup error: %v", err)
		util.InternalError(c, "failed to validate owner email")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		util.Log.Errorf("[ADMIN] owner password hash error: %v", err)
		util.InternalError(c, "failed to secure owner password")
		return
	}
	agent := &model.Agent{
		Username: req.Username, Email: req.Email, PasswordHash: string(passwordHash),
		Name: req.Name, Phone: req.Phone, Address: req.Address, BankName: req.BankName,
		BankAccount: req.BankAccountNumber, IsActive: true,
	}
	if err := h.agentRepo.Create(c.Request.Context(), agent); err != nil {
		util.Log.Errorf("[ADMIN] agent create error: %v", err)
		util.BadRequest(c, "failed to create agent (username or email may already exist)")
		return
	}
	util.Created(c, "agent owner created", agent)
}

func (h *AdminHandler) ListEmployees(c *gin.Context) {
	agentID, ok := parsePositiveID(c.Param("id"))
	if !ok {
		util.BadRequest(c, "invalid agent id")
		return
	}
	if _, err := h.agentRepo.FindByID(c.Request.Context(), agentID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			util.NotFound(c, "agent not found")
			return
		}
		util.Log.Errorf("[ADMIN] agent lookup error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to load agent")
		return
	}

	users, err := h.agentUserRepo.ListByAgent(c.Request.Context(), agentID)
	if err != nil {
		util.Log.Errorf("[ADMIN] employee list error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to list employees")
		return
	}
	util.OK(c, "employees", users)
}

func (h *AdminHandler) CreateEmployee(c *gin.Context) {
	agentID, ok := parsePositiveID(c.Param("id"))
	if !ok {
		util.BadRequest(c, "invalid agent id")
		return
	}
	agent, err := h.agentRepo.FindByID(c.Request.Context(), agentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			util.NotFound(c, "agent not found")
			return
		}
		util.Log.Errorf("[ADMIN] agent lookup error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to load agent")
		return
	}
	if !agent.IsActive {
		util.Forbidden(c, "agent is inactive")
		return
	}

	var req struct {
		Username string `json:"username" binding:"required,min=7,max=50"`
		Email    string `json:"email" binding:"required,email,max=100"`
		Name     string `json:"name" binding:"required,max=100"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	if _, err := h.agentRepo.FindByEmail(c.Request.Context(), req.Email); err == nil {
		util.BadRequest(c, "email already exists")
		return
	} else if !errors.Is(err, repository.ErrNotFound) {
		util.Log.Errorf("[ADMIN] owner email lookup error: %v", err)
		util.InternalError(c, "failed to validate employee email")
		return
	}
	if _, _, err := h.agentUserRepo.FindByEmail(c.Request.Context(), req.Email); err == nil {
		util.BadRequest(c, "email already exists")
		return
	} else if !errors.Is(err, repository.ErrNotFound) {
		util.Log.Errorf("[ADMIN] employee email lookup error: %v", err)
		util.InternalError(c, "failed to validate employee email")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		util.Log.Errorf("[ADMIN] employee password hash error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to secure employee password")
		return
	}
	user := &model.AgentUser{
		AgentID: agentID, Username: req.Username, Email: req.Email,
		PasswordHash: string(passwordHash), Name: req.Name, IsActive: true,
	}
	if err := h.agentUserRepo.Create(c.Request.Context(), user); err != nil {
		util.Log.Errorf("[ADMIN] employee create error agent=%d: %v", agentID, err)
		util.BadRequest(c, "failed to create employee (username or email may already exist)")
		return
	}
	util.Created(c, "employee created", user)
}

func (h *AdminHandler) SetEmployeeActive(c *gin.Context) {
	agentID, ok := parsePositiveID(c.Param("id"))
	if !ok {
		util.BadRequest(c, "invalid agent id")
		return
	}
	employeeID, ok := parsePositiveID(c.Param("employee_id"))
	if !ok {
		util.BadRequest(c, "invalid employee id")
		return
	}
	var req struct {
		IsActive *bool `json:"is_active" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, "is_active is required")
		return
	}
	if err := h.agentUserRepo.SetActive(c.Request.Context(), agentID, employeeID, *req.IsActive); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			util.NotFound(c, "employee not found")
			return
		}
		util.Log.Errorf("[ADMIN] employee status update error agent=%d employee=%d: %v", agentID, employeeID, err)
		util.InternalError(c, "failed to update employee")
		return
	}
	util.OK(c, "employee status updated", gin.H{
		"agent_id":    agentID,
		"employee_id": employeeID,
		"is_active":   *req.IsActive,
	})
}

func parsePositiveID(value string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0
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
