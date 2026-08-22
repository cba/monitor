package monitor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type memoryMonitor struct{}

func init() {
	Register(&memoryMonitor{})
}

func (m *memoryMonitor) Name() string { return "memory" }

func (m *memoryMonitor) Check(ctx context.Context, target string) (*Result, error) {
	re, rest := parseTarget(target)

	warnThreshold := 80.0
	critThreshold := 90.0
	if parts := strings.Split(rest, ","); len(parts) > 0 && parts[0] == "memory" {
		if len(parts) > 1 {
			if v, err := strconv.ParseFloat(parts[1], 64); err == nil {
				warnThreshold = v
			}
		}
		if len(parts) > 2 {
			if v, err := strconv.ParseFloat(parts[2], 64); err == nil {
				critThreshold = v
			}
		}
	}

	start := time.Now()
	out, err := execCommand(ctx, re, `free -m | awk 'NR==2{printf "%d %d %.1f", $3, $2, $3/$2*100}'`)
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}

	fields := strings.Fields(strings.TrimSpace(out))
	if len(fields) < 3 {
		return &Result{Status: "down", Message: fmt.Sprintf("unexpected output: %s", out), Latency: latency, Timestamp: time.Now()}, nil
	}

	usedMB, _ := strconv.Atoi(fields[0])
	totalMB, _ := strconv.Atoi(fields[1])
	usagePct, _ := strconv.ParseFloat(fields[2], 64)

	msg := fmt.Sprintf("used: %dMB/%dMB (%.1f%%)", usedMB, totalMB, usagePct)
	if usagePct >= critThreshold {
		return &Result{Status: "down", Message: msg, Latency: latency, Timestamp: time.Now()}, nil
	}
	if usagePct >= warnThreshold {
		return &Result{Status: "warning", Message: msg, Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: msg, Latency: latency, Timestamp: time.Now()}, nil
}
