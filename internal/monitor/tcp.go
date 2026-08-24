package monitor

import (
	"context"
	"net"
	"time"

	"github.com/cba/monitor/internal/config"
)

type tcpMonitor struct{}

func init() {
	Register(&tcpMonitor{})
}

func (m *tcpMonitor) Name() string { return "tcp" }

func (m *tcpMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	host := cfg.Host
	if cfg.Port != "" {
		host = net.JoinHostPort(cfg.Host, cfg.Port)
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", host)
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}
	conn.Close()
	return &Result{Status: "up", Message: "TCP connect OK", Latency: latency, Timestamp: time.Now()}, nil
}
