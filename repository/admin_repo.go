package repository

import (
	"context"
	"database/sql"

	"pulsa-backend/model"
)

type AdminRepository struct{ db *sql.DB }

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) FindByUsername(ctx context.Context, username string) (*model.Admin, error) {
	q := `SELECT id, username, password_hash, name, role, is_active, created_at FROM admins WHERE username = ? LIMIT 1`
	var a model.Admin
	err := r.db.QueryRowContext(ctx, q, username).Scan(
		&a.ID, &a.Username, &a.PasswordHash, &a.Name, &a.Role, &a.IsActive, &a.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &a, err
}
