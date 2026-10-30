package handler

import (
	"errors"
	"strings"
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
	limit, offset, ok := parsePagination(c, 20)
	if !ok {
		return
	}
	status := c.Query("status")
	if !validTransactionStatus(status) {
		util.BadRequest(c, "status must be PENDING, SUCCESS, FAILED, or REFUNDED")
		return
	}
	dateFrom, err := parseReportDate(c.Query("date_from"))
	if err != nil {
		util.BadRequest(c, "date_from must use YYYY-MM-DD")
		return
	}
	dateTo, err := parseReportDate(c.Query("date_to"))
	if err != nil {
		util.BadRequest(c, "date_to must use YYYY-MM-DD")
		return
	}
	if dateTo != nil {
		endExclusive := dateTo.AddDate(0, 0, 1)
		dateTo = &endExclusive
	}
	if dateFrom != nil && dateTo != nil && !dateFrom.Before(*dateTo) {
		util.BadRequest(c, "date_from must not be after date_to")
		return
	}

	operator := strings.TrimSpace(c.Query("operator"))
	txs, err := h.txRepo.ListByAgent(c.Request.Context(), agentID, limit, offset,
		status, operator, dateFrom, dateTo)
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
	Markup           float64   `json:"keuntungan"`
	CustomerPrice    float64   `json:"customer_price"`
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
		Amount: tx.Amount, SellPrice: tx.SellPrice, Markup: tx.Markup,
		CustomerPrice: tx.CustomerPrice, Status: tx.Status, FailureReason: tx.FailureReason,
		CallbackStatus: tx.CallbackStatus, CallbackAttempts: tx.CallbackAttempts,
		CreatedAt: tx.CreatedAt, UpdatedAt: tx.UpdatedAt,
	}
}

func (h *HistoryHandler) Deposits(c *gin.Context) {
	agentID := c.GetInt64("agent_id")
	limit, offset, ok := parsePagination(c, 20)
	if !ok {
		return
	}

	ds, err := h.depositRepo.ListByAgent(c.Request.Context(), agentID, limit, offset)
	if err != nil {
		util.Log.Errorf("[HISTORY] deposit error: %v", err)
		util.InternalError(c, "failed to load deposits")
		return
	}
	util.OK(c, "deposits", ds)
}

func (h *HistoryHandler) DashboardSummary(c *gin.Context) {
	now := time.Now()
	dateFrom := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dateTo := dateFrom.AddDate(0, 0, 1)
	summary, err := h.txRepo.SummaryByAgent(c.Request.Context(), c.GetInt64("agent_id"), dateFrom, dateTo)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			util.NotFound(c, "agent not found")
			return
		}
		util.Log.Errorf("[DASHBOARD] summary error agent=%d: %v", c.GetInt64("agent_id"), err)
		util.InternalError(c, "failed to load dashboard summary")
		return
	}
	util.OK(c, "dashboard summary", summary)
}

func parseReportDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	date, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, err
	}
	return &date, nil
}

func validTransactionStatus(status string) bool {
	return status == "" || status == "PENDING" || status == "SUCCESS" ||
		status == "FAILED" || status == "REFUNDED"
}
