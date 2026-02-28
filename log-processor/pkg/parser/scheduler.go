// Package parser 提供解析器选择和调度功能
package parser

import (
	"fmt"
	"sync"
	"time"

	"github.com/log-system/log-processor/pkg/detector"
)

// ParserScheduler 解析器调度器
type ParserScheduler struct {
	mu            sync.RWMutex
	parsers       map[FormatType]Parser
	detector      *detector.FormatDetectorImpl
	strategy      SchedulingStrategy
	stats         map[FormatType]*ParserStats
	statsMu       sync.RWMutex
	cache         *ParserCache
}

// SchedulingStrategy 调度策略接口
type SchedulingStrategy interface {
	SelectParser(format FormatType, parsers map[FormatType]Parser) (Parser, error)
}

// ParserStats 解析器统计
type ParserStats struct {
	Format         FormatType
	TotalParsed    int64
	SuccessCount   int64
	FailureCount   int64
	AvgParseTime   time.Duration
	LastUsed       time.Time
}

// ParserCache 解析器缓存
type ParserCache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	ttl      time.Duration
	maxSize  int
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Result    *ParsedLog
	ExpiresAt time.Time
}

// NewParserScheduler 创建解析器调度器
func NewParserScheduler() *ParserScheduler {
	s := &ParserScheduler{
		parsers:   make(map[FormatType]Parser),
		detector:  detector.NewFormatDetector(),
		strategy:  &DefaultSchedulingStrategy{},
		stats:     make(map[FormatType]*ParserStats),
		cache:     NewParserCache(1000, 5*time.Minute),
	}

	// 注册默认解析器
	s.registerDefaultParsers()

	return s
}

// registerDefaultParsers 注册默认解析器
func (s *ParserScheduler) registerDefaultParsers() {
	s.RegisterParser(FormatJSON, NewJSONParser())
	s.RegisterParser(FormatKeyValue, NewKeyValueParser())
	s.RegisterParser(FormatSyslog, NewSyslogParser())
	s.RegisterParser(FormatApache, NewApacheParser())
	s.RegisterParser(FormatNginx, NewNginxParser())
	s.RegisterParser(FormatUnstructured, NewUnstructuredParser())

	// 初始化统计
	for format := range s.parsers {
		s.stats[format] = &ParserStats{
			Format: format,
		}
	}
}

// RegisterParser 注册解析器
func (s *ParserScheduler) RegisterParser(format FormatType, parser Parser) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.parsers[format] = parser
	if s.stats[format] == nil {
		s.stats[format] = &ParserStats{
			Format: format,
		}
	}
}

// SetStrategy 设置调度策略
func (s *ParserScheduler) SetStrategy(strategy SchedulingStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.strategy = strategy
}

// Parse 调度解析
func (s *ParserScheduler) Parse(raw []byte) (*ParsedLog, error) {
	// 检查缓存
	cacheKey := string(raw)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached, nil
	}

	// 检测格式
	detectionResult := s.detector.Detect(raw)
	format := FormatType(detectionResult.Format)

	// 选择解析器
	parser, err := s.strategy.SelectParser(format, s.parsers)
	if err != nil {
		return nil, fmt.Errorf("failed to select parser: %w", err)
	}

	// 解析
	startTime := time.Now()
	result, err := parser.Parse(raw)
	parseTime := time.Since(startTime)

	// 更新统计
	s.updateStats(format, err == nil, parseTime)

	if err != nil {
		// 尝试降级解析
		result, err = s.fallbackParse(raw)
		if err != nil {
			return nil, err
		}
	}

	result.Format = format

	// 缓存结果
	s.cache.Set(cacheKey, result)

	return result, nil
}

