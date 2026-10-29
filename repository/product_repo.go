package repository

import (
	"context"
	"database/sql"

	"pulsa-backend/model"
)

type ProductRepository struct{ db *sql.DB }

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// FindByID (untuk agent)
func (r *ProductRepository) FindByID(ctx context.Context, id string) (*model.Product, error) {
	q := `SELECT id, operator, name, price, is_active, created_at 
	      FROM products WHERE id = ? AND is_active = 1 LIMIT 1`
	var p model.Product
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&p.ID, &p.Operator, &p.Name, &p.Price, &p.IsActive, &p.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &p, err
}

// FindAnyByID (untuk admin, termasuk yang nonaktif)
func (r *ProductRepository) FindAnyByID(ctx context.Context, id string) (*model.Product, error) {
	q := `SELECT id, operator, name, price, is_active, created_at 
	      FROM products WHERE id = ? LIMIT 1`
	var p model.Product
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&p.ID, &p.Operator, &p.Name, &p.Price, &p.IsActive, &p.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &p, err
}

// ListActive (untuk agent)
func (r *ProductRepository) ListActive(ctx context.Context, operator string) ([]model.Product, error) {
	q := `SELECT id, operator, name, price, is_active, created_at 
	      FROM products WHERE is_active = 1`
	args := []interface{}{}
	if operator != "" {
		q += " AND operator = ?"
		args = append(args, operator)
	}
	q += " ORDER BY operator, price ASC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Operator, &p.Name, &p.Price, &p.IsActive, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// ListAll (untuk admin)
func (r *ProductRepository) ListAll(ctx context.Context, operator string) ([]model.Product, error) {
	q := `SELECT id, operator, name, price, is_active, created_at 
	      FROM products WHERE 1=1`
	args := []interface{}{}
	if operator != "" {
		q += " AND operator = ?"
		args = append(args, operator)
	}
	q += " ORDER BY operator, price ASC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Operator, &p.Name, &p.Price, &p.IsActive, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// Create
func (r *ProductRepository) Create(ctx context.Context, p *model.Product) error {
	q := `INSERT INTO products (id, operator, name, price, is_active) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.Operator, p.Name, p.Price, p.IsActive)
	return err
}

// Update
func (r *ProductRepository) Update(ctx context.Context, p *model.Product) error {
	q := `UPDATE products SET operator=?, name=?, price=?, is_active=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, q, p.Operator, p.Name, p.Price, p.IsActive, p.ID)
	return err
}

// SetActive
func (r *ProductRepository) SetActive(ctx context.Context, id string, active bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE products SET is_active = ? WHERE id = ?`, active, id)
	return err
}
