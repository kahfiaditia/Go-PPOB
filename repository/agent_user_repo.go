package repository

import (
	"context"
	"database/sql"

	"pulsa-backend/model"
)

type AgentUserRepository struct {
	db *sql.DB
}

func NewAgentUserRepository(db *sql.DB) *AgentUserRepository {
	return &AgentUserRepository{db: db}
}

func (r *AgentUserRepository) FindByEmail(ctx context.Context, email string) (*model.AgentUser, bool, error) {
	var user model.AgentUser
	var agentActive bool
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.agent_id, u.username, u.email, u.password_hash, u.name,
		       u.is_active, u.created_at, a.is_active
		FROM agent_users u
		JOIN agents a ON a.id = u.agent_id
		WHERE u.email = ?
		LIMIT 1`, email).Scan(
		&user.ID, &user.AgentID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Name, &user.IsActive, &user.CreatedAt, &agentActive,
	)
	if err == sql.ErrNoRows {
		return nil, false, ErrNotFound
	}
	return &user, agentActive, err
}

func (r *AgentUserRepository) FindByID(ctx context.Context, agentID, userID int64) (*model.AgentUser, bool, error) {
	var user model.AgentUser
	var agentActive bool
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.agent_id, u.username, u.email, u.password_hash, u.name,
		       u.is_active, u.created_at, a.is_active
		FROM agent_users u
		JOIN agents a ON a.id = u.agent_id
		WHERE u.agent_id = ? AND u.id = ?
		LIMIT 1`, agentID, userID).Scan(
		&user.ID, &user.AgentID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Name, &user.IsActive, &user.CreatedAt, &agentActive,
	)
	if err == sql.ErrNoRows {
		return nil, false, ErrNotFound
	}
	return &user, agentActive, err
}

func (r *AgentUserRepository) ListByAgent(ctx context.Context, agentID int64) ([]model.AgentUser, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, agent_id, username, email, name, is_active, created_at
		FROM agent_users
		WHERE agent_id = ?
		ORDER BY id ASC`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]model.AgentUser, 0)
	for rows.Next() {
		var user model.AgentUser
		if err := rows.Scan(
			&user.ID, &user.AgentID, &user.Username, &user.Email,
			&user.Name, &user.IsActive, &user.CreatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *AgentUserRepository) Create(ctx context.Context, user *model.AgentUser) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_users (agent_id, username, email, password_hash, name)
		VALUES (?, ?, ?, ?, ?)`,
		user.AgentID, user.Username, user.Email, user.PasswordHash, user.Name,
	)
	if err != nil {
		return err
	}
	user.ID, err = result.LastInsertId()
	return err
}

func (r *AgentUserRepository) SetActive(ctx context.Context, agentID, userID int64, active bool) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE agent_users SET is_active = ? WHERE id = ? AND agent_id = ?`,
		active, userID, agentID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		var exists bool
		if err := r.db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM agent_users WHERE id = ? AND agent_id = ?)`,
			userID, agentID,
		).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
	}
	return nil
}
