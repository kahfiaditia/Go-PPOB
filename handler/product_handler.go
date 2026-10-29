package handler

import (
	"pulsa-backend/model"
	"pulsa-backend/repository"
	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	productRepo *repository.ProductRepository
}

// Fungsi ini yang dicari oleh main.go
func NewProductHandler(pr *repository.ProductRepository) *ProductHandler {
	return &ProductHandler{productRepo: pr}
}

// ============================================
// AGENT — GET /api/products
// ============================================
func (h *ProductHandler) ListForAgent(c *gin.Context) {
	operator := c.Query("operator")
	products, err := h.productRepo.ListActive(c.Request.Context(), operator)
	if err != nil {
		util.Log.Errorf("[PRODUCT] list error: %v", err)
		util.InternalError(c, "failed to load products")
		return
	}
	util.OK(c, "products", products)
}

// ============================================
// AGENT — GET /api/products/:id
// ============================================
func (h *ProductHandler) GetForAgent(c *gin.Context) {
	id := c.Param("id")
	p, err := h.productRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		util.NotFound(c, "product not found or inactive")
		return
	}
	util.OK(c, "product detail", p)
}

// ============================================
// ADMIN — GET /api/admin/products
// ============================================
func (h *ProductHandler) ListForAdmin(c *gin.Context) {
	operator := c.Query("operator")
	products, err := h.productRepo.ListAll(c.Request.Context(), operator)
	if err != nil {
		util.InternalError(c, "failed to load products")
		return
	}
	util.OK(c, "all products", products)
}

// ============================================
// ADMIN — POST /api/admin/products
// ============================================
type createProductReq struct {
	ID       string  `json:"id" binding:"required,min=2,max=20"`
	Operator string  `json:"operator" binding:"required"`
	Name     string  `json:"name" binding:"required"`
	Price    float64 `json:"price" binding:"required,gt=0"`
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req createProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}
	p := &model.Product{
		ID: req.ID, Operator: req.Operator, Name: req.Name,
		Price: req.Price, IsActive: true,
	}
	if err := h.productRepo.Create(c.Request.Context(), p); err != nil {
		util.Log.Errorf("[PRODUCT] create error: %v", err)
		util.BadRequest(c, "failed to create product (maybe duplicate id)")
		return
	}
	util.Created(c, "product created", p)
}

// ============================================
// ADMIN — PATCH /api/admin/products/:id
// ============================================
type updateProductReq struct {
	Operator *string  `json:"operator"`
	Name     *string  `json:"name"`
	Price    *float64 `json:"price"`
	IsActive *bool    `json:"is_active"`
}

func (h *ProductHandler) Update(c *gin.Context) {
	id := c.Param("id")
	p, err := h.productRepo.FindAnyByID(c.Request.Context(), id)
	if err != nil {
		util.NotFound(c, "product not found")
		return
	}

	var req updateProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	if req.Operator != nil {
		p.Operator = *req.Operator
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Price != nil {
		if *req.Price <= 0 {
			util.BadRequest(c, "price must be > 0")
			return
		}
		p.Price = *req.Price
	}
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}

	if err := h.productRepo.Update(c.Request.Context(), p); err != nil {
		util.InternalError(c, "failed to update product")
		return
	}
	util.OK(c, "product updated", p)
}

// ============================================
// ADMIN — DELETE /api/admin/products/:id
// ============================================
func (h *ProductHandler) Deactivate(c *gin.Context) {
	id := c.Param("id")
	if err := h.productRepo.SetActive(c.Request.Context(), id, false); err != nil {
		util.InternalError(c, "failed to deactivate product")
		return
	}
	util.OK(c, "product deactivated", gin.H{"id": id})
}
