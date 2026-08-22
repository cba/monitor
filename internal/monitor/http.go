package monitor

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type httpMonitor struct{}

func init() {
	Register(&httpMonitor{})
}

func (m *httpMonitor) Name() string { return "http" }

func (m *httpMonitor) Check(ctx context.Context, target string) (*Result, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return &Result{Status: "up", Message: fmt.Sprintf("HTTP %d", resp.StatusCode), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "down", Message: fmt.Sprintf("HTTP %d", resp.StatusCode), Latency: latency, Timestamp: time.Now()}, nil
}
