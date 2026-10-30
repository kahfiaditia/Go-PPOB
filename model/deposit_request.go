package model

import "time"

type DepositRequest struct {
	ID          string     `json:"id"`
	AgentID     int64      `json:"agent_id"`
	Amount      float64    `json:"amount"`
	Reference   string     `json:"reference,omitempty"`
	Status      string     `json:"status"`
	AdminID     int64      `json:"admin_id,omitempty"`
	AdminNote   string     `json:"admin_note,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}
