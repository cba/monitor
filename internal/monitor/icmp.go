package monitor

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
)

type icmpMonitor struct{}

func init() {
	Register(&icmpMonitor{})
}

func (m *icmpMonitor) Name() string { return "icmp" }

func (m *icmpMonitor) Check(ctx context.Context, target string) (*Result, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", "3000", target)
	} else {
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "3", target)
	}

	start := time.Now()
	output, err := cmd.CombinedOutput()
	latency := time.Since(start)
	if err != nil {
		msg := string(output)
		if runtime.GOOS == "windows" {
			if decoded, dErr := simplifiedchinese.GBK.NewDecoder().Bytes(output); dErr == nil {
				msg = string(decoded)
			}
		}
		return &Result{Status: "down", Message: fmt.Sprintf("ping failed: %s", msg), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: "Ping successful", Latency: latency, Timestamp: time.Now()}, nil
}
