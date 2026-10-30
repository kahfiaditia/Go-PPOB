package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"pulsa-backend/model"
	"pulsa-backend/repository"
	"pulsa-backend/util"

	"github.com/google/uuid"
)

var (
	ErrInvalidDepositAmount    = errors.New("deposit amount must be greater than zero")
	ErrInvalidDepositReference = errors.New("deposit transfer reference is required")
	ErrDepositAlreadyDecided   = errors.New("deposit request has already been decided")
)

type DepositRequestService struct {
	db          *sql.DB
	agentRepo   *repository.AgentRepository
	requestRepo *repository.DepositRequestRepository
	depositRepo *repository.DepositRepository
	ledgerRepo  *repository.LedgerRepository
}

func NewDepositRequestService(db *sql.DB, agentRepo *repository.AgentRepository,
	requestRepo *repository.DepositRequestRepository, depositRepo *repository.DepositRepository,
	ledgerRepo *repository.LedgerRepository) *DepositRequestService {
	return &DepositRequestService{
		db: db, agentRepo: agentRepo, requestRepo: requestRepo,
		depositRepo: depositRepo, ledgerRepo: ledgerRepo,
	}
}

func (s *DepositRequestService) Create(ctx context.Context, agentID int64,
	amount float64, reference string) (*model.DepositRequest, error) {
	if amount <= 0 {
		return nil, ErrInvalidDepositAmount
	}
	reference = strings.TrimSpace(reference)
	if reference == "" || len(reference) > 100 {
		return nil, ErrInvalidDepositReference
	}
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if !agent.IsActive {
		return nil, ErrAgentInactive
	}

	request := &model.DepositRequest{
		ID: uuid.NewString(), AgentID: agentID, Amount: amount,
		Reference: reference, Status: "PENDING", CreatedAt: time.Now(),
	}
	if err := s.requestRepo.Create(ctx, request); err != nil {
		return nil, err
	}
	return request, nil
}

func (s *DepositRequestService) Approve(ctx context.Context, adminID int64,
	requestID, note string) (*model.Deposit, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	request, err := s.requestRepo.GetByIDForUpdate(ctx, tx, requestID)
	if err != nil {
		return nil, err
	}
	if request.Status != "PENDING" {
		return nil, ErrDepositAlreadyDecided
	}
	before, err := s.agentRepo.GetBalanceForUpdate(ctx, tx, request.AgentID)
	if err != nil {
		return nil, err
	}
	if err := s.agentRepo.AddBalance(ctx, tx, request.AgentID, request.Amount); err != nil {
		return nil, err
	}

	deposit := &model.Deposit{
		ID: uuid.NewString(), RequestID: request.ID, AgentID: request.AgentID, AdminID: adminID,
		Amount: request.Amount, BalanceBefore: before, BalanceAfter: before + request.Amount,
		Reference: request.Reference, Note: note, CreatedAt: time.Now(),
	}
	if err := s.depositRepo.Create(ctx, tx, deposit); err != nil {
		return nil, err
	}
	if err := s.ledgerRepo.Insert(ctx, tx, request.AgentID, "DEPOSIT", deposit.ID,
		request.Amount, before, deposit.BalanceAfter,
		fmt.Sprintf("Deposit request %s approved by admin %d", request.ID, adminID)); err != nil {
		return nil, err
	}
	if err := s.requestRepo.UpdateDecision(ctx, tx, request.ID, adminID, "APPROVED", note); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	request.Status = "APPROVED"
	request.AdminID = adminID
	request.AdminNote = note
	processedAt := time.Now()
	request.ProcessedAt = &processedAt
	util.Log.Infof("[DEPOSIT] request=%s admin=%d agent=%d amount=%.2f approved",
		request.ID, adminID, request.AgentID, request.Amount)
	return deposit, nil
}

func (s *DepositRequestService) Reject(ctx context.Context, adminID int64,
	requestID, note string) (*model.DepositRequest, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	request, err := s.requestRepo.GetByIDForUpdate(ctx, tx, requestID)
	if err != nil {
		return nil, err
	}
	if request.Status != "PENDING" {
		return nil, ErrDepositAlreadyDecided
	}
	if err := s.requestRepo.UpdateDecision(ctx, tx, request.ID, adminID, "REJECTED", note); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	request.Status = "REJECTED"
	request.AdminID = adminID
	request.AdminNote = note
	processedAt := time.Now()
	request.ProcessedAt = &processedAt
	return request, nil
}
