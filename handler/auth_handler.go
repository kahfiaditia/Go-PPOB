package handler

import (
	"errors"

	"pulsa-backend/service"
	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct{ authSvc *service.AuthService }

func NewAuthHandler(s *service.AuthService) *AuthHandler { return &AuthHandler{authSvc: s} }

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func (h *AuthHandler) AgentLogin(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}
	token, err := h.authSvc.LoginAgent(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.writeLoginError(c, err)
		return
	}
	util.OK(c, "login success", gin.H{"token": token})
}

func (h *AuthHandler) AdminLogin(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, err.Error())
		return
	}
	token, err := h.authSvc.LoginAdmin(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.writeLoginError(c, err)
		return
	}
	util.OK(c, "login success", gin.H{"token": token})
}

func (h *AuthHandler) writeLoginError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		util.Unauthorized(c, "invalid credentials")
	case errors.Is(err, service.ErrAuthInactive):
		util.Forbidden(c, "account is inactive")
	default:
		util.Log.Errorf("[AUTH] login error: %v", err)
		util.InternalError(c, "login failed")
	}
}
