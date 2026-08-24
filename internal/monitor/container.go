package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cba/monitor/internal/config"
)

type containerMonitor struct{}

func init() {
	Register(&containerMonitor{})
}

func (m *containerMonitor) Name() string { return "container" }

func (m *containerMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	re := buildRemoteExec(cfg)
	containerName := cfg.ContainerName
	if containerName == "" {
		return nil, fmt.Errorf("container_name is required")
	}

	cmd := fmt.Sprintf(`docker inspect --format '{{.State.Running}}' %s 2>/dev/null`, containerName)
	out, err := execCommand(ctx, re, cmd)
	if err != nil || strings.TrimSpace(out) != "true" {
		return &Result{Status: "down", Message: fmt.Sprintf("Container %s not running", containerName), Latency: 0, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: fmt.Sprintf("Container %s running", containerName), Latency: 0, Timestamp: time.Now()}, nil
}
