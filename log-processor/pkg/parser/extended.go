// Package parser 提供扩展的解析器功能
package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/log-system/log-processor/pkg/detector"
)

// FormatType 日志格式类型
type FormatType string

const (
	FormatJSON        FormatType = "json"
	FormatKeyValue    FormatType = "key_value"
	FormatSyslog      FormatType = "syslog"
	FormatApache      FormatType = "apache"
	FormatNginx       FormatType = "nginx"
	FormatUnstructured FormatType = "unstructured"
)

// ExtendedParser 扩展的解析器接口
type ExtendedParser interface {
	Parse(raw []byte) (*ParsedLog, error)
	ParseWithFormat(raw []byte, format FormatType) (*ParsedLog, error)
	SupportsFormat(format FormatType) bool
	GetName() string
}

// ExtendedMultiParser 扩展的多格式解析器
type ExtendedMultiParser struct {
	parsers     map[FormatType]Parser
	formatOrder []FormatType
	detector    *detector.FormatDetectorImpl
}

// NewExtendedMultiParser 创建扩展的多格式解析器
func NewExtendedMultiParser() *ExtendedMultiParser {
	p := &ExtendedMultiParser{
		parsers:     make(map[FormatType]Parser),
		formatOrder: []FormatType{},
		detector:    detector.NewFormatDetector(),
	}

	// 注册默认解析器
	p.registerDefaultParsers()

	return p
}

// registerDefaultParsers 注册默认解析器
func (p *ExtendedMultiParser) registerDefaultParsers() {
	p.RegisterParser(FormatJSON, NewJSONParser())
	p.RegisterParser(FormatKeyValue, NewKeyValueParser())
	p.RegisterParser(FormatSyslog, NewSyslogParser())
	p.RegisterParser(FormatApache, NewApacheParser())
	p.RegisterParser(FormatNginx, NewNginxParser())
	p.RegisterParser(FormatUnstructured, NewUnstructuredParser())
}

// RegisterParser 注册指定格式的解析器
func (p *ExtendedMultiParser) RegisterParser(format FormatType, parser Parser) {
	if _, ok := p.parsers[format]; !ok {
		p.formatOrder = append(p.formatOrder, format)
	}
	p.parsers[format] = parser
}

// Parse 解析日志（自动检测格式）
func (p *ExtendedMultiParser) Parse(raw []byte) (*ParsedLog, error) {
	// 先检测格式
	result := p.detector.Detect(raw)

	// 将 detector.FormatType 转换为 FormatType
	format := FormatType(result.Format)

	// 尝试对应格式的解析器
	if parser, ok := p.parsers[format]; ok {
		parsed, err := parser.Parse(raw)
		if err == nil {
			parsed.Format = format
			return parsed, nil
		}
	}

	// 如果特定格式解析失败，尝试所有解析器
	for _, f := range p.formatOrder {
		if f == format {
			continue // 已经试过了
		}
		if parser, ok := p.parsers[f]; ok {
			parsed, err := parser.Parse(raw)
			if err == nil {
				parsed.Format = f
				return parsed, nil
			}
		}
	}

	return nil, fmt.Errorf("all parsers failed")
}

// ParseWithFormat 使用指定格式解析
func (p *ExtendedMultiParser) ParseWithFormat(raw []byte, format FormatType) (*ParsedLog, error) {
	if parser, ok := p.parsers[format]; ok {
		parsed, err := parser.Parse(raw)
		if err == nil {
			parsed.Format = format
			return parsed, nil
		}
		return nil, err
	}
	return nil, fmt.Errorf("unsupported format: %s", format)
}

// SupportsFormat 检查是否支持指定格式
func (p *ExtendedMultiParser) SupportsFormat(format FormatType) bool {
	_, ok := p.parsers[format]
	return ok
}

// GetName 获取解析器名称
func (p *ExtendedMultiParser) GetName() string {
	return "ExtendedMultiParser"
}

// SetDetector 设置自定义格式检测器
func (p *ExtendedMultiParser) SetDetector(d *detector.FormatDetectorImpl) {
	p.detector = d
}

// KeyValueParser KeyValue 格式解析器
type KeyValueParser struct {
	delimiter string
}

