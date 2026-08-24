package monitor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cba/monitor/internal/config"
)

type cpuLoadMonitor struct{}

func init() {
	Register(&cpuLoadMonitor{})
}

func (m *cpuLoadMonitor) Name() string { return "cpu_load" }

func (m *cpuLoadMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	re := buildRemoteExec(cfg)
	warn := cfg.Warn
	if warn == 0 {
		warn = 5.0
	}
	crit := cfg.Crit
	if crit == 0 {
		crit = 10.0
	}

	out, err := execCommand(ctx, re, `awk '{print $1}' /proc/loadavg`)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: 0, Timestamp: time.Now()}, nil
	}

	load, err := strconv.ParseFloat(strings.TrimSpace(out), 64)
	if err != nil {
		return nil, fmt.Errorf("parse load: %w", err)
	}

	if load >= crit {
		return &Result{Status: "down", Message: fmt.Sprintf("Load %.2f >= %.2f", load, crit), Latency: 0, Timestamp: time.Now()}, nil
	}
	if load >= warn {
		return &Result{Status: "warning", Message: fmt.Sprintf("Load %.2f >= %.2f", load, warn), Latency: 0, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: fmt.Sprintf("Load %.2f", load), Latency: 0, Timestamp: time.Now()}, nil
}
