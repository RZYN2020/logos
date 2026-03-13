// Package analysis 分析模块测试
package analysis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestExtractPattern 测试模式提取
func TestExtractPattern(t *testing.T) {
	miner := NewPatternMiner()

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "without variables",
			message:  "User logged in successfully",
			expected: "User logged in successfully",
		},
		{
			name:     "with number",
			message:  "Request processed in 100ms",
			expected: "Request processed in 100ms",
		},
		{
			name:     "with multiple numbers",
			message:  "Connection from 192.168.1.1 port 8080",
			expected: "Connection from 192.168.1.1 port 8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := miner.ExtractPattern(tt.message)
			assert.NotEmpty(t, pattern)
		})
	}
}

// TestAnalyzePatterns 测试模式分析
func TestAnalyzePatterns(t *testing.T) {
	miner := NewPatternMiner()

	entries := []LogEntry{
		{Timestamp: time.Now(), Level: "INFO", Service: "api", Message: "User logged in"},
		{Timestamp: time.Now(), Level: "INFO", Service: "api", Message: "User logged in"},
		{Timestamp: time.Now(), Level: "INFO", Service: "api", Message: "User logged in"},
		{Timestamp: time.Now(), Level: "ERROR", Service: "api", Message: "Connection failed"},
		{Timestamp: time.Now(), Level: "ERROR", Service: "api", Message: "Connection failed"},
		{Timestamp: time.Now(), Level: "WARN", Service: "web", Message: "Slow response"},
	}

	patterns := miner.AnalyzePatterns(entries)

	assert.NotEmpty(t, patterns)
	assert.Greater(t, len(patterns), 0)

	// 验证频率统计
	totalFreq := 0
	for _, p := range patterns {
		totalFreq += p.Frequency
	}
	assert.Equal(t, len(entries), totalFreq)
}

// TestDetectAnomalies 测试异常检测
func TestDetectAnomalies(t *testing.T) {
	miner := NewPatternMiner()

	baseline := []LogEntry{
		{Timestamp: time.Now(), Level: "INFO", Message: "Normal operation"},
		{Timestamp: time.Now(), Level: "INFO", Message: "Normal operation"},
	}

	current := []LogEntry{
		{Timestamp: time.Now(), Level: "INFO", Message: "Normal operation"},
		{Timestamp: time.Now(), Level: "ERROR", Message: "New error pattern"},
	}

	anomalies := miner.DetectAnomalies(current, baseline)

	// 新错误模式应该被检测为异常
	assert.Greater(t, len(anomalies), 0)
}

// TestClusterLogs 测试日志聚类
func TestClusterLogs(t *testing.T) {
	miner := NewPatternMiner()

	entries := []LogEntry{
		{Timestamp: time.Now(), Level: "INFO", Message: "Same message"},
		{Timestamp: time.Now(), Level: "INFO", Message: "Same message"},
		{Timestamp: time.Now(), Level: "INFO", Message: "Same message"},
		{Timestamp: time.Now(), Level: "ERROR", Message: "Different message"},
		{Timestamp: time.Now(), Level: "ERROR", Message: "Different message"},
	}

	clusters := miner.ClusterLogs(entries, 0.8)

	assert.NotEmpty(t, clusters)

	// 验证聚类结果
	totalSize := 0
	for _, c := range clusters {
		totalSize += c.Size
	}
	assert.Equal(t, len(entries), totalSize)
}

// TestRecommendRules 测试规则推荐
func TestRecommendRules(t *testing.T) {
	miner := NewPatternMiner()

	// 创建高频模式（超过 50 条以触发 medium severity）
	entries := make([]LogEntry, 100)
	for i := 0; i < 100; i++ {
		entries[i] = LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Service:   "api",
			Message:   "Frequent error message",
		}
	}

	patterns := miner.AnalyzePatterns(entries)
	recommendations := miner.RecommendRules(patterns)

	// 高频模式应该产生推荐
	assert.Greater(t, len(recommendations), 0)

	// 验证推荐内容
	for _, rec := range recommendations {
		assert.NotEmpty(t, rec.Name)
		assert.Greater(t, rec.Priority, 0)
		assert.NotEmpty(t, rec.Conditions)
	}
}

// TestGetPriorityFromSeverity 测试优先级计算
func TestGetPriorityFromSeverity(t *testing.T) {
	tests := []struct {
		severity string
		expected int
	}{
		{"high", 1},
		{"medium", 2},
		{"low", 3},
		{"unknown", 3},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			result := getPriorityFromSeverity(tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractKeywords 测试关键词提取
func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{
			name:     "with placeholders",
			pattern:  "Error {num}: connection failed at {timestamp}",
			expected: "Error : connection failed at",
		},
		{
			name:     "without placeholders",
			pattern:  "Simple error message",
			expected: "Simple error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractKeywords(tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}