// NewKeyValueParser 创建 KeyValue 解析器
func NewKeyValueParser() *KeyValueParser {
	return &KeyValueParser{
		delimiter: "=",
	}
}

// Parse 解析 KeyValue 格式日志
func (p *KeyValueParser) Parse(raw []byte) (*ParsedLog, error) {
	content := string(raw)
	parsed := &ParsedLog{
		Fields:    make(map[string]interface{}),
		Raw:       content,
		Timestamp: time.Now(),
		Level:     "INFO",
	}

	// KeyValue 正则
	kvPattern := regexp.MustCompile(`(\w+)[=:]\s*([^\s]+|"[^"]*")`)
	matches := kvPattern.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("no key-value pairs found")
	}

	for _, match := range matches {
		key := match[1]
		value := strings.Trim(match[2], `"`)

		switch key {
		case "timestamp", "time", "ts":
			parsed.Timestamp = parseTimestamp(value)
		case "level", "lvl":
			parsed.Level = strings.ToUpper(value)
		case "message", "msg":
			parsed.Message = value
		case "service", "svc":
			parsed.Service = value
		case "trace_id", "traceid":
			parsed.TraceID = value
		case "span_id", "spanid":
			parsed.SpanID = value
		default:
			parsed.Fields[key] = value
		}
	}

	return parsed, nil
}

// SyslogParser Syslog 格式解析器
type SyslogParser struct {
	pattern *regexp.Regexp
}

// NewSyslogParser 创建 Syslog 解析器
func NewSyslogParser() *SyslogParser {
	// RFC 3164: <priority>timestamp hostname process[pid]: message
	pattern := regexp.MustCompile(`^<(\d{1,3})>(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+(\S+?)(?:\[(\d+)\])?:\s*(.*)$`)
	return &SyslogParser{
		pattern: pattern,
	}
}

// Parse 解析 Syslog 格式日志
func (p *SyslogParser) Parse(raw []byte) (*ParsedLog, error) {
	matches := p.pattern.FindStringSubmatch(string(raw))
	if matches == nil {
		return nil, fmt.Errorf("does not match syslog pattern")
	}

	parsed := &ParsedLog{
		Fields:    make(map[string]interface{}),
		Raw:       string(raw),
		Timestamp: time.Now(),
		Level:     "INFO",
	}

	// 解析优先级
	priority := parsePriority(matches[1])
	parsed.Level = priorityToLevel(priority)

	// 解析时间戳
	parsed.Timestamp = parseSyslogTimestamp(matches[2])

	// 主机名
	parsed.Fields["hostname"] = matches[3]

	// 进程名
	parsed.Service = matches[4]

	// PID
	if matches[5] != "" {
		parsed.Fields["pid"] = matches[5]
	}

	// 消息
	parsed.Message = matches[6]

	return parsed, nil
}

// parsePriority 解析 syslog 优先级
func parsePriority(s string) int {
	var priority int
	fmt.Sscanf(s, "%d", &priority) // nolint:errcheck
	return priority
}

// priorityToLevel 优先级转日志级别
func priorityToLevel(priority int) string {
	severity := priority % 8
	switch severity {
	case 0, 1, 2, 3:
		return "ERROR"
	case 4:
		return "WARN"
	case 5, 6:
		return "INFO"
	case 7:
		return "DEBUG"
	default:
		return "INFO"
	}
}

