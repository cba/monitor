package reporter

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/cba/monitor/internal/config"
	"github.com/cba/monitor/internal/notifier"
)

// MonitorResult holds check result for reporting.
type MonitorResult struct {
	Name      string
	Status    string // "up", "down", "warning"
	Latency   time.Duration
	Timestamp time.Time
}

// DailyStats holds aggregated daily statistics.
type DailyStats struct {
	Date        string
	TotalChecks int
	UpCount     int
	DownCount   int
	WarningCount int
	AvgLatency  time.Duration
	Results     map[string]*MonitorStats
}

// MonitorStats holds per-monitor statistics.
type MonitorStats struct {
	Name        string
	TotalChecks int
	UpCount     int
	DownCount   int
	WarningCount int
	AvgLatency  time.Duration
}

// Reporter generates and sends daily reports.
type Reporter struct {
	mu        sync.RWMutex
	store     Store
	notifiers []config.NotifierConfig
}

// Store defines the interface for persisting statistics.
type Store interface {
	SaveResult(name string, result *MonitorResult) error
	GetDailyStats(date string) (*DailyStats, error)
}

// New creates a Reporter with the given store.
func New(store Store) *Reporter {
	return &Reporter{
		store: store,
	}
}

// RecordCheck records a monitoring check result.
func (r *Reporter) RecordCheck(name string, result *MonitorResult) error {
	return r.store.SaveResult(name, result)
}

// UpdateNotifiers updates the notifier configuration.
func (r *Reporter) UpdateNotifiers(notifiers []config.NotifierConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifiers = notifiers
}

// GenerateAndSend generates the daily report and sends via all notifiers.
func (r *Reporter) GenerateAndSend(ctx context.Context, date string, title string) error {
	stats, err := r.store.GetDailyStats(date)
	if err != nil {
		return err
	}

	content := RenderReport(stats, title)

	r.mu.RLock()
	notifiers := r.notifiers
	r.mu.RUnlock()

	for _, nc := range notifiers {
		if !nc.Enabled {
			continue
		}

		var n notifier.Notifier
		switch nc.Type {
		case "wechat":
			n = notifier.NewWeChatNotifier(nc.Webhook)
		case "wechat_app":
			n = notifier.NewWeChatAppNotifier(
				nc.Extra["corp_id"],
				nc.Extra["agent_id"],
				nc.Extra["secret"],
				nc.Extra["to_users"],
			)
		case "dingtalk":
			n = notifier.NewDingTalkNotifier(nc.Webhook)
		default:
			log.Printf("reporter: unknown notifier type %s", nc.Type)
			continue
		}

		if err := n.Send(ctx, title, content); err != nil {
			log.Printf("reporter: send to %s (%s): %v", nc.Name, nc.Type, err)
		} else {
			log.Printf("reporter: sent to %s (%s)", nc.Name, nc.Type)
		}
	}

	return nil
}
