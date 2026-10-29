package model

import "time"

type Product struct {
	ID        string    `json:"id"`
	Operator  string    `json:"operator"`
	Nominal   float64   `json:"nominal"`
	Name      string    `json:"name"`
	Price     float64   `json:"-"`
	HargaJual float64   `json:"harga_jual"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}
