package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"pulsa-backend/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	agentRepo *repository.AgentRepository
	adminRepo *repository.AdminRepository
	jwtSecret []byte
	jwtExpire time.Duration
}

func NewAuthService(a *repository.AgentRepository, ad *repository.AdminRepository, secret []byte, hours int) *AuthService {
	return &AuthService{
		agentRepo: a,
		adminRepo: ad,
		jwtSecret: secret,
		jwtExpire: time.Duration(hours) * time.Hour,
	}
}

func (s *AuthService) LoginAgent(ctx context.Context, username, password string) (string, error) {
	agent, err := s.agentRepo.FindByUsername(ctx, username)
	if err != nil {
		return "", errors.New("invalid credentials")
	}
	if !agent.IsActive {
		return "", errors.New("agent is inactive")
	}
	if bcrypt.CompareHashAndPassword([]byte(agent.PasswordHash), []byte(password)) != nil {
		return "", errors.New("invalid credentials")
	}
	return s.generateToken(agent.ID, "agent")
}

func (s *AuthService) LoginAdmin(ctx context.Context, username, password string) (string, error) {
	admin, err := s.adminRepo.FindByUsername(ctx, username)
	if err != nil {
		return "", errors.New("invalid credentials")
	}
	if !admin.IsActive {
		return "", errors.New("admin is inactive")
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		return "", errors.New("invalid credentials")
	}
	return s.generateToken(admin.ID, strings.ToLower(admin.Role))
}

func (s *AuthService) generateToken(sub int64, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(s.jwtExpire).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}
