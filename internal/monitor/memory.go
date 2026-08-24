package monitor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cba/monitor/internal/config"
)

type memoryMonitor struct{}

func init() {
	Register(&memoryMonitor{})
}

func (m *memoryMonitor) Name() string { return "memory" }

func (m *memoryMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	re := buildRemoteExec(cfg)
	warn := cfg.Warn
	if warn == 0 {
		warn = 80.0
	}
	crit := cfg.Crit
	if crit == 0 {
		crit = 90.0
	}

	out, err := execCommand(ctx, re, `free -m | awk 'NR==2{printf "%d %d %.1f", $3, $2, $3/$2*100}'`)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: 0, Timestamp: time.Now()}, nil
	}

	fields := strings.Fields(out)
	if len(fields) < 3 {
		return nil, fmt.Errorf("unexpected memory output: %s", out)
	}

	used, _ := strconv.Atoi(fields[0])
	total, _ := strconv.Atoi(fields[1])
	pct, _ := strconv.ParseFloat(fields[2], 64)

	msg := fmt.Sprintf("Memory %dMB/%dMB (%.1f%%)", used, total, pct)

	if pct >= crit {
		return &Result{Status: "down", Message: msg, Latency: 0, Timestamp: time.Now()}, nil
	}
	if pct >= warn {
		return &Result{Status: "warning", Message: msg, Latency: 0, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: msg, Latency: 0, Timestamp: time.Now()}, nil
}
