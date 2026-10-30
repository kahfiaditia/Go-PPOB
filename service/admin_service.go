package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pulsa-backend/repository"

	"github.com/google/uuid"
)

type AdminService struct {
	db         *sql.DB
	agentRepo  *repository.AgentRepository
	ledgerRepo *repository.LedgerRepository
}

func NewAdminService(db *sql.DB, ar *repository.AgentRepository, lr *repository.LedgerRepository) *AdminService {
	return &AdminService{db: db, agentRepo: ar, ledgerRepo: lr}
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
