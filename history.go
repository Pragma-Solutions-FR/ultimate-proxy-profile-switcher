package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

type Snapshot struct {
	Time     time.Time
	Coins    map[string]float64 // ticker -> daily revenue in fiat
	CoinsBTC map[string]float64 // ticker -> BTC/MH/Day
	Mining   string             // ticker being mined at this point
	Switched bool               // true if a switch happened at this snapshot
}

type History struct {
	mu        sync.Mutex
	snapshots []Snapshot
	maxLen    int
}

func NewHistory(maxLen int) *History {
	return &History{maxLen: maxLen}
}

func (h *History) Add(s Snapshot) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.snapshots = append(h.snapshots, s)
	if len(h.snapshots) > h.maxLen {
		h.snapshots = h.snapshots[len(h.snapshots)-h.maxLen:]
	}
}

func (h *History) All() []Snapshot {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Snapshot, len(h.snapshots))
	copy(out, h.snapshots)
	return out
}

// CoinAverage holds averaged values for a single coin across all snapshots.
type CoinAverage struct {
	Ticker   string
	AvgFiat  float64
	AvgBTCMH float64
	Count    int
}

// MinedAverage holds the weighted average of what was actually mined.
type MinedAverage struct {
	AvgFiat  float64
	AvgBTCMH float64
	Count    int
}

// Averages computes per-coin averages across all stored snapshots.
func (h *History) Averages() ([]CoinAverage, MinedAverage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	type acc struct {
		sumFiat float64
		sumBTC  float64
		count   int
	}
	m := make(map[string]*acc)
	var mined MinedAverage

	for _, s := range h.snapshots {
		for t, v := range s.Coins {
			a, ok := m[t]
			if !ok {
				a = &acc{}
				m[t] = a
			}
			a.sumFiat += v
			a.count++
		}
		for t, v := range s.CoinsBTC {
			if a, ok := m[t]; ok {
				a.sumBTC += v
			}
		}
		// Track the coin that was actually being mined
		if s.Mining != "" {
			if fiat, ok := s.Coins[s.Mining]; ok {
				mined.AvgFiat += fiat
				mined.Count++
			}
			if btc, ok := s.CoinsBTC[s.Mining]; ok {
				mined.AvgBTCMH += btc
			}
		}
	}

	avgs := make([]CoinAverage, 0, len(m))
	for t, a := range m {
		avgs = append(avgs, CoinAverage{
			Ticker:   t,
			AvgFiat:  a.sumFiat / float64(a.count),
			AvgBTCMH: a.sumBTC / float64(a.count),
			Count:    a.count,
		})
	}
	sort.Slice(avgs, func(i, j int) bool {
		return avgs[i].AvgFiat > avgs[j].AvgFiat
	})

	if mined.Count > 0 {
		mined.AvgFiat /= float64(mined.Count)
		mined.AvgBTCMH /= float64(mined.Count)
	}

	return avgs, mined
}

// ---------------------------------------------------------------------------
// Persistence
// ---------------------------------------------------------------------------

type persistedHistory struct {
	Snapshots []Snapshot `json:"snapshots"`
}

func (h *History) Save(path string) error {
	h.mu.Lock()
	data, err := json.Marshal(persistedHistory{Snapshots: h.snapshots})
	h.mu.Unlock()
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write history: %w", err)
	}
	return nil
}

func (h *History) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err // caller can check os.IsNotExist
	}
	var p persistedHistory
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("parse history: %w", err)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.snapshots = p.Snapshots
	// Trim to maxLen
	if len(h.snapshots) > h.maxLen {
		h.snapshots = h.snapshots[len(h.snapshots)-h.maxLen:]
	}
	return nil
}
