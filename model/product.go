package model

import "time"

type Product struct {
	ID            string    `json:"id"`
	Operator      string    `json:"operator"`
	Nominal       float64   `json:"nominal"` // <-- Baru
	Name          string    `json:"name"`
	BasePrice     float64   `json:"base_price"`     // <-- Harga dasar (Ganti 'price')
	DefaultMarkup float64   `json:"default_markup"` // <-- Markup default
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}
