package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"pulsa-backend/model"
)

var ErrNotFound = errors.New("record not found")

type AgentRepository struct {
	db *sql.DB
}

func NewAgentRepository(db *sql.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

func (r *AgentRepository) FindByUsername(ctx context.Context, username string) (*model.Agent, error) {
	q := `SELECT id, username, password_hash, name, COALESCE(email,''), COALESCE(phone,''),
	         balance, COALESCE(callback_url,''), is_active, created_at
	      FROM agents WHERE username = ? LIMIT 1`
	var a model.Agent
	err := r.db.QueryRowContext(ctx, q, username).Scan(
		&a.ID, &a.Username, &a.PasswordHash, &a.Name, &a.Email, &a.Phone,
		&a.Balance, &a.CallbackURL, &a.IsActive, &a.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &a, err
}

func (r *AgentRepository) FindByID(ctx context.Context, id int64) (*model.Agent, error) {
	q := `SELECT id, username, password_hash, name, COALESCE(email,''), COALESCE(phone,''),
	         balance, COALESCE(callback_url,''), is_active, created_at
	      FROM agents WHERE id = ? LIMIT 1`
	var a model.Agent
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&a.ID, &a.Username, &a.PasswordHash, &a.Name, &a.Email, &a.Phone,
		&a.Balance, &a.CallbackURL, &a.IsActive, &a.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &a, err
}

// GetBalanceForUpdate — HARUS dipanggil dalam DB transaction
func (r *AgentRepository) GetBalanceForUpdate(ctx context.Context, tx *sql.Tx, agentID int64) (float64, error) {
	var balance float64
	err := tx.QueryRowContext(ctx, `SELECT balance FROM agents WHERE id = ? FOR UPDATE`, agentID).Scan(&balance)
	return balance, err
}

func (r *AgentRepository) DeductBalance(ctx context.Context, tx *sql.Tx, agentID int64, amount float64) error {
	res, err := tx.ExecContext(ctx, `UPDATE agents SET balance = balance - ? WHERE id = ? AND balance >= ?`,
		amount, agentID, amount)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("insufficient balance")
	}
	return nil
}

func (r *AgentRepository) AddBalance(ctx context.Context, tx *sql.Tx, agentID int64, amount float64) error {
	_, err := tx.ExecContext(ctx, `UPDATE agents SET balance = balance + ? WHERE id = ?`, amount, agentID)
	return err
}

// ListAgents — untuk admin dashboard
func (r *AgentRepository) List(ctx context.Context, limit, offset int) ([]model.Agent, error) {
	q := `SELECT id, username, name, COALESCE(email,''), COALESCE(phone,''),
	         balance, is_active, created_at
	      FROM agents ORDER BY id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []model.Agent
	for rows.Next() {
		var a model.Agent
		if err := rows.Scan(&a.ID, &a.Username, &a.Name, &a.Email, &a.Phone,
			&a.Balance, &a.IsActive, &a.CreatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// GetBalance — ambil saldo terbaru + info dasar agent (untuk endpoint /api/balance)
func (r *AgentRepository) GetBalance(ctx context.Context, agentID int64) (map[string]interface{}, error) {
	q := `SELECT id, username, name, balance, is_active, updated_at
	      FROM agents WHERE id = ? LIMIT 1`
	var (
		id        int64
		username  string
		name      string
		balance   float64
		isActive  bool
		updatedAt time.Time
	)
	err := r.db.QueryRowContext(ctx, q, agentID).Scan(&id, &username, &name, &balance, &isActive, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"agent_id":   id,
		"username":   username,
		"name":       name,
		"balance":    balance,
		"is_active":  isActive,
		"updated_at": updatedAt,
	}, nil
}
