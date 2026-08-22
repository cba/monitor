package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type processMonitor struct{}

func init() {
	Register(&processMonitor{})
}

func (m *processMonitor) Name() string { return "process" }

func (m *processMonitor) Check(ctx context.Context, target string) (*Result, error) {
	re, rest := parseTarget(target)

	var procName string
	if parts := strings.Split(rest, ","); len(parts) >= 2 && parts[0] == "process" {
		procName = parts[1]
	}
	if procName == "" {
		return nil, fmt.Errorf("process target must include process name, e.g. 'process,nginx'")
	}

	cmd := fmt.Sprintf(`pgrep -x %s || pgrep -f %s`, procName, procName)
	start := time.Now()
	out, err := execCommand(ctx, re, cmd)
	latency := time.Since(start)
	if err != nil || strings.TrimSpace(out) == "" {
		return &Result{Status: "down", Message: fmt.Sprintf("process '%s' not found", procName), Latency: latency, Timestamp: time.Now()}, nil
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	pid := strings.TrimSpace(lines[0])
	return &Result{Status: "up", Message: fmt.Sprintf("process '%s' running (pid: %s)", procName, pid), Latency: latency, Timestamp: time.Now()}, nil
}
