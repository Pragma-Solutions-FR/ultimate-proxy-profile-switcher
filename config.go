package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type CoinConfig struct {
	Ticker        string `yaml:"ticker"`
	RevenueTicker string `yaml:"revenue_ticker,omitempty"` // override for /daily-revenue/ endpoint (e.g. XTM_rx)
	ProfileID     string `yaml:"profile_id"`
}

type Config struct {
	ProxyBaseURL    string
	KryptexBaseURL  string       `yaml:"kryptex_base_url"`
	ProxyAPIKey     string       `yaml:"proxy_api_key"`
	ProxyAlgorithm  string       `yaml:"proxy_algorithm"` // algorithm used to list workers (e.g. kawpow, randomx)
	FiatCurrency    string       `yaml:"fiat_currency"`
	Interval        int          `yaml:"interval"`
	DefaultHashrate int          `yaml:"default_hashrate"`
	HistoryFile     string       `yaml:"history_file"` // path to persist history (default: profswitch_history.json)
	Coins           []CoinConfig `yaml:"coins"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	// Defaults
	cfg.ProxyBaseURL = "https://api.ultimate-proxy.com"
	if cfg.KryptexBaseURL == "" {
		cfg.KryptexBaseURL = "https://pool.kryptex.com/api/v1"
	}
	if cfg.FiatCurrency == "" {
		cfg.FiatCurrency = "USD"
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 300
	}
	if cfg.DefaultHashrate <= 0 {
		cfg.DefaultHashrate = 1000
	}
	if cfg.HistoryFile == "" {
		cfg.HistoryFile = "profswitch_history.json"
	}
	if cfg.ProxyAlgorithm == "" {
		return nil, fmt.Errorf("proxy_algorithm is required (e.g. kawpow, randomx, verushash)")
	}
	cfg.ProxyAlgorithm = strings.ToLower(cfg.ProxyAlgorithm)
	for i := range cfg.Coins {
		cfg.Coins[i].Ticker = strings.ToUpper(cfg.Coins[i].Ticker)
		if cfg.Coins[i].RevenueTicker != "" {
			cfg.Coins[i].RevenueTicker = strings.ToUpper(cfg.Coins[i].RevenueTicker)
		}
	}
	if len(cfg.Coins) == 0 {
		return nil, fmt.Errorf("no coins configured")
	}
	return &cfg, nil
}
