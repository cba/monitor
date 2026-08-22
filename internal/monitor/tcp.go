package monitor

import (
	"context"
	"net"
	"time"
)

type tcpMonitor struct{}

func init() {
	Register(&tcpMonitor{})
}

func (m *tcpMonitor) Name() string { return "tcp" }

func (m *tcpMonitor) Check(ctx context.Context, target string) (*Result, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", target)
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}
	conn.Close()
	return &Result{Status: "up", Message: "Connection successful", Latency: latency, Timestamp: time.Now()}, nil
}
