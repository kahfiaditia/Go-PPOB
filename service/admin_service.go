package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"pulsa-backend/model"
	"pulsa-backend/repository"
	"pulsa-backend/util"

	"github.com/google/uuid"
)

type AdminService struct {
	db          *sql.DB
	agentRepo   *repository.AgentRepository
	depositRepo *repository.DepositRepository
	ledgerRepo  *repository.LedgerRepository
}

func NewAdminService(db *sql.DB, ar *repository.AgentRepository,
	dr *repository.DepositRepository, lr *repository.LedgerRepository) *AdminService {
	return &AdminService{db: db, agentRepo: ar, depositRepo: dr, ledgerRepo: lr}
}

// Deposit — admin menambah saldo agent (Top-up Deposit)
func (s *AdminService) Deposit(ctx context.Context, adminID, agentID int64,
	amount float64, reference, note string) (*model.Deposit, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be > 0")
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	before, err := s.agentRepo.GetBalanceForUpdate(ctx, tx, agentID)
	if err != nil {
		return nil, err
	}

	if err := s.agentRepo.AddBalance(ctx, tx, agentID, amount); err != nil {
		return nil, err
	}

	after := before + amount
	d := &model.Deposit{
		ID: uuid.NewString(), AgentID: agentID, AdminID: adminID,
		Amount: amount, BalanceBefore: before, BalanceAfter: after,
		Reference: reference, Note: note, CreatedAt: time.Now(),
	}

	if err := s.depositRepo.Create(ctx, tx, d); err != nil {
		return nil, err
	}
	if err := s.ledgerRepo.Insert(ctx, tx, agentID, "DEPOSIT", d.ID,
		amount, before, after, fmt.Sprintf("Deposit by admin %d", adminID)); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	util.Log.Infof("[DEPOSIT] admin=%d agent=%d amount=%.2f ref=%s", adminID, agentID, amount, reference)
	return d, nil
}

// AdjustBalance — koreksi saldo (mis. untuk refund manual)
func (s *AdminService) AdjustBalance(ctx context.Context, adminID, agentID int64,
	amount float64, note string) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	before, err := s.agentRepo.GetBalanceForUpdate(ctx, tx, agentID)
	if err != nil {
		return err
	}
	if before+amount < 0 {
		return errors.New("resulting balance would be negative")
	}

	if amount >= 0 {
		if err := s.agentRepo.AddBalance(ctx, tx, agentID, amount); err != nil {
			return err
		}
	} else {
		if err := s.agentRepo.DeductBalance(ctx, tx, agentID, -amount); err != nil {
			return err
		}
	}

	after := before + amount
	if err := s.ledgerRepo.Insert(ctx, tx, agentID, "ADJUSTMENT",
		uuid.NewString(), amount, before, after, fmt.Sprintf("Adjust by admin %d: %s", adminID, note)); err != nil {
		return err
	}
	return tx.Commit()
}
