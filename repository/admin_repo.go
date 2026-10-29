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

func (r *AdminRepository) FindByEmail(ctx context.Context, email string) (*model.Admin, error) {
	q := `SELECT id, username, email, password_hash, name, role, is_active, created_at FROM admins WHERE email = ? LIMIT 1`
	var a model.Admin
	err := r.db.QueryRowContext(ctx, q, email).Scan(
		&a.ID, &a.Username, &a.Email, &a.PasswordHash, &a.Name, &a.Role, &a.IsActive, &a.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &a, err
}
