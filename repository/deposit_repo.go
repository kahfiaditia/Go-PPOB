package repository

import (
	"context"
	"database/sql"

	"pulsa-backend/model"
)

type DepositRepository struct{ db *sql.DB }

func NewDepositRepository(db *sql.DB) *DepositRepository {
	return &DepositRepository{db: db}
}

func (r *DepositRepository) Create(ctx context.Context, tx *sql.Tx, d *model.Deposit) error {
	q := `INSERT INTO deposits (id, agent_id, admin_id, amount, balance_before, balance_after, reference, note)
	      VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, q, d.ID, d.AgentID, d.AdminID, d.Amount,
		d.BalanceBefore, d.BalanceAfter, d.Reference, d.Note)
	return err
}

func (r *DepositRepository) ListByAgent(ctx context.Context, agentID int64, limit, offset int) ([]model.Deposit, error) {
	q := `SELECT id, agent_id, admin_id, amount, balance_before, balance_after,
	         COALESCE(reference,''), COALESCE(note,''), created_at
	      FROM deposits WHERE agent_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, agentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ds []model.Deposit
	for rows.Next() {
		var d model.Deposit
		if err := rows.Scan(&d.ID, &d.AgentID, &d.AdminID, &d.Amount, &d.BalanceBefore,
			&d.BalanceAfter, &d.Reference, &d.Note, &d.CreatedAt); err != nil {
			return nil, err
		}
		ds = append(ds, d)
	}
	return ds, nil
}
