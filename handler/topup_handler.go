package handler

import (
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
	ProductID  string `json:"product_id" binding:"required"`
	CustomerNo string `json:"customer_no" binding:"required,min=8,max=20"`
}

func (h *TopupHandler) Topup(c *gin.Context) {
	agentID := c.GetInt64("agent_id")
	var req topupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	t, err := h.topupSvc.CreatePending(c.Request.Context(), agentID, req.ProductID, req.CustomerNo)
	if err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	if !h.pool.Enqueue(worker.Job{TxID: t.ID}) {
		util.Log.Warnf("[TOPUP] queue full, tx=%s pending", t.ID)
		util.TooManyRequests(c, "server busy, transaction queued but delayed. retry later")
		return
	}

	util.Accepted(c, "transaction queued", gin.H{
		"tx_id":  t.ID,
		"status": "PENDING",
	})
}
