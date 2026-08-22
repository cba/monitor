package monitor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type cpuLoadMonitor struct{}

func init() {
	Register(&cpuLoadMonitor{})
}

func (m *cpuLoadMonitor) Name() string { return "cpu_load" }

func (m *cpuLoadMonitor) Check(ctx context.Context, target string) (*Result, error) {
	re, rest := parseTarget(target)

	warnThreshold := 5.0
	critThreshold := 10.0
	if parts := strings.Split(rest, ","); len(parts) > 0 && parts[0] == "cpu_load" {
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
	out, err := execCommand(ctx, re, `awk '{print $1}' /proc/loadavg`)
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}

	load1, err := strconv.ParseFloat(strings.TrimSpace(out), 64)
	if err != nil {
		return &Result{Status: "down", Message: fmt.Sprintf("parse error: %s", out), Latency: latency, Timestamp: time.Now()}, nil
	}

	msg := fmt.Sprintf("load1: %.2f", load1)
	if load1 >= critThreshold {
		return &Result{Status: "down", Message: msg, Latency: latency, Timestamp: time.Now()}, nil
	}
	if load1 >= warnThreshold {
		return &Result{Status: "warning", Message: msg, Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: msg, Latency: latency, Timestamp: time.Now()}, nil
}
