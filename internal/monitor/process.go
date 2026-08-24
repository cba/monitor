package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cba/monitor/internal/config"
)

type processMonitor struct{}

func init() {
	Register(&processMonitor{})
}

func (m *processMonitor) Name() string { return "process" }

func (m *processMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	re := buildRemoteExec(cfg)
	procName := cfg.ProcessName
	if procName == "" {
		return nil, fmt.Errorf("process_name is required")
	}

	cmd := fmt.Sprintf(`pgrep -x %s || pgrep -f %s`, procName, procName)
	out, err := execCommand(ctx, re, cmd)
	if err != nil || strings.TrimSpace(out) == "" {
		return &Result{Status: "down", Message: fmt.Sprintf("Process %s not running", procName), Latency: 0, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: fmt.Sprintf("Process %s running", procName), Latency: 0, Timestamp: time.Now()}, nil
}
