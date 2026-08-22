package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type containerMonitor struct{}

func init() {
	Register(&containerMonitor{})
}

func (m *containerMonitor) Name() string { return "container" }

func (m *containerMonitor) Check(ctx context.Context, target string) (*Result, error) {
	re, rest := parseTarget(target)

	var containerName string
	if parts := strings.Split(rest, ","); len(parts) >= 2 && parts[0] == "container" {
		containerName = parts[1]
	}
	if containerName == "" {
		return nil, fmt.Errorf("container target must include container name, e.g. 'container,my-app'")
	}

	cmd := fmt.Sprintf(`docker inspect --format '{{.State.Running}}' %s 2>/dev/null`, containerName)
	start := time.Now()
	out, err := execCommand(ctx, re, cmd)
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: fmt.Sprintf("container '%s' not found or docker error: %s", containerName, err), Latency: latency, Timestamp: time.Now()}, nil
	}

	running := strings.TrimSpace(out)
	if running == "true" {
		return &Result{Status: "up", Message: fmt.Sprintf("container '%s' running", containerName), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "down", Message: fmt.Sprintf("container '%s' not running (state: %s)", containerName, running), Latency: latency, Timestamp: time.Now()}, nil
}
