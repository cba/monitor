package config

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors config file changes and triggers reload.
type Watcher struct {
	path    string
	mu      sync.RWMutex
	cfg     *Config
	onChange func(*Config)
	watcher *fsnotify.Watcher
	stop    chan struct{}
}

// NewWatcher creates a config watcher with initial config.
func NewWatcher(path string, cfg *Config, onChange func(*Config)) *Watcher {
	return &Watcher{
		path:     path,
		cfg:      cfg,
		onChange: onChange,
		stop:     make(chan struct{}),
	}
}

// Start begins watching the config file.
func (w *Watcher) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	w.watcher = watcher

	if err := watcher.Add(filepath.Dir(w.path)); err != nil {
		watcher.Close()
		return fmt.Errorf("watch dir: %w", err)
	}

	go w.loop()
	log.Printf("config watcher started: %s", w.path)
	return nil
}

// Stop stops the watcher.
func (w *Watcher) Stop() {
	close(w.stop)
	if w.watcher != nil {
		w.watcher.Close()
	}
}

// Config returns the current config (thread-safe).
func (w *Watcher) Config() *Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cfg
}

func (w *Watcher) loop() {
	defer w.watcher.Close()

	for {
		select {
		case <-w.stop:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Name == w.path && event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				w.reload()
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("config watcher error: %v", err)
		}
	}
}

func (w *Watcher) reload() {
	// Re-read and parse the config
	newCfg, err := Load(w.path)
	if err != nil {
		log.Printf("config reload failed (keeping old config): %v", err)
		return
	}

	// Validate: check if at least one monitor exists
	if len(newCfg.Monitors) == 0 {
		log.Printf("config reload failed: no monitors defined (keeping old config)")
		return
	}

	// Apply the new config
	w.mu.Lock()
	w.cfg = newCfg
	w.mu.Unlock()

	log.Printf("config reloaded: %d monitors, %d notifiers", len(newCfg.Monitors), len(newCfg.Notifiers))

	if w.onChange != nil {
		w.onChange(newCfg)
	}
}
