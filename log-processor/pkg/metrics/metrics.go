// Package metrics 提供性能监控和统计功能
package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ParseTotal 解析日志总数
	ParseTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "log_processor_parse_total",
		Help: "The total number of parsed logs",
	}, []string{"status"}) // status: success, failure

	// ParseLatency 解析延迟
	ParseLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "log_processor_parse_latency_seconds",
		Help:    "Latency of log parsing in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// FilterTotal 过滤日志总数
	FilterTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "log_processor_filter_total",
		Help: "The total number of filtered logs",
	}, []string{"action"}) // action: kept, dropped

	// WriteTotal 写入存储总数
	WriteTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "log_processor_write_total",
		Help: "The total number of logs written to sink",
	}, []string{"status"}) // status: success, failure

	// WriteLatency 写入延迟
	WriteLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "log_processor_write_latency_seconds",
		Help:    "Latency of writing logs to sink in seconds",
		Buckets: prometheus.DefBuckets,
	})
)

// Metrics 性能指标 (Legacy for backward compatibility)
type Metrics struct {
	mu sync.RWMutex

	// 解析指标
	ParseCount      int64
	ParseSuccess    int64
	ParseFailure    int64
	ParseLatencySum time.Duration

	// 过滤指标
	FilterCount   int64
	FilterKept    int64
	FilterDropped int64

	// 转换指标
	TransformCount   int64
	TransformSuccess int64

	// 分析指标
	AnalyzeCount int64

	// 写入指标
	WriteCount      int64
	WriteSuccess    int64
	WriteFailure    int64
	WriteLatencySum time.Duration

	// 时间窗口
	windowStart time.Time
	windowSize  time.Duration
}

// LatencyStats 延迟统计
type LatencyStats struct {
	Count int64
	Avg   time.Duration
	Min   time.Duration
	Max   time.Duration
	P50   time.Duration
	P95   time.Duration
	P99   time.Duration
}

// ParserMetrics 解析器指标
type ParserMetrics struct {
	Format     string
	Count      int64
	Success    int64
	AvgLatency time.Duration
}

// FilterMetrics 过滤器指标
type FilterMetrics struct {
	RuleName   string
	MatchCount int64
	Action     string
}

// NewMetrics 创建性能指标
func NewMetrics() *Metrics {
	return &Metrics{
		windowStart: time.Now(),
		windowSize:  time.Minute,
	}
}

// RecordParse 记录解析指标
func (m *Metrics) RecordParse(success bool, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ParseCount++
	if success {
		m.ParseSuccess++
		ParseTotal.WithLabelValues("success").Inc()
	} else {
		m.ParseFailure++
		ParseTotal.WithLabelValues("failure").Inc()
	}
	m.ParseLatencySum += latency
	ParseLatency.Observe(latency.Seconds())
}

// RecordFilter 记录过滤指标
func (m *Metrics) RecordFilter(kept bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.FilterCount++
	if kept {
		m.FilterKept++
		FilterTotal.WithLabelValues("kept").Inc()
	} else {
		m.FilterDropped++
		FilterTotal.WithLabelValues("dropped").Inc()
	}
}

// RecordTransform 记录转换指标
func (m *Metrics) RecordTransform(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TransformCount++
	if success {
		m.TransformSuccess++
	}
}

// RecordAnalyze 记录分析指标
func (m *Metrics) RecordAnalyze() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.AnalyzeCount++
}

// RecordWrite 记录写入指标
func (m *Metrics) RecordWrite(success bool, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.WriteCount++
	if success {
		m.WriteSuccess++
		WriteTotal.WithLabelValues("success").Inc()
	} else {
		m.WriteFailure++
		WriteTotal.WithLabelValues("failure").Inc()
	}
	m.WriteLatencySum += latency
	WriteLatency.Observe(latency.Seconds())
}

// GetParseLatency 获取解析延迟统计
func (m *Metrics) GetParseLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ParseSuccess == 0 {
		return 0
	}
	return m.ParseLatencySum / time.Duration(m.ParseSuccess)
}

// GetWriteLatency 获取写入延迟统计
func (m *Metrics) GetWriteLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.WriteSuccess == 0 {
		return 0
	}
	return m.WriteLatencySum / time.Duration(m.WriteSuccess)
}

