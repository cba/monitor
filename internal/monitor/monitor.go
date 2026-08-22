package monitor

import (
	"context"
	"time"
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
	// Name returns the monitor type identifier (e.g. "http", "tcp").
	Name() string

	// Check probes the target and returns a Result.
	// An error is only returned for internal failures, not target-down.
	Check(ctx context.Context, target string) (*Result, error)
}
