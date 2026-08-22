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
func New() *Scheduler {
	return &Scheduler{
		monitors:   make(map[string]*monitorRunner),
		alertState: make(map[string]*alertInfo),
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

	// Build new monitor set
	newSet := make(map[string]*config.MonitorConfig)
	for i := range cfg.Monitors {
		m := &cfg.Monitors[i]
		if m.Enabled {
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
			if r.config.Type == m.Type && r.config.Target == m.Target &&
				r.config.Interval == m.Interval && r.config.AlertInterval == m.AlertInterval {
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

	result, err := mon.Check(ctx, m.Target)
	if err != nil {
		log.Printf("[%s] %s: check error: %v", m.Type, m.Name, err)
		return
	}

	log.Printf("[%s] %s: %s - %s (%v)", m.Type, m.Name, result.Status, result.Message, result.Latency)

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
	content := fmt.Sprintf("🔴 监控报警\n- 名称：%s\n- 类型：%s\n- 状态：%s\n- 时间：%s\n- 详情：%s",
		m.Name, m.Type, result.Status, time.Now().Format("2006-01-02 15:04:05"), result.Message)

	s.notifyAll(ctx, title, content)
}

func (s *Scheduler) sendRecovery(ctx context.Context, m *config.MonitorConfig, result *monitor.Result) {
	s.alertMu.Lock()
	delete(s.alertState, m.Name)
	s.alertMu.Unlock()

	title := fmt.Sprintf("🟢 监控恢复: %s", m.Name)
	content := fmt.Sprintf("🟢 监控恢复\n- 名称：%s\n- 类型：%s\n- 状态：up\n- 时间：%s\n- 详情：%s",
		m.Name, m.Type, time.Now().Format("2006-01-02 15:04:05"), result.Message)

	s.notifyAll(ctx, title, content)
}

func (s *Scheduler) notifyAll(ctx context.Context, title, content string) {
	s.notifierMu.RLock()
	notifiers := s.notifiers
	s.notifierMu.RUnlock()

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
