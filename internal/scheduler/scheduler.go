package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cba/monitor/internal/config"
	"github.com/cba/monitor/internal/monitor"
	"github.com/cba/monitor/internal/notifier"
	"github.com/cba/monitor/internal/reporter"
)

// Scheduler runs monitoring goroutines.
type Scheduler struct {
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	monitors   map[string]*monitorRunner
	notifiers  []config.NotifierConfig
	notifierMu sync.RWMutex

	// Alert state tracking (in-memory)
	alertState map[string]*alertInfo
	alertMu    sync.Mutex

	// Reporter for daily reports
	reporter        *reporter.Reporter
	reportConfig    *config.ReporterConfig
	reporterRunning bool
}

type monitorRunner struct {
	config *config.MonitorConfig
	cancel context.CancelFunc
}

type alertInfo struct {
	lastAlertAt time.Time
	alertCount  int
}

// New creates a Scheduler.
func New(r *reporter.Reporter) *Scheduler {
	return &Scheduler{
		monitors:   make(map[string]*monitorRunner),
		alertState: make(map[string]*alertInfo),
		reporter:   r,
	}
}

// Start begins monitoring with the given context.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()
	<-s.ctx.Done()
}

// StartWithContext begins monitoring and signals when context is ready.
func (s *Scheduler) StartWithContext(ctx context.Context, ready chan struct{}) {
	s.mu.Lock()
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()
	close(ready) // Signal that context is ready

	<-s.ctx.Done()
}

// Stop halts all monitoring goroutines.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, r := range s.monitors {
		r.cancel()
		delete(s.monitors, name)
	}
}

// Reload updates the monitors based on new config.
// Stops removed monitors, starts new ones, keeps unchanged ones.
func (s *Scheduler) Reload(cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update notifiers
	s.notifierMu.Lock()
	s.notifiers = cfg.Notifiers
	s.notifierMu.Unlock()

	// Update reporter config
	if s.reporter != nil {
		s.reporter.UpdateNotifiers(cfg.Notifiers)
		s.reportConfig = cfg.Reporter
		if s.ctx != nil {
			s.startReporterScheduler(s.ctx)
		}
	}

	// Build new monitor set
	newSet := make(map[string]*config.MonitorConfig)
	for i := range cfg.Monitors {
		m := &cfg.Monitors[i]
		if m.Enabled == nil || *m.Enabled {
			newSet[m.Name] = m
		}
	}

	// Stop monitors that are no longer in config or disabled
	for name, r := range s.monitors {
		if _, ok := newSet[name]; !ok {
			log.Printf("[%s] %s stopping (removed from config)", r.config.Type, name)
			r.cancel()
			delete(s.monitors, name)
		}
	}

	// Start new monitors or update existing ones
	for name, m := range newSet {
		if r, ok := s.monitors[name]; ok {
			// Monitor exists - check if config changed
			boolChanged := func(a, b *bool) bool {
				if a == nil && b == nil {
					return false
				}
				if a == nil || b == nil {
					return true
				}
				return *a != *b
			}
			if r.config.Type == m.Type &&
				r.config.URL == m.URL &&
				r.config.Host == m.Host &&
				r.config.Port == m.Port &&
				r.config.Keyword == m.Keyword &&
				r.config.DSN == m.DSN &&
				r.config.Path == m.Path &&
				r.config.ProcessName == m.ProcessName &&
				r.config.ContainerName == m.ContainerName &&
				r.config.Interval == m.Interval &&
				r.config.AlertInterval == m.AlertInterval &&
				!boolChanged(r.config.Enabled, m.Enabled) {
				continue // No change, skip
			}
			// Config changed, restart
			log.Printf("[%s] %s restarting (config changed)", m.Type, name)
			r.cancel()
			delete(s.monitors, name)
		}

		// Start new monitor
		if s.ctx == nil {
			continue // Context not set yet, skip starting monitors
		}
		monCtx, monCancel := context.WithCancel(s.ctx)
		r := &monitorRunner{config: m, cancel: monCancel}
		s.monitors[name] = r
		go s.runMonitor(monCtx, m)
		log.Printf("[%s] %s started (check=%ds, alert=%ds)", m.Type, m.Name, m.Interval, m.AlertInterval)
	}

	log.Printf("scheduler: %d monitors active", len(s.monitors))
}