// fallbackParse 降级解析（当首选解析器失败时）
func (s *ParserScheduler) fallbackParse(raw []byte) (*ParsedLog, error) {
	// 尝试所有解析器
	for format, parser := range s.parsers {
		result, err := parser.Parse(raw)
		if err == nil {
			s.updateStats(format, true, 0)
			result.Format = format
			return result, nil
		}
	}

	return nil, fmt.Errorf("all parsers failed")
}

// updateStats 更新统计
func (s *ParserScheduler) updateStats(format FormatType, success bool, parseTime time.Duration) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	stats := s.stats[format]
	if stats == nil {
		stats = &ParserStats{
			Format: format,
		}
		s.stats[format] = stats
	}

	stats.TotalParsed++
	if success {
		stats.SuccessCount++
	} else {
		stats.FailureCount++
	}

	// 更新平均解析时间
	if stats.SuccessCount > 0 {
		totalNanos := int64(stats.AvgParseTime)*(stats.SuccessCount-1) + int64(parseTime)
		stats.AvgParseTime = time.Duration(totalNanos / int64(stats.SuccessCount))
	}
	stats.LastUsed = time.Now()
}

// GetStats 获取解析器统计
func (s *ParserScheduler) GetStats(format FormatType) (*ParserStats, error) {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()

	stats, ok := s.stats[format]
	if !ok {
		return nil, fmt.Errorf("no stats for format: %s", format)
	}

	return stats, nil
}

// GetAllStats 获取所有统计
func (s *ParserScheduler) GetAllStats() map[FormatType]*ParserStats {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()

	result := make(map[FormatType]*ParserStats)
	for k, v := range s.stats {
		result[k] = v
	}
	return result
}

// SetDetector 设置自定义检测器
func (s *ParserScheduler) SetDetector(d *detector.FormatDetectorImpl) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.detector = d
}

// DefaultSchedulingStrategy 默认调度策略
type DefaultSchedulingStrategy struct{}

// SelectParser 选择解析器
func (s *DefaultSchedulingStrategy) SelectParser(format FormatType, parsers map[FormatType]Parser) (Parser, error) {
	parser, ok := parsers[format]
	if !ok {
		return nil, fmt.Errorf("no parser for format: %s", format)
	}
	return parser, nil
}

// NewParserCache 创建解析器缓存
func NewParserCache(maxSize int, ttl time.Duration) *ParserCache {
	return &ParserCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get 获取缓存
func (c *ParserCache) Get(key string) (*ParsedLog, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry.Result, true
}

// Set 设置缓存
func (c *ParserCache) Set(key string, result *ParsedLog) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否需要清理
	if len(c.entries) >= c.maxSize {
		c.cleanup()
	}

	c.entries[key] = &CacheEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// cleanup 清理过期缓存
func (c *ParserCache) cleanup() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

// AdaptiveSchedulingStrategy 自适应调度策略（基于历史性能）
type AdaptiveSchedulingStrategy struct {
	performanceHistory map[FormatType][]time.Duration
}

// NewAdaptiveSchedulingStrategy 创建自适应调度策略
func NewAdaptiveSchedulingStrategy() *AdaptiveSchedulingStrategy {
	return &AdaptiveSchedulingStrategy{
		performanceHistory: make(map[FormatType][]time.Duration),
	}
}

// SelectParser 基于性能选择最佳解析器
func (s *AdaptiveSchedulingStrategy) SelectParser(format FormatType, parsers map[FormatType]Parser) (Parser, error) {
	parser, ok := parsers[format]
	if !ok {
		return nil, fmt.Errorf("no parser for format: %s", format)
	}

	// 可以基于历史性能选择解析器
	// 目前简单返回对应的解析器
	return parser, nil
}

// RecordPerformance 记录性能
func (s *AdaptiveSchedulingStrategy) RecordPerformance(format FormatType, parseTime time.Duration) {
	history := s.performanceHistory[format]
	history = append(history, parseTime)

	// 保留最近 100 次记录
	if len(history) > 100 {
		history = history[len(history)-100:]
	}

	s.performanceHistory[format] = history
}
