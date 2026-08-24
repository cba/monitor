package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/cba/monitor/internal/config"
)

type sslMonitor struct{}

func init() {
	Register(&sslMonitor{})
}

func (m *sslMonitor) Name() string { return "ssl" }

func (m *sslMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	host := cfg.Host
	port := cfg.Port
	if port == "" {
		port = "443"
	}
	addr := net.JoinHostPort(host, port)

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	start := time.Now()
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
	})
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}
	defer conn.Close()

	cert := conn.ConnectionState().PeerCertificates[0]
	daysLeft := time.Until(cert.NotAfter).Hours() / 24

	if daysLeft < 7 {
		return &Result{Status: "warning", Message: fmt.Sprintf("SSL cert expires in %d days (%s)", int(daysLeft), cert.NotAfter.Format("2006-01-02")), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: fmt.Sprintf("SSL cert valid for %d days", int(daysLeft)), Latency: latency, Timestamp: time.Now()}, nil
}
