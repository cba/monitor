package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisMonitor struct{}

func init() {
	Register(&redisMonitor{})
}

func (m *redisMonitor) Name() string { return "redis" }

func (m *redisMonitor) Check(ctx context.Context, target string) (*Result, error) {
	opts, err := redis.ParseURL("redis://" + target)
	if err != nil {
		// If target doesn't look like a URL, treat as host:port
		opts = &redis.Options{Addr: target}
	}

	client := redis.NewClient(opts)
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	start := time.Now()
	result, err := client.Ping(ctx).Result()
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: fmt.Sprintf("PONG: %s", strings.TrimSpace(result)), Latency: latency, Timestamp: time.Now()}, nil
}
