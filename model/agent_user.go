package model

import "time"

type AgentUser struct {
	ID           int64     `json:"id"`
	AgentID      int64     `json:"agent_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}
