package monitor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type keywordMonitor struct{}

func init() {
	Register(&keywordMonitor{})
}

func (m *keywordMonitor) Name() string { return "keyword" }

func (m *keywordMonitor) Check(ctx context.Context, target string) (*Result, error) {
	parts := strings.SplitN(target, "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("keyword target must be 'url|keyword', got: %s", target)
	}
	url, keyword := parts[0], parts[1]

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}

	if strings.Contains(string(body), keyword) {
		return &Result{Status: "up", Message: fmt.Sprintf("keyword '%s' found", keyword), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "down", Message: fmt.Sprintf("keyword '%s' not found", keyword), Latency: latency, Timestamp: time.Now()}, nil
}
