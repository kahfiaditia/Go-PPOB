package handler

import (
	"strconv"

	"pulsa-backend/repository"
	"pulsa-backend/service"
	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
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
