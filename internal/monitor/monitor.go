package monitor

import (
	"context"
	"time"

	"github.com/cba/monitor/internal/config"
)

// Result holds the outcome of a monitoring check.
type Result struct {
	Status    string        // "up", "down", "warning"
	Message   string        // Human-readable detail
	Latency   time.Duration // Check latency
	Timestamp time.Time     // When the check ran
}

// Monitor checks the health of a target.
type Monitor interface {
	Name() string
	Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error)
}
