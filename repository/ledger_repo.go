package repository

import (
	"context"
	"database/sql"
)

// LedgerRepository mencatat setiap perubahan saldo untuk audit trail
type LedgerRepository struct{ db *sql.DB }

func NewLedgerRepository(db *sql.DB) *LedgerRepository {
	return &LedgerRepository{db: db}
}

func (r *LedgerRepository) Insert(ctx context.Context, tx *sql.Tx,
	agentID int64, refType, refID string, amount, before, after float64, note string) error {
	q := `INSERT INTO balance_ledger (agent_id, ref_type, ref_id, amount, balance_before, balance_after, note)
	      VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, q, agentID, refType, refID, amount, before, after, note)
	return err
}
