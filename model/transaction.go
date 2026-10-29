package model

import "time"

type Transaction struct {
	ID               string    `json:"id"`
	AgentID          int64     `json:"agent_id"`
	ProductID        string    `json:"product_id"`
	CustomerNo       string    `json:"customer_no"`
	Amount           float64   `json:"amount"`
	Status           string    `json:"status"`
	FailureReason    string    `json:"failure_reason,omitempty"`
	CallbackStatus   string    `json:"callback_status"`
	CallbackAttempts int       `json:"callback_attempts"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
