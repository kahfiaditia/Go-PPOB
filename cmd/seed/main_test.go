package main

import "testing"

func TestSeedProductPricing(t *testing.T) {
	for _, product := range seedProducts() {
		costMargin := product.price - product.nominal
		if costMargin < 500 || costMargin > 1000 {
			t.Errorf("%s price margin = %.2f; want 500..1000", product.id, costMargin)
		}
		if profit := product.hargaJual - product.price; profit != 500 {
			t.Errorf("%s default profit = %.2f; want 500", product.id, profit)
		}
	}
}
