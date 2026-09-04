package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStore persists daily statistics as JSON files.
type FileStore struct {
	mu      sync.RWMutex
	dataDir string
}

// NewFileStore creates a FileStore with the given data directory.
func NewFileStore(dataDir string) (*FileStore, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return &FileStore{dataDir: dataDir}, nil
}

type fileData struct {
	Results map[string]*fileResult `json:"results"`
}

type fileResult struct {
	TotalChecks  int           `json:"total_checks"`
	UpCount      int           `json:"up_count"`
	DownCount    int           `json:"down_count"`
	WarningCount int           `json:"warning_count"`
	TotalLatency time.Duration `json:"total_latency"`
}

func (s *FileStore) filePath(date string) string {
	return filepath.Join(s.dataDir, date+".json")
}

func (s *FileStore) loadDate(date string) (*fileData, error) {
	path := s.filePath(date)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &fileData{Results: make(map[string]*fileResult)}, nil
		}
		return nil, err
	}
	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		return nil, err
	}
	if fd.Results == nil {
		fd.Results = make(map[string]*fileResult)
	}
	return &fd, nil
}

func (s *FileStore) saveDate(date string, fd *fileData) error {
	data, err := json.MarshalIndent(fd, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath(date), data, 0644)
}

// SaveResult records a single check result.
func (s *FileStore) SaveResult(name string, result *MonitorResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	date := result.Timestamp.Format("2006-01-02")
	fd, err := s.loadDate(date)
	if err != nil {
		return err
	}

	r, ok := fd.Results[name]
	if !ok {
		r = &fileResult{}
		fd.Results[name] = r
	}

	r.TotalChecks++
	switch result.Status {
	case "up":
		r.UpCount++
	case "down":
		r.DownCount++
	case "warning":
		r.WarningCount++
	}
	r.TotalLatency += result.Latency

	return s.saveDate(date, fd)
}

// GetDailyStats retrieves aggregated stats for a given date.
func (s *FileStore) GetDailyStats(date string) (*DailyStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fd, err := s.loadDate(date)
	if err != nil {
		return nil, err
	}

	stats := &DailyStats{
		Date:    date,
		Results: make(map[string]*MonitorStats),
	}

	for name, r := range fd.Results {
		stats.TotalChecks += r.TotalChecks
		stats.UpCount += r.UpCount
		stats.DownCount += r.DownCount
		stats.WarningCount += r.WarningCount

		var avg time.Duration
		if r.TotalChecks > 0 {
			avg = r.TotalLatency / time.Duration(r.TotalChecks)
		}

		stats.Results[name] = &MonitorStats{
			Name:         name,
			TotalChecks:  r.TotalChecks,
			UpCount:      r.UpCount,
			DownCount:    r.DownCount,
			WarningCount: r.WarningCount,
			AvgLatency:   avg,
		}
	}

	if stats.TotalChecks > 0 {
		totalLatency := time.Duration(0)
		for _, r := range fd.Results {
			totalLatency += r.TotalLatency
		}
		stats.AvgLatency = totalLatency / time.Duration(stats.TotalChecks)
	}

	return stats, nil
}
