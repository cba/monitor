package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

type sslMonitor struct{}

func init() {
	Register(&sslMonitor{})
}

func (m *sslMonitor) Name() string { return "ssl" }

func (m *sslMonitor) Check(ctx context.Context, target string) (*Result, error) {
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimRight(target, "/")

	host, port, err := net.SplitHostPort(target)
	if err != nil {
		host = target
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

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return &Result{Status: "down", Message: "no certificate", Latency: latency, Timestamp: time.Now()}, nil
	}

	cert := state.PeerCertificates[0]
	expires := time.Until(cert.NotAfter)
	if expires < 0 {
		return &Result{Status: "down", Message: fmt.Sprintf("certificate expired %v ago", -expires), Latency: latency, Timestamp: time.Now()}, nil
	}
	if expires < 7*24*time.Hour {
		return &Result{Status: "warning", Message: fmt.Sprintf("certificate expires in %v", expires), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: fmt.Sprintf("valid until %s", cert.NotAfter.Format("2006-01-02")), Latency: latency, Timestamp: time.Now()}, nil
}
