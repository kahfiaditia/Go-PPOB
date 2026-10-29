package model

import "time"

type Product struct {
	ID        string    `json:"id"`
	Operator  string    `json:"operator"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}
