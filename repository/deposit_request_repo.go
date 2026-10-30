package repository

import (
	"context"
	"database/sql"
	"errors"

	"pulsa-backend/model"

	mysql "github.com/go-sql-driver/mysql"
)

var ErrDuplicateDepositReference = errors.New("deposit reference already submitted")

type DepositRequestRepository struct {
	db *sql.DB
}

func NewDepositRequestRepository(db *sql.DB) *DepositRequestRepository {
	return &DepositRequestRepository{db: db}
}

func (r *DepositRequestRepository) Create(ctx context.Context, request *model.DepositRequest) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO deposit_requests (id, agent_id, amount, reference)
		VALUES (?, ?, ?, ?)`,
		request.ID, request.AgentID, request.Amount, request.Reference,
	)
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return ErrDuplicateDepositReference
	}
	return err
}

func (r *DepositRequestRepository) GetByIDForUpdate(ctx context.Context, tx *sql.Tx, id string) (*model.DepositRequest, error) {
	request, err := scanDepositRequest(tx.QueryRowContext(ctx, `
		SELECT id, agent_id, amount, COALESCE(reference,''), status,
		       COALESCE(admin_id,0), COALESCE(admin_note,''), created_at, processed_at
		FROM deposit_requests
		WHERE id = ?
		FOR UPDATE`, id))
	return request, err
}

func (r *DepositRequestRepository) UpdateDecision(ctx context.Context, tx *sql.Tx,
	requestID string, adminID int64, status, note string) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE deposit_requests
		SET status = ?, admin_id = ?, admin_note = ?, processed_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'PENDING'`,
		status, adminID, note, requestID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DepositRequestRepository) ListByAgent(ctx context.Context, agentID int64,
	status string, limit, offset int) ([]model.DepositRequest, error) {
	return r.list(ctx, `agent_id = ?`, []interface{}{agentID}, status, limit, offset)
}

func (r *DepositRequestRepository) ListAll(ctx context.Context, status string, limit, offset int) ([]model.DepositRequest, error) {
	return r.list(ctx, `1 = 1`, nil, status, limit, offset)
}

func (r *DepositRequestRepository) list(ctx context.Context, condition string, args []interface{},
	status string, limit, offset int) ([]model.DepositRequest, error) {
	query := `
		SELECT id, agent_id, amount, COALESCE(reference,''), status,
		       COALESCE(admin_id,0), COALESCE(admin_note,''), created_at, processed_at
		FROM deposit_requests
		WHERE ` + condition
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]model.DepositRequest, 0)
	for rows.Next() {
		request, err := scanDepositRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, *request)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return requests, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDepositRequest(row rowScanner) (*model.DepositRequest, error) {
	var request model.DepositRequest
	var processedAt sql.NullTime
	err := row.Scan(
		&request.ID, &request.AgentID, &request.Amount, &request.Reference,
		&request.Status, &request.AdminID, &request.AdminNote, &request.CreatedAt, &processedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if processedAt.Valid {
		request.ProcessedAt = &processedAt.Time
	}
	return &request, nil
}
