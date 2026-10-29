package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"pulsa-backend/model"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound   = errors.New("record not found")
	ErrPINNotSet  = errors.New("transaction PIN is not set")
	ErrInvalidPIN = errors.New("invalid transaction PIN")
	ErrPINLocked  = errors.New("transaction PIN is temporarily locked")
)

type AgentRepository struct {
	db *sql.DB
}

func NewAgentRepository(db *sql.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

func (r *AgentRepository) FindByEmail(ctx context.Context, email string) (*model.Agent, error) {
	q := `SELECT id, username, password_hash, name, COALESCE(email,''), COALESCE(phone,''),
	         balance, COALESCE(callback_url,''), is_active, created_at
	      FROM agents WHERE email = ? LIMIT 1`
	var a model.Agent
	err := r.db.QueryRowContext(ctx, q, email).Scan(
		&a.ID, &a.Username, &a.PasswordHash, &a.Name, &a.Email, &a.Phone,
		&a.Balance, &a.CallbackURL, &a.IsActive, &a.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &a, err
}

func (r *AgentRepository) FindByID(ctx context.Context, id int64) (*model.Agent, error) {
	q := `SELECT id, username, password_hash, COALESCE(pin_hash, ''), name, COALESCE(email,''), 
	         COALESCE(phone,''), balance, COALESCE(callback_url,''), COALESCE(callback_secret,''), is_active, created_at
	      FROM agents WHERE id = ? LIMIT 1`
	var a model.Agent
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&a.ID, &a.Username, &a.PasswordHash, &a.PinHash, &a.Name, &a.Email, &a.Phone,
		&a.Balance, &a.CallbackURL, &a.CallbackSecret, &a.IsActive, &a.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &a, err
}

func (r *AgentRepository) SetCallbackSecret(ctx context.Context, agentID int64, secret string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE agents SET callback_secret = ? WHERE id = ?`, secret, agentID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		var exists bool
		if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM agents WHERE id = ?)`, agentID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
	}
	return nil
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

func (r *AgentRepository) UpdatePin(ctx context.Context, agentID int64, pinHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE agents SET pin_hash = ?, pin_failed_attempts = 0, pin_locked_until = NULL WHERE id = ?`,
		pinHash, agentID)
	return err
}

func (r *AgentRepository) VerifyPIN(ctx context.Context, agentID int64, pin string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var pinHash string
	var failedAttempts int
	var locked bool
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(pin_hash, ''), pin_failed_attempts,
	        COALESCE(pin_locked_until > CURRENT_TIMESTAMP, 0)
	      FROM agents WHERE id = ? FOR UPDATE`, agentID).Scan(&pinHash, &failedAttempts, &locked)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if locked {
		return ErrPINLocked
	}
	if pinHash == "" {
		return ErrPINNotSet
	}
	if bcrypt.CompareHashAndPassword([]byte(pinHash), []byte(pin)) != nil {
		failedAttempts++
		if failedAttempts >= 5 {
			_, err = tx.ExecContext(ctx, `UPDATE agents SET pin_failed_attempts = 0,
			        pin_locked_until = DATE_ADD(CURRENT_TIMESTAMP, INTERVAL 15 MINUTE) WHERE id = ?`, agentID)
			if err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return err
			}
			return ErrPINLocked
		}
		if _, err := tx.ExecContext(ctx, `UPDATE agents SET pin_failed_attempts = ? WHERE id = ?`,
			failedAttempts, agentID); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrInvalidPIN
	}

	if _, err := tx.ExecContext(ctx, `UPDATE agents SET pin_failed_attempts = 0,
	        pin_locked_until = NULL WHERE id = ?`, agentID); err != nil {
		return err
	}
	return tx.Commit()
}
