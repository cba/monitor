package monitor

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/cba/monitor/internal/config"
)

type icmpMonitor struct{}

func init() {
	Register(&icmpMonitor{})
}

func (m *icmpMonitor) Name() string { return "icmp" }

func (m *icmpMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	target := cfg.Target
	re := buildRemoteExec(cfg)

	var out string
	var err error
	start := time.Now()

	if re.User != "" {
		// Remote: always Linux
		out, err = execCommand(ctx, re, fmt.Sprintf("ping -c 1 -W 3 %s", target))
	} else if runtime.GOOS == "windows" {
		cmd := exec.CommandContext(ctx, "ping", "-n", "1", "-w", "3000", target)
		var buf []byte
		buf, err = cmd.CombinedOutput()
		out = string(buf)
	} else {
		cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "3", target)
		var buf []byte
		buf, err = cmd.CombinedOutput()
		out = string(buf)
	}

	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: fmt.Sprintf("Ping %s failed: %s", target, strings.TrimSpace(out)), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: fmt.Sprintf("Ping %s OK", target), Latency: latency, Timestamp: time.Now()}, nil
}
