package handler

import (
	"errors"

	"pulsa-backend/model"
	"pulsa-backend/repository"
	"pulsa-backend/service"
	"pulsa-backend/util"
	"pulsa-backend/worker"

	"github.com/gin-gonic/gin"
)

type TopupHandler struct {
	topupSvc *service.TopupService
	pool     *worker.Pool
}

func NewTopupHandler(s *service.TopupService, p *worker.Pool) *TopupHandler {
	return &TopupHandler{topupSvc: s, pool: p}
}

type topupReq struct {
	ProductID  string   `json:"product_id" binding:"required"`
	CustomerNo string   `json:"customer_no" binding:"required,min=8,max=20"`
	Keuntungan *float64 `json:"keuntungan"`
	Pin        string   `json:"pin" binding:"required,len=6,numeric"`
}

func (h *TopupHandler) Topup(c *gin.Context) {
	agentID := c.GetInt64("agent_id")
	var req topupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	// Kirim req.Pin ke service
	if req.Keuntungan != nil && (*req.Keuntungan < model.MinAgentProfit || *req.Keuntungan > model.MaxAgentProfit) {
		util.BadRequest(c, "keuntungan must be between Rp500 and Rp2.000")
		return
	}

	t, err := h.topupSvc.CreatePending(c.Request.Context(), agentID, req.ProductID, req.CustomerNo, req.Pin, req.Keuntungan)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProductNotFound):
			util.NotFound(c, "product not found or inactive")
		case errors.Is(err, service.ErrAgentInactive):
			util.Forbidden(c, "agent is inactive")
		case errors.Is(err, service.ErrPINLocked):
			util.TooManyRequests(c, "PIN transaksi terkunci selama 15 menit")
		case errors.Is(err, service.ErrPINIncorrect):
			util.Forbidden(c, "PIN transaksi salah")
		case errors.Is(err, service.ErrPINNotSet):
			util.Forbidden(c, "PIN transaksi belum diatur; hubungi superadmin")
		case errors.Is(err, service.ErrInsufficient):
			util.BadRequest(c, "insufficient balance")
		case errors.Is(err, service.ErrInvalidProfit):
			util.BadRequest(c, "keuntungan must be between Rp500 and Rp2.000")
		case errors.Is(err, repository.ErrNotFound):
			util.NotFound(c, "agent not found")
		default:
			util.Log.Errorf("[TOPUP] create pending error agent=%d: %v", agentID, err)
			util.InternalError(c, "failed to create transaction")
		}
		return
	}

	if !h.pool.Enqueue(worker.Job{TxID: t.ID}) {
		util.Log.Warnf("[TOPUP] queue full, tx=%s will be picked up by pending recovery", t.ID)
	}

	util.Accepted(c, "transaction accepted for processing", gin.H{
		"tx_id":          t.ID,
		"status":         "PENDING",
		"nominal":        t.Amount,
		"sell_price":     t.SellPrice,
		"keuntungan":     t.Markup,
		"customer_price": t.CustomerPrice,
	})
}
