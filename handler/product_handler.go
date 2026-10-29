package handler

import (
	"pulsa-backend/model"
	"pulsa-backend/repository"
	"pulsa-backend/util"
	"time"

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
	response := make([]agentProductResponse, 0, len(products))
	for _, product := range products {
		response = append(response, toAgentProductResponse(product))
	}
	util.OK(c, "products", response)
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
	util.OK(c, "product detail", toAgentProductResponse(*p))
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
	response := make([]adminProductResponse, 0, len(products))
	for _, product := range products {
		response = append(response, toAdminProductResponse(product))
	}
	util.OK(c, "all products", response)
}

type agentProductResponse struct {
	ID        string    `json:"id"`
	Operator  string    `json:"operator"`
	Nominal   float64   `json:"nominal"`
	Name      string    `json:"name"`
	HargaJual float64   `json:"harga_jual"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type adminProductResponse struct {
	ID        string    `json:"id"`
	Operator  string    `json:"operator"`
	Nominal   float64   `json:"nominal"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	HargaJual float64   `json:"harga_jual"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

func toAgentProductResponse(p model.Product) agentProductResponse {
	return agentProductResponse{
		ID: p.ID, Operator: p.Operator, Nominal: p.Nominal, Name: p.Name,
		HargaJual: p.HargaJual, IsActive: p.IsActive, CreatedAt: p.CreatedAt,
	}
}

func toAdminProductResponse(p model.Product) adminProductResponse {
	return adminProductResponse{
		ID: p.ID, Operator: p.Operator, Nominal: p.Nominal, Name: p.Name,
		Price: p.Price, HargaJual: p.HargaJual, IsActive: p.IsActive, CreatedAt: p.CreatedAt,
	}
}

// ============================================
// ADMIN — POST /api/admin/products
// ============================================
type createProductReq struct {
	ID        string  `json:"id" binding:"required,min=2,max=20"`
	Operator  string  `json:"operator" binding:"required"`
	Nominal   float64 `json:"nominal" binding:"required,gt=0"`
	Name      string  `json:"name" binding:"required"`
	Price     float64 `json:"price" binding:"required,gt=0"`
	HargaJual float64 `json:"harga_jual" binding:"required,gt=0"`
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req createProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}
	if req.HargaJual < req.Price {
		util.BadRequest(c, "harga_jual must be >= price")
		return
	}
	p := &model.Product{
		ID: req.ID, Operator: req.Operator, Nominal: req.Nominal, Name: req.Name,
		Price: req.Price, HargaJual: req.HargaJual, IsActive: true,
	}
	if err := h.productRepo.Create(c.Request.Context(), p); err != nil {
		util.Log.Errorf("[PRODUCT] create error: %v", err)
		util.BadRequest(c, "failed to create product (maybe duplicate id)")
		return
	}
	util.Created(c, "product created", toAdminProductResponse(*p))
}

// ============================================
// ADMIN — PATCH /api/admin/products/:id
// ============================================
type updateProductReq struct {
	Operator  *string  `json:"operator"`
	Nominal   *float64 `json:"nominal"`
	Name      *string  `json:"name"`
	Price     *float64 `json:"price"`
	HargaJual *float64 `json:"harga_jual"`
	IsActive  *bool    `json:"is_active"`
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
	if req.Nominal != nil {
		if *req.Nominal <= 0 {
			util.BadRequest(c, "nominal must be > 0")
			return
		}
		p.Nominal = *req.Nominal
	}
	if req.Price != nil {
		if *req.Price <= 0 {
			util.BadRequest(c, "price must be > 0")
			return
		}
		p.Price = *req.Price
	}
	if req.HargaJual != nil {
		if *req.HargaJual <= 0 {
			util.BadRequest(c, "harga_jual must be > 0")
			return
		}
		p.HargaJual = *req.HargaJual
	}
	if p.HargaJual < p.Price {
		util.BadRequest(c, "harga_jual must be >= price")
		return
	}
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}

	if err := h.productRepo.Update(c.Request.Context(), p); err != nil {
		util.InternalError(c, "failed to update product")
		return
	}
	util.OK(c, "product updated", toAdminProductResponse(*p))
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
