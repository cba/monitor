package scheduler

import (
	"context"
	"testing"

	"github.com/cba/monitor/internal/config"
	"github.com/cba/monitor/internal/monitor"
)

// 恢复流程：down 报警后 wasDown 为真并触发恢复；恢复通知后状态清除，
// 再次 up 不重复发恢复。notifiers 为空，通知发送为无操作。
func TestRecoveryAfterDown(t *testing.T) {
	s := New(nil)
	m := &config.MonitorConfig{Name: "mem-test", Type: "memory", AlertInterval: 3600}

	if s.wasDown(m.Name) {
		t.Fatal("wasDown should be false initially")
	}
	if !s.shouldAlert(m.Name, m.AlertInterval) {
		t.Fatal("first failure should alert")
	}
	s.sendAlert(context.Background(), m, &monitor.Result{Status: "down"})

	if !s.wasDown(m.Name) {
		t.Fatal("wasDown should be true after down alert")
	}
	s.sendRecovery(context.Background(), m, &monitor.Result{Status: "up"})

	if s.wasDown(m.Name) {
		t.Fatal("recovery should clear alert state")
	}
	if !s.shouldAlert(m.Name, m.AlertInterval) {
		t.Fatal("state cleared, next failure should alert immediately")
	}
}
