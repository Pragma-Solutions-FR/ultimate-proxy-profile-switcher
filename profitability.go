package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

// CoinProfitability holds the computed profitability metrics for a single coin.
type CoinProfitability struct {
	Ticker           string
	ProfileID        string
	DailyRevCoin     float64
	CryptoRateUSD    float64
	DailyRevenueFiat float64
	BTCPerMHDay      float64 // BTC equivalent per MH/day
}

// formatHashrate returns a human-readable hashrate string (H/s, KH/s, MH/s, GH/s, TH/s).
func formatHashrate(h float64) string {
	switch {
	case h >= 1e12:
		return fmt.Sprintf("%.2f TH/s", h/1e12)
	case h >= 1e9:
		return fmt.Sprintf("%.2f GH/s", h/1e9)
	case h >= 1e6:
		return fmt.Sprintf("%.2f MH/s", h/1e6)
	case h >= 1e3:
		return fmt.Sprintf("%.2f KH/s", h/1e3)
	default:
		return fmt.Sprintf("%.0f H/s", h)
	}
}

// computeProfitability fetches live rates and daily revenue for all configured coins
// and returns them sorted from most to least profitable.
func computeProfitability(cfg *Config, hashrate int) ([]CoinProfitability, error) {
	rates, err := fetchRates(cfg.KryptexBaseURL)
	if err != nil {
		return nil, err
	}

	fiatRate, ok := rates.Fiat[strings.ToUpper(cfg.FiatCurrency)]
	if !ok {
		return nil, fmt.Errorf("unknown fiat currency: %s", cfg.FiatCurrency)
	}

	btcRate, ok := rates.Crypto["BTC"]
	if !ok {
		return nil, fmt.Errorf("BTC rate not found in rates")
	}

	type result struct {
		prof CoinProfitability
		err  error
	}

	results := make([]result, len(cfg.Coins))
	var wg sync.WaitGroup

	for i, coin := range cfg.Coins {
		wg.Add(1)
		go func(idx int, c CoinConfig) {
			defer wg.Done()
			// Use revenue_ticker for daily-revenue endpoint if set (e.g. XTM_rx), otherwise use ticker
			revTicker := c.Ticker
			if c.RevenueTicker != "" {
				revTicker = c.RevenueTicker
			}
			rev, err := fetchDailyRevenue(cfg.KryptexBaseURL, revTicker, hashrate)
			if err != nil {
				results[idx] = result{err: err}
				return
			}
			// Always use the base ticker for rate lookup
			cryptoRate, ok := rates.Crypto[c.Ticker]
			if !ok {
				results[idx] = result{err: fmt.Errorf("no crypto rate for %s", c.Ticker)}
				return
			}
			// daily_revenue (in coin) × coin_price_in_USD / fiat_rate
			fiatRevenue := rev * cryptoRate / fiatRate

			// BTC per MH/day: normalize revenue to 1 MH/s then convert to BTC
			// daily_revenue is for hashrate (in H/s), so scale to 1,000,000 H/s
			btcPerMHDay := (rev * cryptoRate / btcRate) * (1_000_000 / float64(hashrate))

			results[idx] = result{
				prof: CoinProfitability{
					Ticker:           c.Ticker,
					ProfileID:        c.ProfileID,
					DailyRevCoin:     rev,
					CryptoRateUSD:    cryptoRate,
					DailyRevenueFiat: fiatRevenue,
					BTCPerMHDay:      btcPerMHDay,
				},
			}
		}(i, coin)
	}

	wg.Wait()

	var profs []CoinProfitability
	for _, r := range results {
		if r.err != nil {
			log.Printf("[WARN] %v", r.err)
			continue
		}
		profs = append(profs, r.prof)
	}

	sort.Slice(profs, func(i, j int) bool {
		return profs[i].DailyRevenueFiat > profs[j].DailyRevenueFiat
	})

	return profs, nil
}

// printTable prints the profitability ranking table and historical averages.
func printTable(profs []CoinProfitability, fiat string, currentTicker string, hist *History, hashrate int) {
	now := time.Now().Format("2006-01-02 15:04:05")
	currency := strings.ToUpper(fiat)

	fmt.Println()
	fmt.Printf("  Profitability Report — %s  ⚡ %s\n", now, formatHashrate(float64(hashrate)))
	fmt.Println(strings.Repeat("─", 84))
	fmt.Printf("  %-4s  %-10s  %16s  %16s  %14s  %12s\n",
		"Rank", "Coin", "Daily (coin)", fmt.Sprintf("Daily (%s)", currency), "BTC/MH/Day", "Price (USD)")
	fmt.Println(strings.Repeat("─", 84))

	for i, p := range profs {
		marker := "  "
		if p.Ticker == currentTicker {
			marker = "★ "
		}
		fmt.Printf("  %-4d  %s%-8s  %16.8f  %16.8f  %14.10f  %12.6f\n",
			i+1, marker, p.Ticker, p.DailyRevCoin, p.DailyRevenueFiat, p.BTCPerMHDay, p.CryptoRateUSD)
	}

	fmt.Println(strings.Repeat("─", 84))
	fmt.Println("  ★ = currently mining")

	// Print averages if we have history
	avgs, mined := hist.Averages()
	if len(avgs) > 0 && avgs[0].Count > 1 {
		fmt.Println()
		fmt.Printf("  Averages (%d samples)\n", avgs[0].Count)
		fmt.Println(strings.Repeat("─", 56))
		fmt.Printf("  %-4s  %-10s  %16s  %14s\n", "Rank", "Coin", fmt.Sprintf("Avg (%s)", currency), "Avg BTC/MH/D")
		fmt.Println(strings.Repeat("─", 56))
		for i, a := range avgs {
			marker := "  "
			if a.Ticker == currentTicker {
				marker = "★ "
			}
			fmt.Printf("  %-4d  %s%-8s  %16.8f  %14.10f\n",
				i+1, marker, a.Ticker, a.AvgFiat, a.AvgBTCMH)
		}
		fmt.Println(strings.Repeat("─", 56))
		if mined.Count > 0 {
			fmt.Printf("  %s⛏  MINED AVG%s  %16.8f  %14.10f\n",
				colorBold, colorReset, mined.AvgFiat, mined.AvgBTCMH)
			fmt.Println(strings.Repeat("─", 56))
		}
	}

	fmt.Println()
}

// switchWorkers bulk-assigns all workers not already on targetProfileID to the new profile.
func switchWorkers(cfg *Config, targetProfileID, targetTicker string) error {
	workers, err := fetchAllWorkers(cfg.ProxyBaseURL, cfg.ProxyAPIKey, cfg.ProxyAlgorithm)
	if err != nil {
		return fmt.Errorf("fetch workers: %w", err)
	}

	// Only switch workers that are NOT already on the target profile
	var ids []string
	for _, w := range workers {
		if w.ID == "" {
			continue
		}
		if w.ProfileID != targetProfileID {
			ids = append(ids, w.ID)
		}
	}

	if len(ids) == 0 {
		return nil
	}

	log.Printf("[SWITCH] Assigning %d/%d worker(s) to profile %s (%s)...\n", len(ids), len(workers), targetProfileID, targetTicker)
	if err := bulkAssignWorkers(cfg.ProxyBaseURL, cfg.ProxyAPIKey, ids, targetProfileID); err != nil {
		return fmt.Errorf("bulk assign: %w", err)
	}
	return nil
}
