package handler

import (
	"errors"
	"strconv"
	"strings"

	"pulsa-backend/repository"
	"pulsa-backend/service"
	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DepositRequestHandler struct {
	depositService *service.DepositRequestService
	requestRepo    *repository.DepositRequestRepository
}

func NewDepositRequestHandler(ds *service.DepositRequestService,
	rr *repository.DepositRequestRepository) *DepositRequestHandler {
	return &DepositRequestHandler{depositService: ds, requestRepo: rr}
}

func (h *DepositRequestHandler) Create(c *gin.Context) {
	var req struct {
		Amount    float64 `json:"amount" binding:"required,gt=0"`
		Reference string  `json:"reference" binding:"required,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}

	reference := strings.TrimSpace(req.Reference)
	if reference == "" {
		util.BadRequest(c, "reference is required")
		return
	}
	request, err := h.depositService.Create(c.Request.Context(), c.GetInt64("agent_id"), req.Amount, reference)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidDepositAmount):
			util.BadRequest(c, err.Error())
		case errors.Is(err, service.ErrInvalidDepositReference):
			util.BadRequest(c, err.Error())
		case errors.Is(err, service.ErrAgentInactive):
			util.Forbidden(c, err.Error())
		case errors.Is(err, repository.ErrNotFound):
			util.NotFound(c, "agent not found")
		case errors.Is(err, repository.ErrDuplicateDepositReference):
			util.Conflict(c, err.Error())
		default:
			util.Log.Errorf("[DEPOSIT] request create error agent=%d: %v", c.GetInt64("agent_id"), err)
			util.InternalError(c, "failed to create deposit request")
		}
		return
	}
	util.Created(c, "deposit request submitted", request)
}

func (h *DepositRequestHandler) ListByAgent(c *gin.Context) {
	status := c.Query("status")
	if !validDepositRequestStatus(status) {
		util.BadRequest(c, "status must be PENDING, APPROVED, or REJECTED")
		return
	}
	limit, offset, ok := parsePagination(c, 20)
	if !ok {
		return
	}

	requests, err := h.requestRepo.ListByAgent(c.Request.Context(), c.GetInt64("agent_id"), status, limit, offset)
	if err != nil {
		util.Log.Errorf("[DEPOSIT] request history error agent=%d: %v", c.GetInt64("agent_id"), err)
		util.InternalError(c, "failed to load deposit requests")
		return
	}
	util.OK(c, "deposit requests", requests)
}

func (h *DepositRequestHandler) ListAll(c *gin.Context) {
	status := c.Query("status")
	if !validDepositRequestStatus(status) {
		util.BadRequest(c, "status must be PENDING, APPROVED, or REJECTED")
		return
	}
	limit, offset, ok := parsePagination(c, 50)
	if !ok {
		return
	}

	requests, err := h.requestRepo.ListAll(c.Request.Context(), status, limit, offset)
	if err != nil {
		util.Log.Errorf("[ADMIN] deposit request list error: %v", err)
		util.InternalError(c, "failed to list deposit requests")
		return
	}
	util.OK(c, "deposit requests", requests)
}

func (h *DepositRequestHandler) Approve(c *gin.Context) {
	h.decide(c, true)
}

func (h *DepositRequestHandler) Reject(c *gin.Context) {
	h.decide(c, false)
}

func (h *DepositRequestHandler) decide(c *gin.Context, approve bool) {
	requestID := c.Param("id")
	if _, err := uuid.Parse(requestID); err != nil {
		util.BadRequest(c, "invalid deposit request id")
		return
	}
	var req struct {
		Note string `json:"note" binding:"max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, "request body must contain an optional note")
		return
	}

	var data interface{}
	var err error
	if approve {
		data, err = h.depositService.Approve(c.Request.Context(), c.GetInt64("admin_id"), requestID, req.Note)
	} else {
		data, err = h.depositService.Reject(c.Request.Context(), c.GetInt64("admin_id"), requestID, req.Note)
	}
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDepositAlreadyDecided):
			util.Conflict(c, err.Error())
		case errors.Is(err, repository.ErrNotFound):
			util.NotFound(c, "deposit request not found")
		default:
			util.Log.Errorf("[ADMIN] deposit decision failed request=%s: %v", requestID, err)
			util.InternalError(c, "failed to process deposit request")
		}
		return
	}

	if approve {
		util.OK(c, "deposit request approved", data)
		return
	}
	util.OK(c, "deposit request rejected", data)
}

func validDepositRequestStatus(status string) bool {
	return status == "" || status == "PENDING" || status == "APPROVED" || status == "REJECTED"
}

func parsePagination(c *gin.Context, defaultLimit int) (int, int, bool) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	if err != nil || limit <= 0 {
		util.BadRequest(c, "limit must be a positive integer")
		return 0, 0, false
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		util.BadRequest(c, "offset must be a non-negative integer")
		return 0, 0, false
	}
	if limit > 100 {
		limit = 100
	}
	return limit, offset, true
}
