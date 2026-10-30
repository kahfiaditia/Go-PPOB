package repository

import (
	"context"
	"database/sql"

	"pulsa-backend/model"
)

type AgentProductPricingRepository struct {
	db *sql.DB
}

func NewAgentProductPricingRepository(db *sql.DB) *AgentProductPricingRepository {
	return &AgentProductPricingRepository{db: db}
}

func (r *AgentProductPricingRepository) List(ctx context.Context, agentID int64) ([]model.AgentProductPricing, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.operator, p.nominal, p.name,
		       COALESCE(app.default_profit, ?) AS default_profit
		FROM products p
		LEFT JOIN agent_product_pricing app
		  ON app.product_id = p.id AND app.agent_id = ?
		WHERE p.is_active = 1
		ORDER BY p.operator, p.nominal ASC`, model.DefaultAgentProfit, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pricing := make([]model.AgentProductPricing, 0)
	for rows.Next() {
		var item model.AgentProductPricing
		if err := rows.Scan(
			&item.ProductID, &item.Operator, &item.Nominal, &item.ProductName, &item.DefaultProfit,
		); err != nil {
			return nil, err
		}
		pricing = append(pricing, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return pricing, nil
}

func (r *AgentProductPricingRepository) GetDefaultProfit(ctx context.Context, agentID int64, productID string) (float64, error) {
	var profit float64
	err := r.db.QueryRowContext(ctx,
		`SELECT default_profit FROM agent_product_pricing WHERE agent_id = ? AND product_id = ?`,
		agentID, productID,
	).Scan(&profit)
	if err == sql.ErrNoRows {
		return model.DefaultAgentProfit, nil
	}
	return profit, err
}

func (r *AgentProductPricingRepository) SetDefaultProfit(ctx context.Context, agentID int64, productID string, profit float64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_product_pricing (agent_id, product_id, default_profit)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE default_profit = VALUES(default_profit)`,
		agentID, productID, profit,
	)
	return err
}
