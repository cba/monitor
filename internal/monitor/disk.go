package monitor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cba/monitor/internal/config"
)

type diskMonitor struct{}

func init() {
	Register(&diskMonitor{})
}

func (m *diskMonitor) Name() string { return "disk" }

func (m *diskMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	re := buildRemoteExec(cfg)
	mountPoint := cfg.Path
	if mountPoint == "" {
		mountPoint = "/"
	}
	warn := cfg.Warn
	if warn == 0 {
		warn = 80.0
	}
	crit := cfg.Crit
	if crit == 0 {
		crit = 90.0
	}

	cmd := fmt.Sprintf(`df -h %s | awk 'NR==2{print $5}' | tr -d '%%'`, mountPoint)
	out, err := execCommand(ctx, re, cmd)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: 0, Timestamp: time.Now()}, nil
	}

	pct, err := strconv.ParseFloat(strings.TrimSpace(out), 64)
	if err != nil {
		return nil, fmt.Errorf("parse disk usage: %w", err)
	}

	msg := fmt.Sprintf("Disk %s: %.1f%%", mountPoint, pct)

	if pct >= crit {
		return &Result{Status: "down", Message: msg, Latency: 0, Timestamp: time.Now()}, nil
	}
	if pct >= warn {
		return &Result{Status: "warning", Message: msg, Latency: 0, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: msg, Latency: 0, Timestamp: time.Now()}, nil
}
