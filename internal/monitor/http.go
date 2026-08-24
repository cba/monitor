package monitor

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cba/monitor/internal/config"
)

type httpMonitor struct{}

func init() {
	Register(&httpMonitor{})
}

func (m *httpMonitor) Name() string { return "http" }

func (m *httpMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return &Result{Status: "warning", Message: fmt.Sprintf("HTTP %d", resp.StatusCode), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: fmt.Sprintf("HTTP %d", resp.StatusCode), Latency: latency, Timestamp: time.Now()}, nil
}
