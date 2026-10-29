package handler

import (
	"strconv"

	"pulsa-backend/repository"
	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
)

type HistoryHandler struct {
	txRepo      *repository.TransactionRepository
	depositRepo *repository.DepositRepository
}

func NewHistoryHandler(tr *repository.TransactionRepository, dr *repository.DepositRepository) *HistoryHandler {
	return &HistoryHandler{txRepo: tr, depositRepo: dr}
}

func (h *HistoryHandler) Transactions(c *gin.Context) {
	agentID := c.GetInt64("agent_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit > 100 {
		limit = 100
	}

	txs, err := h.txRepo.ListByAgent(c.Request.Context(), agentID, limit, offset)
	if err != nil {
		util.Log.Errorf("[HISTORY] error: %v", err)
		util.InternalError(c, "failed to load history")
		return
	}
	util.OK(c, "transactions", txs)
}

func (h *HistoryHandler) Deposits(c *gin.Context) {
	agentID := c.GetInt64("agent_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit > 100 {
		limit = 100
	}

	ds, err := h.depositRepo.ListByAgent(c.Request.Context(), agentID, limit, offset)
	if err != nil {
		util.Log.Errorf("[HISTORY] deposit error: %v", err)
		util.InternalError(c, "failed to load deposits")
		return
	}
	util.OK(c, "deposits", ds)
}
