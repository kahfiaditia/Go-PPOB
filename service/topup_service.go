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
	ErrProductNotFound     = errors.New("product not found")
	ErrInsufficient        = errors.New("insufficient balance")
	ErrAgentInactive       = errors.New("agent is inactive")
	ErrPINNotSet           = errors.New("transaction PIN is not set")
	ErrPINIncorrect        = errors.New("transaction PIN is incorrect")
	ErrPINLocked           = errors.New("transaction PIN is temporarily locked")
	ErrInvalidProfit       = errors.New("agent profit must be between 500 and 2000")
	ErrInvalidProductPrice = errors.New("invalid product price configuration")
)

type TopupService struct {
	db          *sql.DB
	agentRepo   *repository.AgentRepository
	productRepo *repository.ProductRepository
	pricingRepo *repository.AgentProductPricingRepository
	txRepo      *repository.TransactionRepository
	ledgerRepo  *repository.LedgerRepository
	cbService   *CallbackService
}

func NewTopupService(db *sql.DB, ar *repository.AgentRepository, pr *repository.ProductRepository,
	ppr *repository.AgentProductPricingRepository, tr *repository.TransactionRepository,
	lr *repository.LedgerRepository, cb *CallbackService) *TopupService {
	return &TopupService{
		db: db, agentRepo: ar, productRepo: pr, pricingRepo: ppr,
		txRepo: tr, ledgerRepo: lr, cbService: cb,
	}
}

// CreatePending — dipanggil handler (cepat), hanya insert ke DB
func (s *TopupService) CreatePending(ctx context.Context, agentID int64, productID, customerNo, pin string, requestedProfit *float64) (*model.Transaction, error) {
	prod, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}

	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if !agent.IsActive {
		return nil, ErrAgentInactive
	}

	if err := s.agentRepo.VerifyPIN(ctx, agentID, pin); err != nil {
		switch {
		case errors.Is(err, repository.ErrPINNotSet):
			return nil, ErrPINNotSet
		case errors.Is(err, repository.ErrInvalidPIN):
			return nil, ErrPINIncorrect
		case errors.Is(err, repository.ErrPINLocked):
			return nil, ErrPINLocked
		default:
			return nil, err
		}
	}

	defaultProfit, err := s.pricingRepo.GetDefaultProfit(ctx, agentID, productID)
	if err != nil {
		return nil, err
	}
	sellPrice, profit, err := calculateTopupPrice(prod.HargaJual, defaultProfit, requestedProfit)
	if err != nil {
		return nil, err
	}
	if agent.Balance < sellPrice {
		return nil, ErrInsufficient
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	t := &model.Transaction{
		ID:            uuid.NewString(),
		AgentID:       agentID,
		ProductID:     productID,
		CustomerNo:    customerNo,
		Amount:        prod.Nominal,
		SellPrice:     sellPrice,
		Markup:        profit,
		CustomerPrice: sellPrice + profit,
		Status:        "PENDING",
	}
	if err := s.txRepo.Create(ctx, tx, t); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return t, nil
}

func calculateTopupPrice(distributorSellPrice, defaultProfit float64, requestedProfit *float64) (sellPrice, profit float64, err error) {
	if distributorSellPrice <= 0 {
		return 0, 0, ErrInvalidProductPrice
	}
	if defaultProfit < model.MinAgentProfit || defaultProfit > model.MaxAgentProfit {
		return 0, 0, ErrInvalidProfit
	}
	if requestedProfit == nil {
		return distributorSellPrice, defaultProfit, nil
	}
	if *requestedProfit < model.MinAgentProfit || *requestedProfit > model.MaxAgentProfit {
		return 0, 0, ErrInvalidProfit
	}
	return distributorSellPrice, *requestedProfit, nil
}

func (s *TopupService) PendingTransactionIDs(ctx context.Context, limit int) ([]string, error) {
	return s.txRepo.ListPendingIDs(ctx, limit)
}

// Process — dipanggil worker; potong saldo atomik + callback
func (s *TopupService) Process(ctx context.Context, txID string) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		util.Log.Errorf("[TOPUP] begin tx error: %v", err)
		return
	}
	defer tx.Rollback()

	t, err := s.txRepo.GetByIDForUpdate(ctx, tx, txID)
	if err != nil {
		util.Log.Errorf("[TOPUP] cannot lock tx=%s: %v", txID, err)
		return
	}
	if t.Status != "PENDING" {
		util.Log.Warnf("[TOPUP] tx=%s already processed status=%s", txID, t.Status)
		return
	}

	// Lock and read the balance in the same transaction as the transaction status.
	before, err := s.agentRepo.GetBalanceForUpdate(ctx, tx, t.AgentID)
	if err != nil {
		util.Log.Errorf("[TOPUP] lock error tx=%s: %v", txID, err)
		return
	}

	// 2. Cek saldo
	if before < t.SellPrice {
		if err := s.txRepo.UpdateStatus(ctx, tx, t.ID, "FAILED", "insufficient balance"); err != nil {
			util.Log.Errorf("[TOPUP] mark failed error tx=%s: %v", txID, err)
			return
		}
		if err := tx.Commit(); err != nil {
			util.Log.Errorf("[TOPUP] commit failed status error tx=%s: %v", txID, err)
			return
		}
		s.cbService.Send(t.AgentID, CallbackPayload{
			TxID: t.ID, Status: "FAILED", Reason: "insufficient balance",
			CustomerNo: t.CustomerNo, ProductID: t.ProductID, Amount: t.Amount, Timestamp: time.Now(),
		})
		return
	}

	// 3. Potong saldo
	if err := s.agentRepo.DeductBalance(ctx, tx, t.AgentID, t.SellPrice); err != nil {
		util.Log.Errorf("[TOPUP] deduct error tx=%s: %v", txID, err)
		return
	}

	// 4. Update status
	if err := s.txRepo.UpdateStatus(ctx, tx, t.ID, "SUCCESS", ""); err != nil {
		util.Log.Errorf("[TOPUP] update status error tx=%s: %v", txID, err)
		return
	}

	// 5. Ledger
	after := before - t.SellPrice
	if err := s.ledgerRepo.Insert(ctx, tx, t.AgentID, "TRANSACTION", t.ID,
		-t.SellPrice, before, after, fmt.Sprintf("Topup %s to %s", t.ProductID, t.CustomerNo)); err != nil {
		util.Log.Errorf("[TOPUP] ledger error tx=%s: %v", txID, err)
		return
	}

	// 6. Commit
	if err := tx.Commit(); err != nil {
		util.Log.Errorf("[TOPUP] commit error tx=%s: %v", txID, err)
		return
	}

	util.Log.Infof("[TOPUP] SUCCESS tx=%s agent=%d amount=%.2f", txID, t.AgentID, t.SellPrice)

	// 7. Callback async
	s.cbService.Send(t.AgentID, CallbackPayload{
		TxID: t.ID, Status: "SUCCESS", CustomerNo: t.CustomerNo,
		ProductID: t.ProductID, Amount: t.Amount, Timestamp: time.Now(),
	})
}
