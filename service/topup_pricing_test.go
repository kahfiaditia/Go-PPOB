package service

import (
	"errors"
	"testing"
)

func TestCalculateTopupPriceUsesDefaultWhenProfitOmitted(t *testing.T) {
	sellPrice, profit, err := calculateTopupPrice(10000, 10400, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sellPrice != 10400 || profit != 400 {
		t.Fatalf("got sell price %v and profit %v; want 10400 and 400", sellPrice, profit)
	}
}

func TestCalculateTopupPriceAcceptsHigherRequestedProfit(t *testing.T) {
	requestedProfit := 700.0
	sellPrice, profit, err := calculateTopupPrice(10000, 10400, &requestedProfit)
	if err != nil {
		t.Fatal(err)
	}
	if sellPrice != 10700 || profit != 700 {
		t.Fatalf("got sell price %v and profit %v; want 10700 and 700", sellPrice, profit)
	}
}

func TestCalculateTopupPriceRejectsProfitBelowDefault(t *testing.T) {
	requestedProfit := 399.0
	_, _, err := calculateTopupPrice(10000, 10400, &requestedProfit)
	if !errors.Is(err, ErrProfitBelowDefault) {
		t.Fatalf("got error %v; want ErrProfitBelowDefault", err)
	}
}

func TestCalculateTopupPriceRejectsInvalidProductPrices(t *testing.T) {
	for _, test := range []struct {
		cost      float64
		sellPrice float64
	}{
		{cost: 0, sellPrice: 100},
		{cost: 100, sellPrice: 99},
	} {
		if _, _, err := calculateTopupPrice(test.cost, test.sellPrice, nil); !errors.Is(err, ErrInvalidProductPrice) {
			t.Errorf("calculateTopupPrice(%v, %v) error = %v; want ErrInvalidProductPrice", test.cost, test.sellPrice, err)
		}
	}
}
