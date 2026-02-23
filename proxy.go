package main

import "fmt"

// Worker represents a mining worker connected through Ultimate Proxy.
type Worker struct {
	ID        string `json:"_id"`
	Name      string `json:"name"`
	ProfileID string `json:"profile_id"`
	Status    string `json:"status"`
	Algorithm string `json:"algorithm"`
	Hashrate  uint64 `json:"hashrate"`
}

type WorkersResponse struct {
	Data       []Worker   `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type BulkAssignRequest struct {
	WorkerIDs []string `json:"worker_ids"`
	ProfileID string   `json:"profile_id"`
}

// HashrateResponse is the response from GET /v1/workers/hashrate.
type HashrateResponse struct {
	Hours       int                      `json:"hours"`
	Granularity int                      `json:"granularity"`
	Data        []map[string]interface{} `json:"data"`
	Stats       *HashrateStats           `json:"stats,omitempty"`
}

type HashrateStats struct {
	AvgHashrate  float64 `json:"avg_hashrate"`
	PeakHashrate float64 `json:"peak_hashrate"`
}

func proxyHeaders(apiKey string) map[string]string {
	return map[string]string{"X-API-Key": apiKey}
}

func fetchAllWorkers(baseURL, apiKey, algorithm string) ([]Worker, error) {
	var all []Worker
	page := 1
	for {
		url := fmt.Sprintf("%s/v1/workers?page=%d&limit=100&algorithm=%s", baseURL, page, algorithm)
		var resp WorkersResponse
		if err := fetchJSON(url, proxyHeaders(apiKey), &resp); err != nil {
			return nil, fmt.Errorf("fetch workers page %d: %w", page, err)
		}
		all = append(all, resp.Data...)
		if page >= resp.Pagination.TotalPages {
			break
		}
		page++
	}
	return all, nil
}

func bulkAssignWorkers(baseURL, apiKey string, workerIDs []string, profileID string) error {
	url := baseURL + "/v1/workers/bulk-assign"
	payload := BulkAssignRequest{
		WorkerIDs: workerIDs,
		ProfileID: profileID,
	}
	return postJSON(url, proxyHeaders(apiKey), payload)
}

func setDefaultProfile(baseURL, apiKey, profileID string) error {
	url := fmt.Sprintf("%s/v1/profiles/%s/default", baseURL, profileID)
	return postJSON(url, proxyHeaders(apiKey), nil)
}

// fetchHashrate calls GET /v1/workers/hashrate and returns the 1h average and peak hashrate (H/s).
func fetchHashrate(baseURL, apiKey, algorithm string) (avg float64, peak float64, err error) {
	url := fmt.Sprintf("%s/v1/workers/hashrate?algorithm=%s&timeRange=1h", baseURL, algorithm)
	var resp HashrateResponse
	if err := fetchJSON(url, proxyHeaders(apiKey), &resp); err != nil {
		return 0, 0, fmt.Errorf("fetch hashrate: %w", err)
	}
	if resp.Stats == nil {
		return 0, 0, nil
	}
	return resp.Stats.AvgHashrate, resp.Stats.PeakHashrate, nil
}
