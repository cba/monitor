package reporter

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// RenderReport generates a text report from daily stats.
func RenderReport(stats *DailyStats, title string) string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("📊 %s\n", title))
	b.WriteString(fmt.Sprintf("📅 日期: %s\n", stats.Date))
	b.WriteString(strings.Repeat("─", 30) + "\n")

	// Summary
	availability := 0.0
	if stats.TotalChecks > 0 {
		availability = float64(stats.UpCount) / float64(stats.TotalChecks) * 100
	}

	b.WriteString(fmt.Sprintf("📈 概览\n"))
	b.WriteString(fmt.Sprintf("  总检查次数: %d\n", stats.TotalChecks))
	b.WriteString(fmt.Sprintf("  正常(UP): %d\n", stats.UpCount))
	b.WriteString(fmt.Sprintf("  故障(DOWN): %d\n", stats.DownCount))
	b.WriteString(fmt.Sprintf("  警告(WARNING): %d\n", stats.WarningCount))
	b.WriteString(fmt.Sprintf("  可用率: %.2f%%\n", availability))
	b.WriteString(fmt.Sprintf("  平均延迟: %s\n", formatDuration(stats.AvgLatency)))
	b.WriteString("\n")

	// Per-monitor details
	if len(stats.Results) > 0 {
		b.WriteString("📋 监控详情\n")

		// Sort by name
		names := make([]string, 0, len(stats.Results))
		for name := range stats.Results {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			s := stats.Results[name]
			avail := 0.0
			if s.TotalChecks > 0 {
				avail = float64(s.UpCount) / float64(s.TotalChecks) * 100
			}

			status := "✅"
			if s.DownCount > 0 {
				status = "🔴"
			} else if s.WarningCount > 0 {
				status = "🟡"
			}

			b.WriteString(fmt.Sprintf("  %s %s\n", status, s.Name))
			b.WriteString(fmt.Sprintf("    检查: %d次 | 可用率: %.2f%% | 延迟: %s\n",
				s.TotalChecks, avail, formatDuration(s.AvgLatency)))

			if s.DownCount > 0 {
				b.WriteString(fmt.Sprintf("    故障: %d次\n", s.DownCount))
			}
			if s.WarningCount > 0 {
				b.WriteString(fmt.Sprintf("    警告: %d次\n", s.WarningCount))
			}
		}
	}

	return b.String()
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
