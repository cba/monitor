package monitor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cba/monitor/internal/config"
)

type keywordMonitor struct{}

func init() {
	Register(&keywordMonitor{})
}

func (m *keywordMonitor) Name() string { return "keyword" }

func (m *keywordMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("keyword request: %w", err)
	}
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return &Result{Status: "down", Message: fmt.Sprintf("read body: %v", err), Latency: latency, Timestamp: time.Now()}, nil
	}

	if strings.Contains(string(body), cfg.Keyword) {
		return &Result{Status: "up", Message: fmt.Sprintf("Keyword '%s' found", cfg.Keyword), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "warning", Message: fmt.Sprintf("Keyword '%s' not found", cfg.Keyword), Latency: latency, Timestamp: time.Now()}, nil
}
