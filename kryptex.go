package main

import "fmt"

// KryptexRates holds fiat and crypto exchange rates from the Kryptex Pool API.
type KryptexRates struct {
	Fiat   map[string]float64 `json:"fiat"`
	Crypto map[string]float64 `json:"crypto"`
}

func fetchRates(baseURL string) (*KryptexRates, error) {
	var rates KryptexRates
	if err := fetchJSON(baseURL+"/rates", nil, &rates); err != nil {
		return nil, fmt.Errorf("fetch rates: %w", err)
	}
	return &rates, nil
}

func fetchDailyRevenue(baseURL, ticker string, hashrate int) (float64, error) {
	url := fmt.Sprintf("%s/daily-revenue/%s?hashrate=%d", baseURL, ticker, hashrate)
	rev, err := fetchFloat(url)
	if err != nil {
		return 0, fmt.Errorf("fetch revenue %s: %w", ticker, err)
	}
	return rev, nil
}
