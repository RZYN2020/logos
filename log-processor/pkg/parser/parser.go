// Package parser 提供日志解析功能
package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Parser 日志解析器接口
type Parser interface {
	Parse(raw []byte) (*ParsedLog, error)
}

// ParsedLog 解析后的日志
type ParsedLog struct {
	Timestamp time.Time
	Level     string
	Message   string
	Service   string
	TraceID   string
	SpanID    string
	Fields    map[string]interface{}
	Raw       string
}

// JSONParser JSON 格式解析器
type JSONParser struct{}

// NewJSONParser 创建 JSON 解析器
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse 解析 JSON 格式日志
func (p *JSONParser) Parse(raw []byte) (*ParsedLog, error) {
	var log map[string]interface{}
	if err := json.Unmarshal(raw, &log); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	parsed := &ParsedLog{
		Fields: make(map[string]interface{}),
		Raw:    string(raw),
	}

	// 提取标准字段
	if ts, ok := log["timestamp"]; ok {
		parsed.Timestamp = parseTimestamp(ts)
	} else {
		parsed.Timestamp = time.Now()
	}

	if level, ok := log["level"]; ok {
		parsed.Level = parseString(level)
	} else {
		parsed.Level = "INFO"
	}

	if msg, ok := log["message"]; ok {
		parsed.Message = parseString(msg)
	}

	if service, ok := log["service"]; ok {
		parsed.Service = parseString(service)
	}

	if traceID, ok := log["trace_id"]; ok {
		parsed.TraceID = parseString(traceID)
	}

	if spanID, ok := log["span_id"]; ok {
		parsed.SpanID = parseString(spanID)
	}

	// 其余字段放入 Fields
	for k, v := range log {
		switch k {
		case "timestamp", "level", "message", "service", "trace_id", "span_id":
			// 已处理的标准字段
		default:
			parsed.Fields[k] = v
		}
	}

	return parsed, nil
}

// RegexParser 正则表达式解析器（用于非结构化日志）
type RegexParser struct {
	pattern *regexp.Regexp
	fields  []string
}

// NewRegexParser 创建正则解析器
func NewRegexParser(pattern string, fields []string) (*RegexParser, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return &RegexParser{
		pattern: re,
		fields:  fields,
	}, nil
}

// Parse 使用正则解析日志
func (p *RegexParser) Parse(raw []byte) (*ParsedLog, error) {
	matches := p.pattern.FindStringSubmatch(string(raw))
	if matches == nil {
		return nil, fmt.Errorf("log does not match pattern")
	}

	parsed := &ParsedLog{
		Fields:    make(map[string]interface{}),
		Raw:       string(raw),
		Timestamp: time.Now(),
		Level:     "INFO",
	}

	// 将捕获组映射到字段
	for i, field := range p.fields {
		if i+1 < len(matches) {
			value := matches[i+1]

			switch field {
			case "timestamp":
				parsed.Timestamp = parseTimestamp(value)
			case "level":
				parsed.Level = strings.ToUpper(value)
			case "message":
				parsed.Message = value
			case "service":
				parsed.Service = value
			default:
				parsed.Fields[field] = value
			}
		}
	}

	return parsed, nil
}

// MultiParser 多格式解析器（自动检测格式）
type MultiParser struct {
	parsers []Parser
}

// NewMultiParser 创建多格式解析器
func NewMultiParser() *MultiParser {
	return &MultiParser{
		parsers: []Parser{
			NewJSONParser(),
		},
	}
}

// AddParser 添加解析器
func (p *MultiParser) AddParser(parser Parser) {
	p.parsers = append(p.parsers, parser)
}

// Parse 尝试所有解析器直到成功
func (p *MultiParser) Parse(raw []byte) (*ParsedLog, error) {
	var lastErr error

	for _, parser := range p.parsers {
		log, err := parser.Parse(raw)
		if err == nil {
			return log, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all parsers failed, last error: %w", lastErr)
}

// 辅助函数

func parseTimestamp(v interface{}) time.Time {
	switch val := v.(type) {
	case string:
		// 尝试多种时间格式
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02 15:04:05.000",
		}

		for _, format := range formats {
			if t, err := time.Parse(format, val); err == nil {
				return t
			}
		}
	case float64:
		// Unix 时间戳（秒）
		return time.Unix(int64(val), 0)
	}

	return time.Now()
}

func parseString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
