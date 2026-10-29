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

var (
	ErrProductNotFound = errors.New("product not found")
	ErrInsufficient    = errors.New("insufficient balance")
)

type TopupService struct {
	db          *sql.DB
	agentRepo   *repository.AgentRepository
	productRepo *repository.ProductRepository
	txRepo      *repository.TransactionRepository
	ledgerRepo  *repository.LedgerRepository
	cbService   *CallbackService
}

func NewTopupService(db *sql.DB, ar *repository.AgentRepository, pr *repository.ProductRepository,
	tr *repository.TransactionRepository, lr *repository.LedgerRepository, cb *CallbackService) *TopupService {
	return &TopupService{db: db, agentRepo: ar, productRepo: pr, txRepo: tr, ledgerRepo: lr, cbService: cb}
}

// CreatePending — dipanggil handler (cepat), hanya insert ke DB
func (s *TopupService) CreatePending(ctx context.Context, agentID int64, productID, customerNo string) (*model.Transaction, error) {
	prod, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, ErrProductNotFound
	}

	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if agent.Balance < prod.Price {
		return nil, ErrInsufficient
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	t := &model.Transaction{
		ID:         uuid.NewString(),
		AgentID:    agentID,
		ProductID:  productID,
		CustomerNo: customerNo,
		Amount:     prod.Price,
		Status:     "PENDING",
	}
	if err := s.txRepo.Create(ctx, tx, t); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return t, nil
}

// Process — dipanggil worker; potong saldo atomik + callback
func (s *TopupService) Process(ctx context.Context, txID string) {
	t, err := s.txRepo.GetByID(ctx, txID)
	if err != nil {
		util.Log.Errorf("[TOPUP] cannot load tx=%s: %v", txID, err)
		return
	}
	if t.Status != "PENDING" {
		util.Log.Warnf("[TOPUP] tx=%s already processed status=%s", txID, t.Status)
		return
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		util.Log.Errorf("[TOPUP] begin tx error: %v", err)
		return
	}
	defer tx.Rollback()

	// 1. Lock & baca saldo (SELECT FOR UPDATE)
	before, err := s.agentRepo.GetBalanceForUpdate(ctx, tx, t.AgentID)
	if err != nil {
		util.Log.Errorf("[TOPUP] lock error tx=%s: %v", txID, err)
		return
	}

	// 2. Cek saldo
	if before < t.Amount {
		_ = s.txRepo.UpdateStatus(ctx, tx, t.ID, "FAILED", "insufficient balance")
		_ = tx.Commit()
		s.cbService.Send(t.AgentID, CallbackPayload{
			TxID: t.ID, Status: "FAILED", Reason: "insufficient balance",
			CustomerNo: t.CustomerNo, ProductID: t.ProductID, Amount: t.Amount, Timestamp: time.Now(),
		})
		return
	}

	// 3. Potong saldo
	if err := s.agentRepo.DeductBalance(ctx, tx, t.AgentID, t.Amount); err != nil {
		util.Log.Errorf("[TOPUP] deduct error tx=%s: %v", txID, err)
		return
	}

	// 4. Update status
	if err := s.txRepo.UpdateStatus(ctx, tx, t.ID, "SUCCESS", ""); err != nil {
		util.Log.Errorf("[TOPUP] update status error tx=%s: %v", txID, err)
		return
	}

	// 5. Ledger
	after := before - t.Amount
	if err := s.ledgerRepo.Insert(ctx, tx, t.AgentID, "TRANSACTION", t.ID,
		-t.Amount, before, after, fmt.Sprintf("Topup %s to %s", t.ProductID, t.CustomerNo)); err != nil {
		util.Log.Errorf("[TOPUP] ledger error tx=%s: %v", txID, err)
		return
	}

	// 6. Commit
	if err := tx.Commit(); err != nil {
		util.Log.Errorf("[TOPUP] commit error tx=%s: %v", txID, err)
		return
	}

	util.Log.Infof("[TOPUP] SUCCESS tx=%s agent=%d amount=%.2f", txID, t.AgentID, t.Amount)

	// 7. Callback async
	s.cbService.Send(t.AgentID, CallbackPayload{
		TxID: t.ID, Status: "SUCCESS", CustomerNo: t.CustomerNo,
		ProductID: t.ProductID, Amount: t.Amount, Timestamp: time.Now(),
	})
}