// startReporterScheduler starts the daily report scheduler if enabled. Called with s.mu held.
func (s *Scheduler) startReporterScheduler(ctx context.Context) {
	if s.reporter == nil || s.reportConfig == nil || !s.reportConfig.Enabled || s.reporterRunning {
		return
	}
	s.reporterRunning = true

	cronExpr := s.reportConfig.Cron
	if cronExpr == "" {
		log.Println("reporter: cron expression not set, skipping")
		return
	}

	go func() {
		for {
			nextRun := parseCronNext(cronExpr)
			if nextRun.IsZero() {
				log.Printf("reporter: invalid cron expression: %s", cronExpr)
				return
			}

			timer := time.NewTimer(time.Until(nextRun))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				s.sendDailyReport(ctx)
			}
		}
	}()

	log.Printf("reporter: scheduler started (cron=%s)", cronExpr)
}

func (s *Scheduler) sendDailyReport(ctx context.Context) {
	title := s.reportConfig.Title
	if title == "" {
		title = "每日监控日报"
	}

	date := time.Now().Format("2006-01-02")
	log.Printf("reporter: generating daily report for %s", date)

	if err := s.reporter.GenerateAndSend(ctx, date, title); err != nil {
		log.Printf("reporter: send daily report: %v", err)
	} else {
		log.Printf("reporter: daily report sent")
	}
}

func parseCronNext(expr string) time.Time {
	// Simple cron parser: only support "mm hh * * *" format
	// Example: "0 8 * * *" = every day at 08:00
	parts := splitCron(expr)
	if len(parts) < 5 {
		return time.Time{}
	}

	now := time.Now()
	year, month, day := now.Date()
	hour := now.Hour()
	min := now.Minute()

	// Parse cron fields
	cronMin := -1
	cronHour := -1
	if parts[0] != "*" {
		cronMin = parseInt(parts[0])
	}
	if parts[1] != "*" {
		cronHour = parseInt(parts[1])
	}

	// Calculate next run time
	nextMin := 0
	nextHour := 0

	if cronMin >= 0 && cronHour >= 0 {
		// Both specified: check if today is still possible
		if hour < cronHour || (hour == cronHour && min < cronMin) {
			nextHour = cronHour
			nextMin = cronMin
		} else {
			// Tomorrow
			nextHour = cronHour
			nextMin = cronMin
			day++
		}
	} else if cronHour >= 0 {
		// Only hour specified
		if hour < cronHour {
			nextHour = cronHour
			nextMin = 0
		} else if hour == cronHour {
			// Same hour, next minute 0
			nextHour = cronHour
			nextMin = 0
			day++ // Tomorrow same hour
		} else {
			// Tomorrow
			nextHour = cronHour
			nextMin = 0
			day++
		}
	} else if cronMin >= 0 {
		// Only minute specified: every hour at this minute
		if min < cronMin {
			nextHour = hour
			nextMin = cronMin
		} else {
			nextHour = hour + 1
			nextMin = cronMin
			if nextHour > 23 {
				nextHour = 0
				day++
			}
		}
	} else {
		// Both wildcard: run every minute
		nextHour = hour
		nextMin = min + 1
		if nextMin > 59 {
			nextMin = 0
			nextHour++
			if nextHour > 23 {
				nextHour = 0
				day++
			}
		}
	}

	return time.Date(year, month, day, nextHour, nextMin, 0, 0, now.Location())
}

func splitCron(expr string) []string {
	var parts []string
	for _, p := range splitSpaces(expr) {
		parts = append(parts, p)
	}
	for len(parts) < 5 {
		parts = append(parts, "*")
	}
	return parts
}

func splitSpaces(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return -1
		}
	}
	return n
}

func (s *Scheduler) runMonitor(ctx context.Context, m *config.MonitorConfig) {
	ticker := time.NewTicker(time.Duration(m.Interval) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	s.checkAndAlert(ctx, m)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAndAlert(ctx, m)
		}
	}
}

