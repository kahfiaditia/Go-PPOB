package handler

import (
	"pulsa-backend/repository"
	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
)

type BalanceHandler struct {
	agentRepo *repository.AgentRepository
}

func NewBalanceHandler(ar *repository.AgentRepository) *BalanceHandler {
	return &BalanceHandler{agentRepo: ar}
}

// GET /api/balance — agent lihat saldo sendiri
func (h *BalanceHandler) MyBalance(c *gin.Context) {
	agentID := c.GetInt64("agent_id")

	data, err := h.agentRepo.GetBalance(c.Request.Context(), agentID)
	if err != nil {
		if err == repository.ErrNotFound {
			util.NotFound(c, "agent not found")
			return
		}
		util.Log.Errorf("[BALANCE] error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to load balance")
		return
	}
	util.OK(c, "balance retrieved", data)
}
