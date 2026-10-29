package handler

import (
	"strconv"
	"time"

	"pulsa-backend/model"
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
	response := make([]agentTransactionResponse, 0, len(txs))
	for _, tx := range txs {
		response = append(response, toAgentTransactionResponse(tx))
	}
	util.OK(c, "transactions", response)
}

type agentTransactionResponse struct {
	ID               string    `json:"id"`
	AgentID          int64     `json:"agent_id"`
	ProductID        string    `json:"product_id"`
	CustomerNo       string    `json:"customer_no"`
	Amount           float64   `json:"amount"`
	SellPrice        float64   `json:"sell_price"`
	Status           string    `json:"status"`
	FailureReason    string    `json:"failure_reason,omitempty"`
	CallbackStatus   string    `json:"callback_status"`
	CallbackAttempts int       `json:"callback_attempts"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func toAgentTransactionResponse(tx model.Transaction) agentTransactionResponse {
	return agentTransactionResponse{
		ID: tx.ID, AgentID: tx.AgentID, ProductID: tx.ProductID, CustomerNo: tx.CustomerNo,
		Amount: tx.Amount, SellPrice: tx.SellPrice, Status: tx.Status, FailureReason: tx.FailureReason,
		CallbackStatus: tx.CallbackStatus, CallbackAttempts: tx.CallbackAttempts,
		CreatedAt: tx.CreatedAt, UpdatedAt: tx.UpdatedAt,
	}
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
