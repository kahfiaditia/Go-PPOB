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

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAuthInactive       = errors.New("account is inactive")
)

type AuthService struct {
	agentRepo     *repository.AgentRepository
	agentUserRepo *repository.AgentUserRepository
	adminRepo     *repository.AdminRepository
	jwtSecret     []byte
	jwtExpire     time.Duration
}

func NewAuthService(a *repository.AgentRepository, au *repository.AgentUserRepository,
	ad *repository.AdminRepository, secret []byte, hours int) *AuthService {
	return &AuthService{
		agentRepo: a, agentUserRepo: au, adminRepo: ad,
		jwtSecret: secret, jwtExpire: time.Duration(hours) * time.Hour,
	}
}

func (s *AuthService) LoginAgent(ctx context.Context, email, password string) (string, error) {
	email = strings.TrimSpace(email)
	agent, err := s.agentRepo.FindByEmail(ctx, email)
	if err == nil {
		if !agent.IsActive {
			return "", ErrAuthInactive
		}
		if bcrypt.CompareHashAndPassword([]byte(agent.PasswordHash), []byte(password)) != nil {
			return "", ErrInvalidCredentials
		}
		return s.generateAgentToken(agent.ID, agent.ID, "owner")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return "", err
	}

	employee, agentActive, err := s.agentUserRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}
	if !employee.IsActive || !agentActive {
		return "", ErrAuthInactive
	}
	if bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(password)) != nil {
		return "", ErrInvalidCredentials
	}
	return s.generateAgentToken(employee.ID, employee.AgentID, "employee")
}

func (s *AuthService) LoginAdmin(ctx context.Context, email, password string) (string, error) {
	admin, err := s.adminRepo.FindByEmail(ctx, strings.TrimSpace(email))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}
	if !admin.IsActive {
		return "", ErrAuthInactive
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		return "", ErrInvalidCredentials
	}
	return s.generateToken(admin.ID, strings.ToLower(admin.Role))
}

func (s *AuthService) ValidateAgentSession(ctx context.Context, userID, agentID int64, role string) (bool, error) {
	switch role {
	case "owner":
		if userID != agentID {
			return false, nil
		}
		agent, err := s.agentRepo.FindByID(ctx, agentID)
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return agent.IsActive, nil
	case "employee":
		user, agentActive, err := s.agentUserRepo.FindByID(ctx, agentID, userID)
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return user.IsActive && agentActive, nil
	default:
		return false, nil
	}
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

func (s *AuthService) generateAgentToken(userID, agentID int64, role string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      userID,
		"agent_id": agentID,
		"role":     role,
		"iat":      now.Unix(),
		"exp":      now.Add(s.jwtExpire).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}
