package service

import (
	"errors"
	"testing"
)

func TestCalculateTopupPriceUsesConfiguredProfitAndDistributorPrice(t *testing.T) {
	sellPrice, profit, err := calculateTopupPrice(11000, 750, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sellPrice != 11000 || profit != 750 {
		t.Fatalf("got distributor price %v and profit %v; want 11000 and 750", sellPrice, profit)
	}
}

func TestCalculateTopupPriceUsesRequestedProfitWithoutChangingDistributorPrice(t *testing.T) {
	requestedProfit := 2000.0
	sellPrice, profit, err := calculateTopupPrice(11000, 750, &requestedProfit)
	if err != nil {
		t.Fatal(err)
	}
	if sellPrice != 11000 || profit != 2000 {
		t.Fatalf("got distributor price %v and profit %v; want 11000 and 2000", sellPrice, profit)
	}
}

func TestCalculateTopupPriceRejectsProfitOutsideAllowedRange(t *testing.T) {
	for _, requestedProfit := range []float64{499, 2001} {
		_, _, err := calculateTopupPrice(11000, 750, &requestedProfit)
		if !errors.Is(err, ErrInvalidProfit) {
			t.Errorf("calculateTopupPrice requested profit %v error = %v; want ErrInvalidProfit", requestedProfit, err)
		}
	}
}

func TestCalculateTopupPriceRejectsInvalidDefaultsAndDistributorPrice(t *testing.T) {
	for _, test := range []struct {
		distributorPrice float64
		defaultProfit    float64
		wantErr          error
	}{
		{distributorPrice: 0, defaultProfit: 500, wantErr: ErrInvalidProductPrice},
		{distributorPrice: 10000, defaultProfit: 499, wantErr: ErrInvalidProfit},
		{distributorPrice: 10000, defaultProfit: 2001, wantErr: ErrInvalidProfit},
	} {
		if _, _, err := calculateTopupPrice(test.distributorPrice, test.defaultProfit, nil); !errors.Is(err, test.wantErr) {
			t.Errorf("calculateTopupPrice(%v, %v) error = %v; want %v", test.distributorPrice, test.defaultProfit, err, test.wantErr)
		}
	}
}
