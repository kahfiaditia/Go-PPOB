package handler

import (
	"errors"

	"pulsa-backend/repository"
	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type ProfileHandler struct {
	agentRepo *repository.AgentRepository
}

func NewProfileHandler(ar *repository.AgentRepository) *ProfileHandler {
	return &ProfileHandler{agentRepo: ar}
}

type updatePinReq struct {
	CurrentPIN string `json:"current_pin" binding:"required,len=6,numeric"`
	NewPIN     string `json:"new_pin" binding:"required,len=6,numeric"`
}

type updatePasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required,min=6"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// PATCH /api/pin
func (h *ProfileHandler) UpdatePin(c *gin.Context) {
	agentID := c.GetInt64("agent_id")
	var req updatePinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, "current_pin dan new_pin harus masing-masing berupa 6 angka")
		return
	}

	agent, err := h.agentRepo.FindByID(c.Request.Context(), agentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			util.NotFound(c, "agent not found")
			return
		}
		util.InternalError(c, "Gagal memverifikasi PIN")
		return
	}
	if !agent.IsActive {
		util.Forbidden(c, "agent is inactive")
		return
	}
	if err := h.agentRepo.VerifyPIN(c.Request.Context(), agentID, req.CurrentPIN); err != nil {
		switch {
		case errors.Is(err, repository.ErrInvalidPIN):
			util.Forbidden(c, "PIN lama salah")
		case errors.Is(err, repository.ErrPINLocked):
			util.TooManyRequests(c, "PIN transaksi terkunci selama 15 menit")
		case errors.Is(err, repository.ErrPINNotSet):
			util.Forbidden(c, "PIN belum diatur; hubungi superadmin")
		default:
			util.Log.Errorf("[PROFILE] PIN verification error agent=%d: %v", agentID, err)
			util.InternalError(c, "Gagal memverifikasi PIN")
		}
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPIN), bcrypt.DefaultCost)
	if err != nil {
		util.Log.Errorf("[PROFILE] PIN hash error agent=%d: %v", agentID, err)
		util.InternalError(c, "Gagal mengamankan PIN")
		return
	}

	if err := h.agentRepo.UpdatePin(c.Request.Context(), agentID, string(hash)); err != nil {
		util.InternalError(c, "Gagal menyimpan PIN")
		return
	}

	util.OK(c, "PIN berhasil diatur", nil)
}

func (h *ProfileHandler) UpdatePassword(c *gin.Context) {
	agentID := c.GetInt64("agent_id")
	var req updatePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, "current_password dan new_password wajib diisi (minimal 6 karakter)")
		return
	}

	agent, err := h.agentRepo.FindByID(c.Request.Context(), agentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			util.NotFound(c, "agent not found")
			return
		}
		util.Log.Errorf("[PROFILE] password lookup error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to verify password")
		return
	}
	if !agent.IsActive {
		util.Forbidden(c, "agent is inactive")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(agent.PasswordHash), []byte(req.CurrentPassword)) != nil {
		util.Forbidden(c, "current password is incorrect")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		util.Log.Errorf("[PROFILE] password hash error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to secure password")
		return
	}
	if err := h.agentRepo.UpdatePassword(c.Request.Context(), agentID, string(hash)); err != nil {
		util.Log.Errorf("[PROFILE] password update error agent=%d: %v", agentID, err)
		util.InternalError(c, "failed to update password")
		return
	}

	util.OK(c, "password updated", nil)
}
