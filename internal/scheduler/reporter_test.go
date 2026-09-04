package scheduler

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cba/monitor/internal/config"
	"github.com/cba/monitor/internal/reporter"
)

func TestParseCronNext(t *testing.T) {
	tests := []struct {
		expr   string
		expect string
	}{
		{"0 8 * * *", "08:00"},
		{"30 12 * * *", "12:30"},
		{"0 0 * * *", "00:00"},
		{"59 23 * * *", "23:59"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			next := parseCronNext(tt.expr)
			got := fmt.Sprintf("%02d:%02d", next.Hour(), next.Minute())
			if got != tt.expect {
				t.Errorf("parseCronNext(%q) = %s, want %s", tt.expr, got, tt.expect)
			}
		})
	}
}

func TestReporterGenerateAndSend(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "reporter-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create store
	store, err := reporter.NewFileStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Record some test results
	now := time.Now()
	date := now.Format("2006-01-02")

	for i := 0; i < 10; i++ {
		status := "up"
		if i%5 == 0 {
			status = "down"
		}
		mr := &reporter.MonitorResult{
			Name:      "test-monitor",
			Status:    status,
			Latency:   time.Duration(100+i*10) * time.Millisecond,
			Timestamp: now,
		}
		if err := store.SaveResult("test-monitor", mr); err != nil {
			t.Fatal(err)
		}
	}

	// Get stats
	stats, err := store.GetDailyStats(date)
	if err != nil {
		t.Fatal(err)
	}

	// Verify stats
	if stats.TotalChecks != 10 {
		t.Errorf("TotalChecks = %d, want 10", stats.TotalChecks)
	}
	if stats.UpCount != 8 {
		t.Errorf("UpCount = %d, want 8", stats.UpCount)
	}
	if stats.DownCount != 2 {
		t.Errorf("DownCount = %d, want 2", stats.DownCount)
	}

	// Render report
	content := reporter.RenderReport(stats, "测试日报")
	if len(content) == 0 {
		t.Error("RenderReport returned empty content")
	}

	t.Logf("Generated report:\n%s", content)
}

func TestSchedulerIntegration(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "scheduler-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create store and reporter
	store, err := reporter.NewFileStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	r := reporter.New(store)

	// Create scheduler
	s := New(r)

	// Create test config
	cfg := &config.Config{
		Monitors: []config.MonitorConfig{},
		Notifiers: []config.NotifierConfig{
			{
				Name:    "test-notifier",
				Type:    "wechat",
				Webhook: "https://test.webhook",
				Enabled: false, // Disable to avoid actual HTTP calls
			},
		},
		Reporter: &config.ReporterConfig{
			Enabled: true,
			Cron:    "0 8 * * *",
			Title:   "测试日报",
		},
	}

	// Reload
	s.Reload(cfg)

	// Verify reporter config is set
	s.mu.RLock()
	if s.reportConfig == nil {
		t.Error("reportConfig not set after Reload")
	}
	s.mu.RUnlock()

	// Test parseCronNext for current time
	next := parseCronNext("0 8 * * *")
	if next.Hour() != 8 || next.Minute() != 0 {
		t.Errorf("parseCronNext returned %s, want 08:00", next.Format("15:04"))
	}

	t.Logf("Next run time: %s", next.Format("2006-01-02 15:04:05"))
}