// parseSyslogTimestamp 解析 syslog 时间戳
func parseSyslogTimestamp(s string) time.Time {
	// 添加当前年份
	currentYear := time.Now().Year()
	t, err := time.Parse("Jan 2 15:04:05", s)
	if err != nil {
		return time.Now()
	}
	return time.Date(currentYear, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
}

// ApacheParser Apache 日志格式解析器
type ApacheParser struct {
	pattern *regexp.Regexp
}

// NewApacheParser 创建 Apache 解析器
func NewApacheParser() *ApacheParser {
	// Common Log Format: 127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326
	pattern := regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)\s+\[([^\]]+)\]\s+"([A-Z]+)\s+(\S+)\s+(\S+)"\s+(\d{3})\s+(\d+|-)`)
	return &ApacheParser{
		pattern: pattern,
	}
}

// Parse 解析 Apache 格式日志
func (p *ApacheParser) Parse(raw []byte) (*ParsedLog, error) {
	matches := p.pattern.FindStringSubmatch(string(raw))
	if matches == nil {
		return nil, fmt.Errorf("does not match apache log pattern")
	}

	parsed := &ParsedLog{
		Fields:    make(map[string]interface{}),
		Raw:       string(raw),
		Timestamp: time.Now(),
		Level:     "INFO",
	}

	// 客户端 IP
	parsed.Fields["client_ip"] = matches[1]

	// 身份
	if matches[2] != "-" {
		parsed.Fields["ident"] = matches[2]
	}

	// 用户
	if matches[3] != "-" {
		parsed.Fields["user"] = matches[3]
	}

	// 时间戳
	parsed.Timestamp = parseApacheTimestamp(matches[4])

	// HTTP 方法
	parsed.Fields["http_method"] = matches[5]

	// URL 路径
	parsed.Fields["http_path"] = matches[6]

	// HTTP 协议
	parsed.Fields["http_protocol"] = matches[7]

	// 状态码
	status := parseStatus(matches[8])
	parsed.Fields["http_status"] = status
	if status >= 400 {
		parsed.Level = "WARN"
	}
	if status >= 500 {
		parsed.Level = "ERROR"
	}

	// 响应大小
	if matches[9] != "-" {
		parsed.Fields["response_size"] = matches[9]
	}

	// 构建消息
	parsed.Message = fmt.Sprintf("%s %s %d", matches[5], matches[6], status)

	return parsed, nil
}

// parseApacheTimestamp 解析 Apache 时间戳
func parseApacheTimestamp(s string) time.Time {
	t, err := time.Parse("02/Jan/2006:15:04:05 -0700", s)
	if err != nil {
		return time.Now()
	}
	return t
}

// parseStatus 解析状态码
func parseStatus(s string) int {
	var status int
	fmt.Sscanf(s, "%d", &status) // nolint:errcheck
	return status
}

// NginxParser Nginx 日志格式解析器
type NginxParser struct {
	pattern *regexp.Regexp
}

// NewNginxParser 创建 Nginx 解析器
func NewNginxParser() *NginxParser {
	// Combined Log Format: 127.0.0.1 - - [28/Feb/2026:12:00:00 +0000] "GET /api HTTP/1.1" 200 1234 "-" "Mozilla/5.0"
	pattern := regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)\s+\[([^\]]+)\]\s+"([A-Z]+)\s+(\S+)\s+(\S+)"\s+(\d{3})\s+(\d+|-)\s+"([^"]*)"\s+"([^"]*)"`)
	return &NginxParser{
		pattern: pattern,
	}
}

// Parse 解析 Nginx 格式日志
func (p *NginxParser) Parse(raw []byte) (*ParsedLog, error) {
	matches := p.pattern.FindStringSubmatch(string(raw))
	if matches == nil {
		return nil, fmt.Errorf("does not match nginx log pattern")
	}

	parsed := &ParsedLog{
		Fields:    make(map[string]interface{}),
		Raw:       string(raw),
		Timestamp: time.Now(),
		Level:     "INFO",
	}

	// 客户端 IP
	parsed.Fields["client_ip"] = matches[1]

	// Referer
	if matches[10] != "-" {
		parsed.Fields["referer"] = matches[10]
	}

	// User Agent
	if matches[11] != "-" {
		parsed.Fields["user_agent"] = matches[11]
	}

	// 时间戳
	parsed.Timestamp = parseApacheTimestamp(matches[4])

	// HTTP 方法
	parsed.Fields["http_method"] = matches[5]

	// URL 路径
	parsed.Fields["http_path"] = matches[6]

	// HTTP 协议
	parsed.Fields["http_protocol"] = matches[7]

	// 状态码
	status := parseStatus(matches[8])
	parsed.Fields["http_status"] = status
	if status >= 400 {
		parsed.Level = "WARN"
	}
	if status >= 500 {
		parsed.Level = "ERROR"
	}

	// 响应大小
	if matches[9] != "-" {
		parsed.Fields["response_size"] = matches[9]
	}

	// 构建消息
	parsed.Message = fmt.Sprintf("%s %s %d", matches[5], matches[6], status)

	return parsed, nil
}