// GetThroughput 获取吞吐量（条/秒）
func (m *Metrics) GetThroughput() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	elapsed := time.Since(m.windowStart).Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(m.ParseCount) / elapsed
}

// GetFilterRate 获取过滤率
func (m *Metrics) GetFilterRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.FilterCount == 0 {
		return 0
	}
	return float64(m.FilterDropped) / float64(m.FilterCount) * 100
}

// GetSuccessRate 获取成功率
func (m *Metrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ParseCount == 0 {
		return 0
	}
	return float64(m.ParseSuccess) / float64(m.ParseCount) * 100
}

// Snapshot 获取指标快照
func (m *Metrics) Snapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"parse_count":       m.ParseCount,
		"parse_success":     m.ParseSuccess,
		"parse_failure":     m.ParseFailure,
		"parse_avg_latency": m.ParseLatencySum / time.Duration(m.ParseSuccess),
		"filter_count":      m.FilterCount,
		"filter_kept":       m.FilterKept,
		"filter_dropped":    m.FilterDropped,
		"filter_rate":       float64(m.FilterDropped) / float64(m.FilterCount) * 100,
		"transform_count":   m.TransformCount,
		"analyze_count":     m.AnalyzeCount,
		"write_count":       m.WriteCount,
		"write_success":     m.WriteSuccess,
		"write_failure":     m.WriteFailure,
		"write_avg_latency": m.WriteLatencySum / time.Duration(m.WriteSuccess),
		"throughput":        float64(m.ParseCount) / time.Since(m.windowStart).Seconds(),
		"success_rate":      float64(m.ParseSuccess) / float64(m.ParseCount) * 100,
		"window_start":      m.windowStart,
	}
}

// Reset 重置指标
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ParseCount = 0
	m.ParseSuccess = 0
	m.ParseFailure = 0
	m.ParseLatencySum = 0
	m.FilterCount = 0
	m.FilterKept = 0
	m.FilterDropped = 0
	m.TransformCount = 0
	m.TransformSuccess = 0
	m.AnalyzeCount = 0
	m.WriteCount = 0
	m.WriteSuccess = 0
	m.WriteFailure = 0
	m.WriteLatencySum = 0
	m.windowStart = time.Now()
}

// LatencyTracker 延迟追踪器
type LatencyTracker struct {
	mu         sync.Mutex
	samples    []time.Duration
	maxSamples int
}

// NewLatencyTracker 创建延迟追踪器
func NewLatencyTracker(maxSamples int) *LatencyTracker {
	return &LatencyTracker{
		samples:    make([]time.Duration, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

// Record 记录延迟样本
func (t *LatencyTracker) Record(latency time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.samples) >= t.maxSamples {
		// 移除最早的样本
		t.samples = t.samples[1:]
	}
	t.samples = append(t.samples, latency)
}

// GetStats 获取延迟统计
func (t *LatencyTracker) GetStats() *LatencyStats {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.samples) == 0 {
		return &LatencyStats{}
	}

	stats := &LatencyStats{
		Count: int64(len(t.samples)),
	}

	// 计算平均值
	var sum time.Duration
	min := t.samples[0]
	max := t.samples[0]

	for _, s := range t.samples {
		sum += s
		if s < min {
			min = s
		}
		if s > max {
			max = s
		}
	}

	stats.Avg = sum / time.Duration(stats.Count)
	stats.Min = min
	stats.Max = max

	// 排序计算百分位数
	sorted := make([]time.Duration, len(t.samples))
	copy(sorted, t.samples)
	sortDuration(sorted)

	stats.P50 = sorted[len(sorted)*50/100]
	stats.P95 = sorted[len(sorted)*95/100]
	stats.P99 = sorted[len(sorted)*99/100]

	return stats
}

func sortDuration(durations []time.Duration) {
	for i := 0; i < len(durations)-1; i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[i] > durations[j] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}
}

// Counter 简单计数器
type Counter struct {
	mu    sync.Mutex
	count int64
}

// Increment 增加计数
func (c *Counter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

// Add 增加指定数量
func (c *Counter) Add(n int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count += n
}

// Count 获取计数
func (c *Counter) Count() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// Reset 重置计数
func (c *Counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count = 0
}
