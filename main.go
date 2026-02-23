package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config YAML file")
	shortConfig := flag.String("c", "", "Path to config YAML file (shorthand)")
	dryRun := flag.Bool("dry-run", false, "Display profitability table without switching")
	once := flag.Bool("once", false, "Run a single cycle and exit")
	flag.Parse()

	if *shortConfig != "" {
		configPath = shortConfig
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}

	log.Printf("[INFO] Loaded %d coin(s), interval=%ds, fiat=%s", len(cfg.Coins), cfg.Interval, cfg.FiatCurrency)
	if *dryRun {
		log.Println("[INFO] Dry-run mode: will NOT switch workers")
	}

	currentTicker := ""
	// 24h of history: 86400s / interval. Chart still shows last 60 points.
	histSize := (86400 / cfg.Interval) + 1
	hist := NewHistory(histSize)

	// Load persisted history
	if err := hist.Load(cfg.HistoryFile); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[WARN] Failed to load history: %v", err)
		}
	} else {
		snaps := hist.All()
		if len(snaps) > 0 {
			currentTicker = snaps[len(snaps)-1].Mining
			log.Printf("[INFO] Restored %d snapshots from %s (last mining: %s)", len(snaps), cfg.HistoryFile, currentTicker)
		}
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	run := func() {
		// Fetch aggregated hashrate from /v1/workers/hashrate (1h avg)
		hashrate := cfg.DefaultHashrate
		avgHR, _, err := fetchHashrate(cfg.ProxyBaseURL, cfg.ProxyAPIKey, cfg.ProxyAlgorithm)
		if err != nil {
			log.Printf("[WARN] Failed to fetch hashrate: %v — using default %d H/s", err, cfg.DefaultHashrate)
		} else if avgHR > 0 {
			hashrate = int(avgHR)
			log.Printf("[INFO] Live hashrate (1h avg): %s", formatHashrate(avgHR))
		} else {
			log.Printf("[WARN] No hashrate data, using default: %d H/s", cfg.DefaultHashrate)
		}

		profs, err := computeProfitability(cfg, hashrate)
		if err != nil {
			log.Printf("[ERROR] %v", err)
			return
		}
		if len(profs) == 0 {
			log.Println("[WARN] No profitability data available")
			return
		}

		printTable(profs, cfg.FiatCurrency, currentTicker, hist, hashrate)

		best := profs[0]
		switched := false

		// Always ensure best coin is the default profile (for new miners connecting)
		if !*dryRun {
			if err := setDefaultProfile(cfg.ProxyBaseURL, cfg.ProxyAPIKey, best.ProfileID); err != nil {
				log.Printf("[WARN] Failed to set default profile: %v", err)
			}
		}

		if best.Ticker != currentTicker {
			switched = currentTicker != "" // not a switch on first run

			if currentTicker != "" && len(profs) > 1 {
				var oldFiat float64
				for _, p := range profs {
					if p.Ticker == currentTicker {
						oldFiat = p.DailyRevenueFiat
						break
					}
				}
				if oldFiat > 0 {
					pctGain := ((best.DailyRevenueFiat - oldFiat) / oldFiat) * 100
					log.Printf("[SWITCH] %s → %s (more profitable by +%.1f%%)\n", currentTicker, best.Ticker, pctGain)
				} else {
					log.Printf("[SWITCH] → %s (most profitable)\n", best.Ticker)
				}
			} else if currentTicker == "" {
				log.Printf("[INIT] Starting with most profitable coin: %s\n", best.Ticker)
			}

			if !*dryRun {
				if err := switchWorkers(cfg, best.ProfileID, best.Ticker); err != nil {
					log.Printf("[ERROR] Switch failed: %v", err)
					return
				}
			}

			currentTicker = best.Ticker
		}

		// Record snapshot for chart
		coins := make(map[string]float64, len(profs))
		coinsBTC := make(map[string]float64, len(profs))
		for _, p := range profs {
			coins[p.Ticker] = p.DailyRevenueFiat
			coinsBTC[p.Ticker] = p.BTCPerMHDay
		}
		hist.Add(Snapshot{
			Time:     time.Now(),
			Coins:    coins,
			CoinsBTC: coinsBTC,
			Mining:   currentTicker,
			Switched: switched,
		})

		// Persist history to disk
		if err := hist.Save(cfg.HistoryFile); err != nil {
			log.Printf("[WARN] Failed to save history: %v", err)
		}

		printChart(hist, cfg.FiatCurrency)
	}

	// First run
	run()

	if *once {
		return
	}

	for {
		select {
		case <-ticker.C:
			run()
		case <-stop:
			log.Println("[INFO] Shutting down...")
			return
		}
	}
}
