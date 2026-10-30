package model

import "time"

type Deposit struct {
	ID            string    `json:"id"`
	RequestID     string    `json:"request_id"`
	AgentID       int64     `json:"agent_id"`
	AdminID       int64     `json:"admin_id"`
	Amount        float64   `json:"amount"`
	BalanceBefore float64   `json:"balance_before"`
	BalanceAfter  float64   `json:"balance_after"`
	Reference     string    `json:"reference,omitempty"`
	Note          string    `json:"note,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
