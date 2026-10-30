package handler

import (
	"errors"

	"pulsa-backend/repository"
	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
)

type PricingHandler struct {
	pricingRepo *repository.AgentProductPricingRepository
	productRepo *repository.ProductRepository
}

func NewPricingHandler(pr *repository.AgentProductPricingRepository, products *repository.ProductRepository) *PricingHandler {
	return &PricingHandler{pricingRepo: pr, productRepo: products}
}

func (h *PricingHandler) List(c *gin.Context) {
	pricing, err := h.pricingRepo.List(c.Request.Context(), c.GetInt64("agent_id"))
	if err != nil {
		util.Log.Errorf("[PRICING] list error: %v", err)
		util.InternalError(c, "failed to load product pricing")
		return
	}
	util.OK(c, "product pricing", pricing)
}

func (h *PricingHandler) Update(c *gin.Context) {
	var req struct {
		DefaultProfit float64 `json:"default_profit" binding:"required,gte=500,lte=2000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, "default_profit must be between Rp500 and Rp2.000")
		return
	}

	productID := c.Param("id")
	if _, err := h.productRepo.FindByID(c.Request.Context(), productID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			util.NotFound(c, "product not found or inactive")
			return
		}
		util.Log.Errorf("[PRICING] product lookup error product=%s: %v", productID, err)
		util.InternalError(c, "failed to load product")
		return
	}

	agentID := c.GetInt64("agent_id")
	if err := h.pricingRepo.SetDefaultProfit(c.Request.Context(), agentID, productID, req.DefaultProfit); err != nil {
		util.Log.Errorf("[PRICING] update error agent=%d product=%s: %v", agentID, productID, err)
		util.InternalError(c, "failed to update product pricing")
		return
	}

	util.OK(c, "default profit updated", gin.H{
		"product_id":     productID,
		"default_profit": req.DefaultProfit,
	})
}