func (s *Scheduler) checkAndAlert(ctx context.Context, m *config.MonitorConfig) {
	mon, err := monitor.Get(m.Type)
	if err != nil {
		log.Printf("[%s] %s: unknown monitor type: %v", m.Type, m.Name, err)
		return
	}

	result, err := mon.Check(ctx, m)
	if err != nil {
		log.Printf("[%s] %s: check error: %v", m.Type, m.Name, err)
		return
	}

	log.Printf("[%s] %s: %s - %s (%v)", m.Type, m.Name, result.Status, result.Message, result.Latency)

	// Record to reporter if enabled
	if s.reporter != nil && s.reportConfig != nil && s.reportConfig.Enabled {
		mr := &reporter.MonitorResult{
			Name:      m.Name,
			Status:    result.Status,
			Latency:   result.Latency,
			Timestamp: result.Timestamp,
		}
		if err := s.reporter.RecordCheck(m.Name, mr); err != nil {
			log.Printf("reporter: record check: %v", err)
		}
	}

	if result.Status == "down" || result.Status == "warning" {
		if s.shouldAlert(m.Name, m.AlertInterval) {
			s.sendAlert(ctx, m, result)
		}
	} else {
		if s.wasDown(m.Name) {
			s.sendRecovery(ctx, m, result)
		}
	}
}

func (s *Scheduler) shouldAlert(key string, alertInterval int) bool {
	s.alertMu.Lock()
	defer s.alertMu.Unlock()

	state, ok := s.alertState[key]
	if !ok || state.lastAlertAt.IsZero() {
		return true
	}
	elapsed := time.Since(state.lastAlertAt)
	return elapsed >= time.Duration(alertInterval)*time.Second
}

func (s *Scheduler) wasDown(key string) bool {
	s.alertMu.Lock()
	defer s.alertMu.Unlock()

	state, ok := s.alertState[key]
	if !ok {
		return false
	}
	return state.alertCount > 0
}

func (s *Scheduler) sendAlert(ctx context.Context, m *config.MonitorConfig, result *monitor.Result) {
	s.alertMu.Lock()
	state, ok := s.alertState[m.Name]
	if !ok {
		state = &alertInfo{}
		s.alertState[m.Name] = state
	}
	state.lastAlertAt = time.Now()
	state.alertCount++
	s.alertMu.Unlock()

	title := fmt.Sprintf("🔴 监控报警: %s", m.Name)
	content := fmt.Sprintf("🔴 监控报警\n名称：%s\n类型：%s\n状态：%s\n时间：%s\n详情：%s",
		m.Name, m.Type, result.Status, time.Now().Format("2006-01-02 15:04:05"), result.Message)

	s.notifyAll(ctx, title, content)
}

func (s *Scheduler) sendRecovery(ctx context.Context, m *config.MonitorConfig, result *monitor.Result) {
	s.alertMu.Lock()
	delete(s.alertState, m.Name)
	s.alertMu.Unlock()

	title := fmt.Sprintf("🟢 监控恢复: %s", m.Name)
	content := fmt.Sprintf("🟢 监控恢复\n名称：%s\n类型：%s\n状态：up\n时间：%s\n详情：%s",
		m.Name, m.Type, time.Now().Format("2006-01-02 15:04:05"), result.Message)

	s.notifyAll(ctx, title, content)
}

func (s *Scheduler) notifyAll(ctx context.Context, title, content string) {
	s.notifierMu.RLock()
	notifiers := s.notifiers
	s.notifierMu.RUnlock()

	for _, nc := range notifiers {
		if nc.Enabled != nil && !*nc.Enabled {
			continue
		}

		var n notifier.Notifier
		switch nc.Type {
		case "wechat":
			n = notifier.NewWeChatNotifier(nc.Webhook)
		case "wechat_app":
			n = notifier.NewWeChatAppNotifier(nc.CorpID, nc.AgentID, nc.Secret, nc.ToUsers)
		case "dingtalk":
			n = notifier.NewDingTalkNotifier(nc.Webhook)
		default:
			log.Printf("notifier %s: unknown type %s", nc.Name, nc.Type)
			continue
		}

		if err := n.Send(ctx, title, content); err != nil {
			log.Printf("send to %s (%s): %v", nc.Name, nc.Type, err)
		} else {
			log.Printf("sent to %s (%s)", nc.Name, nc.Type)
		}
	}
}
