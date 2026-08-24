package monitor

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/cba/monitor/internal/config"
	"github.com/redis/go-redis/v9"
)

type redisMonitor struct{}

func init() {
	Register(&redisMonitor{})
}

func (m *redisMonitor) Name() string { return "redis" }

func (m *redisMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	re := buildRemoteExec(cfg)

	opts := &redis.Options{Addr: addr, Password: cfg.Password}

	if re.User != "" {
		sshClient, err := newSSHClient(ctx, re)
		if err != nil {
			return nil, fmt.Errorf("redis ssh: %w", err)
		}
		defer sshClient.Close()

		opts.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return sshDial(sshClient, cfg.Host, cfg.Port)
		}
		opts.Addr = ""
	}

	client := redis.NewClient(opts)
	defer client.Close()

	start := time.Now()
	_, err := client.Ping(ctx).Result()
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: "Redis ping OK", Latency: latency, Timestamp: time.Now()}, nil
}
