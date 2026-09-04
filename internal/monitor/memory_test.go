package monitor

import (
	"context"
	"testing"

	"github.com/cba/monitor/internal/config"
)

// 边界用例：>=crit 为 down，>=warn 为 warning，否则 up（恢复）。
// 通过注入 free -m 的解析输出模拟高内存，无需真正占用内存。
func TestMemoryCheckThresholds(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	cases := []struct {
		name string
		out  string
		want string
	}{
		{"恢复-低于warn", "3724 7435 50.1", "up"},
		{"边界-正好warn", "6320 7435 85.0", "warning"},
		{"介于warn与crit", "6700 7435 90.1", "warning"},
		{"边界-正好crit", "7063 7435 95.0", "down"},
		{"高于crit", "7300 7435 98.2", "down"},
	}

	cfg := &config.MonitorConfig{Warn: 85, Crit: 95}
	m := &memoryMonitor{}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := c.out
			execCommand = func(ctx context.Context, re remoteExec, cmd string) (string, error) {
				return out, nil
			}
			res, err := m.Check(context.Background(), cfg)
			if err != nil {
				t.Fatalf("check error: %v", err)
			}
			if res.Status != c.want {
				t.Fatalf("pct %s: status = %s, want %s", out, res.Status, c.want)
			}
		})
	}
}
