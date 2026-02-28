// Package filter 提供过滤结果的元数据记录功能
package filter

import (
	"sync"
	"time"

	"github.com/log-system/log-processor/pkg/config"
)

// FilterMetadata 过滤元数据
type FilterMetadata struct {
	mu           sync.RWMutex
	entries      []MetadataEntry
	stats        FilterStats
	lastReset    time.Time
}

// MetadataEntry 单个元数据条目
type MetadataEntry struct {
	Timestamp   time.Time
	Service     string
	ServiceID   string
	Action      string
	RuleName    string
	FilterID    string
	OriginalSize int
	Kept        bool
}

// FilterStats 过滤统计
type FilterStats struct {
	TotalProcessed int64
	TotalKept      int64
	TotalDropped   int64
	TotalMarked    int64
	LastUpdated    time.Time
}

// NewFilterMetadata 创建过滤元数据管理器
func NewFilterMetadata() *FilterMetadata {
	return &FilterMetadata{
		entries:   make([]MetadataEntry, 0),
		lastReset: time.Now(),
		stats: FilterStats{
			LastUpdated: time.Now(),
		},
	}
}

// Record 记录过滤结果
func (m *FilterMetadata) Record(entry *ParsedLog, result FilterResult, filterID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metadata := MetadataEntry{
		Timestamp: time.Now(),
		Service:   entry.Service,
		Action:    result.Action.String(),
		RuleName:  result.MatchedRule,
		FilterID:  filterID,
		Kept:      result.ShouldKeep,
	}

	m.entries = append(m.entries, metadata)
	m.updateStats(result)
}

// updateStats 更新统计信息
func (m *FilterMetadata) updateStats(result FilterResult) {
	m.stats.TotalProcessed++
	m.stats.LastUpdated = time.Now()

	if result.ShouldKeep {
		m.stats.TotalKept++
	} else {
		m.stats.TotalDropped++
	}

	if result.Action == config.ActionMark {
		m.stats.TotalMarked++
	}
}

// GetStats 获取统计信息
func (m *FilterMetadata) GetStats() FilterStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// GetRecentEntries 获取最近的条目
func (m *FilterMetadata) GetRecentEntries(limit int) []MetadataEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit > len(m.entries) {
		limit = len(m.entries)
	}

	start := len(m.entries) - limit
	if start < 0 {
		start = 0
	}

	return m.entries[start:]
}

// GetStatsByService 按服务获取统计
func (m *FilterMetadata) GetStatsByService() map[string]*ServiceStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]*ServiceStats)
	for _, entry := range m.entries {
		if _, ok := stats[entry.Service]; !ok {
			stats[entry.Service] = &ServiceStats{
				Service: entry.Service,
			}
		}
		s := stats[entry.Service]
		s.TotalProcessed++
		if entry.Kept {
			s.TotalKept++
		} else {
			s.TotalDropped++
		}
	}

	return stats
}

// ServiceStats 服务统计
type ServiceStats struct {
	Service      string
	TotalProcessed int64
	TotalKept    int64
	TotalDropped int64
	DropRate     float64
}

// Clear 清空元数据
func (m *FilterMetadata) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = make([]MetadataEntry, 0)
	m.stats = FilterStats{
		LastUpdated: time.Now(),
	}
	m.lastReset = time.Now()
}

// GetLastReset 获取上次清空时间
func (m *FilterMetadata) GetLastReset() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastReset
}

// ExportMetrics 导出指标（用于 Prometheus 等）
func (m *FilterMetadata) ExportMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_processed": m.stats.TotalProcessed,
		"total_kept":      m.stats.TotalKept,
		"total_dropped":   m.stats.TotalDropped,
		"total_marked":    m.stats.TotalMarked,
		"last_updated":    m.stats.LastUpdated.Unix(),
		"last_reset":      m.lastReset.Unix(),
	}
}
