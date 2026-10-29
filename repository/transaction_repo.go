package repository

import (
	"context"
	"database/sql"

	"pulsa-backend/model"
)

type TransactionRepository struct{ db *sql.DB }

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, tx *sql.Tx, t *model.Transaction) error {
	q := `INSERT INTO transactions (id, agent_id, product_id, customer_no, amount, sell_price, markup, status)
	      VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, q, t.ID, t.AgentID, t.ProductID, t.CustomerNo, t.Amount, t.SellPrice, t.Markup, t.Status)
	return err
}

func (r *TransactionRepository) UpdateStatus(ctx context.Context, tx *sql.Tx, id, status, reason string) error {
	q := `UPDATE transactions SET status = ?, failure_reason = ? WHERE id = ?`
	_, err := tx.ExecContext(ctx, q, status, reason, id)
	return err
}

func (r *TransactionRepository) GetByID(ctx context.Context, id string) (*model.Transaction, error) {
	q := `SELECT id, agent_id, product_id, customer_no, amount, sell_price, markup, status,
	         COALESCE(failure_reason,''), callback_status, callback_attempts, created_at, updated_at
	      FROM transactions WHERE id = ? LIMIT 1`
	var t model.Transaction
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&t.ID, &t.AgentID, &t.ProductID, &t.CustomerNo, &t.Amount, &t.SellPrice, &t.Markup, &t.Status,
		&t.FailureReason, &t.CallbackStatus, &t.CallbackAttempts, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &t, err
}

func (r *TransactionRepository) GetByIDForUpdate(ctx context.Context, tx *sql.Tx, id string) (*model.Transaction, error) {
	q := `SELECT id, agent_id, product_id, customer_no, amount, sell_price, markup, status,
	         COALESCE(failure_reason,''), callback_status, callback_attempts, created_at, updated_at
	      FROM transactions WHERE id = ? LIMIT 1 FOR UPDATE`
	var t model.Transaction
	err := tx.QueryRowContext(ctx, q, id).Scan(
		&t.ID, &t.AgentID, &t.ProductID, &t.CustomerNo, &t.Amount, &t.SellPrice, &t.Markup, &t.Status,
		&t.FailureReason, &t.CallbackStatus, &t.CallbackAttempts, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &t, err
}

func (r *TransactionRepository) ListPendingIDs(ctx context.Context, limit int) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id FROM transactions WHERE status = 'PENDING' ORDER BY created_at ASC, id ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// ListByAgent — untuk riwayat pemakaian saldo agent
func (r *TransactionRepository) ListByAgent(ctx context.Context, agentID int64, limit, offset int) ([]model.Transaction, error) {
	q := `SELECT id, agent_id, product_id, customer_no, amount, sell_price, markup, status,
	         COALESCE(failure_reason,''), callback_status, callback_attempts, created_at, updated_at
	      FROM transactions WHERE agent_id = ?
	      ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, agentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.AgentID, &t.ProductID, &t.CustomerNo, &t.Amount, &t.SellPrice, &t.Markup, &t.Status,
			&t.FailureReason, &t.CallbackStatus, &t.CallbackAttempts, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, nil
}

// ListAll — untuk admin report
func (r *TransactionRepository) ListAll(ctx context.Context, status string, limit, offset int) ([]model.Transaction, error) {
	base := `SELECT id, agent_id, product_id, customer_no, amount, sell_price, markup, status,
	         COALESCE(failure_reason,''), callback_status, callback_attempts, created_at, updated_at
	      FROM transactions`
	args := []interface{}{}

	if status != "" {
		base += " WHERE status = ?"
		args = append(args, status)
	}
	base += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.AgentID, &t.ProductID, &t.CustomerNo, &t.Amount, &t.SellPrice, &t.Markup, &t.Status,
			&t.FailureReason, &t.CallbackStatus, &t.CallbackAttempts, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, nil
}

func (r *TransactionRepository) MarkCallback(ctx context.Context, txID string, status string, attempts int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE transactions SET callback_status = ?, callback_attempts = ? WHERE id = ?`,
		status, attempts, txID)
	return err
}
