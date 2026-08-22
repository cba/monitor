package monitor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type diskMonitor struct{}

func init() {
	Register(&diskMonitor{})
}

func (m *diskMonitor) Name() string { return "disk" }

func (m *diskMonitor) Check(ctx context.Context, target string) (*Result, error) {
	re, rest := parseTarget(target)

	warnThreshold := 80.0
	critThreshold := 90.0
	mountPoint := "/"

	if parts := strings.Split(rest, ","); len(parts) > 0 && parts[0] == "disk" {
		if len(parts) > 1 && parts[1] != "" {
			mountPoint = parts[1]
		}
		if len(parts) > 2 {
			if v, err := strconv.ParseFloat(parts[2], 64); err == nil {
				warnThreshold = v
			}
		}
		if len(parts) > 3 {
			if v, err := strconv.ParseFloat(parts[3], 64); err == nil {
				critThreshold = v
			}
		}
	}

	cmd := fmt.Sprintf(`df -h %s | awk 'NR==2{print $5}' | tr -d '%%'`, mountPoint)
	start := time.Now()
	out, err := execCommand(ctx, re, cmd)
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}

	usagePct, err := strconv.ParseFloat(strings.TrimSpace(out), 64)
	if err != nil {
		return &Result{Status: "down", Message: fmt.Sprintf("parse error: %s", out), Latency: latency, Timestamp: time.Now()}, nil
	}

	msg := fmt.Sprintf("%s: %.1f%% used", mountPoint, usagePct)
	if usagePct >= critThreshold {
		return &Result{Status: "down", Message: msg, Latency: latency, Timestamp: time.Now()}, nil
	}
	if usagePct >= warnThreshold {
		return &Result{Status: "warning", Message: msg, Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: msg, Latency: latency, Timestamp: time.Now()}, nil
}
