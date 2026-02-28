// Package parser 提供非结构化日志解析功能
package parser

import (
	"regexp"
	"strings"
	"time"

	"github.com/log-system/log-processor/pkg/detector"
)

// UnstructuredParser 非结构化日志解析器
type UnstructuredParser struct {
	detector    *detector.UnstructuredDetector
	patterns    []ExtractPattern
	fieldExtractors []FieldExtractor
}

// ExtractPattern 提取模式
type ExtractPattern struct {
	Name     string
	Pattern  *regexp.Regexp
	FieldNames []string
}

// FieldExtractor 字段提取器
type FieldExtractor interface {
	Extract(content string) map[string]interface{}
}

// NewUnstructuredParser 创建非结构化日志解析器
func NewUnstructuredParser() *UnstructuredParser {
	p := &UnstructuredParser{
		detector: detector.NewUnstructuredDetector(nil),
		patterns: make([]ExtractPattern, 0),
		fieldExtractors: []FieldExtractor{
			&TimestampExtractor{},
			&LogLevelExtractor{},
			&IPAddressExtractor{},
			&URLEXtractor{},
			&KeyValueExtractor{},
		},
	}

	// 注册默认提取模式
	p.registerDefaultPatterns()

	return p
}

// registerDefaultPatterns 注册默认提取模式
func (p *UnstructuredParser) registerDefaultPatterns() {
	// 时间戳模式
	p.patterns = append(p.patterns, ExtractPattern{
		Name:     "timestamp_iso",
		Pattern:  regexp.MustCompile(`(\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)`),
		FieldNames: []string{"timestamp_str"},
	})

	// 日志级别模式
	p.patterns = append(p.patterns, ExtractPattern{
		Name:     "log_level",
		Pattern:  regexp.MustCompile(`\b(DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|CRITICAL|TRACE)\b`),
		FieldNames: []string{"level"},
	})

	// IP 地址模式
	p.patterns = append(p.patterns, ExtractPattern{
		Name:     "ip_address",
		Pattern:  regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\b`),
		FieldNames: []string{"ip"},
	})

	// URL 模式
	p.patterns = append(p.patterns, ExtractPattern{
		Name:     "url",
		Pattern:  regexp.MustCompile(`(https?://[^\s<>"{}|^\[\]]+)`),
		FieldNames: []string{"url"},
	})

	// HTTP 请求模式
	p.patterns = append(p.patterns, ExtractPattern{
		Name:     "http_request",
		Pattern:  regexp.MustCompile(`(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\s+(\S+)\s*(?:HTTP/[\d.]+)?`),
		FieldNames: []string{"method", "path"},
	})

	// 错误消息模式
	p.patterns = append(p.patterns, ExtractPattern{
		Name:     "error_message",
		Pattern:  regexp.MustCompile(`(?:error|exception|failed|failure)[:\s]+([^\n]+)`),
		FieldNames: []string{"error_detail"},
	})
}

// AddPattern 添加自定义提取模式
func (p *UnstructuredParser) AddPattern(name string, pattern string, fieldNames []string) error {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	p.patterns = append(p.patterns, ExtractPattern{
		Name:     name,
		Pattern:  compiled,
		FieldNames: fieldNames,
	})

	return nil
}

// AddFieldExtractor 添加字段提取器
func (p *UnstructuredParser) AddFieldExtractor(extractor FieldExtractor) {
	p.fieldExtractors = append(p.fieldExtractors, extractor)
}

// Parse 解析非结构化日志
func (p *UnstructuredParser) Parse(raw []byte) (*ParsedLog, error) {
	content := string(raw)

	parsed := &ParsedLog{
		Fields:    make(map[string]interface{}),
		Raw:       content,
		Timestamp: time.Now(),
		Level:     "INFO",
		Format:    FormatUnstructured,
	}

	// 分析内容
	analysis := p.detector.AnalyzeContent(content)

	// 提取实体
	for _, entity := range analysis.Entities {
		switch entity.Type {
		case "IP_ADDRESS":
			parsed.Fields["extracted_ip"] = entity.Value
		case "URL":
			parsed.Fields["extracted_url"] = entity.Value
		case "EMAIL":
			parsed.Fields["extracted_email"] = entity.Value
		}
	}

	// 应用提取模式
	for _, pattern := range p.patterns {
		matches := pattern.Pattern.FindStringSubmatch(content)
		if len(matches) > 1 {
			for i, fieldName := range pattern.FieldNames {
				if i+1 < len(matches) {
					value := matches[i+1]
					parsed.setField(fieldName, value)
				}
			}
		}
	}

	// 应用字段提取器
	for _, extractor := range p.fieldExtractors {
		fields := extractor.Extract(content)
		for k, v := range fields {
			parsed.Fields[k] = v
		}
	}

	// 提取关键词作为消息摘要
	if len(analysis.KeyPhrases) > 0 {
		// 取前 5 个关键词作为消息摘要
		maxLen := 5
		if len(analysis.KeyPhrases) < maxLen {
			maxLen = len(analysis.KeyPhrases)
		}
		parsed.Message = strings.Join(analysis.KeyPhrases[:maxLen], " ")
	} else {
		// 如果没有关键词，使用原始内容的前 200 个字符
		if len(content) > 200 {
			parsed.Message = content[:200] + "..."
		} else {
			parsed.Message = content
		}
	}

	return parsed, nil
}

// setField 设置字段值
func (p *ParsedLog) setField(fieldName string, value string) {
	switch fieldName {
	case "timestamp_str":
		p.Timestamp = parseTimestamp(value)
	case "level":
		p.Level = strings.ToUpper(value)
	case "method":
		p.Fields["http_method"] = value
	case "path":
		p.Fields["http_path"] = value
	case "error_detail":
		p.Fields["error_detail"] = value
	default:
		p.Fields[fieldName] = value
	}
}

// TimestampExtractor 时间戳提取器
type TimestampExtractor struct{}

// Extract 提取时间戳
func (e *TimestampExtractor) Extract(content string) map[string]interface{} {
	fields := make(map[string]interface{})

	// 多种时间格式
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)`),
		regexp.MustCompile(`(\d{2}/\d{2}/\d{4}\s+\d{2}:\d{2}:\d{2})`),
		regexp.MustCompile(`(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindString(content)
		if matches != "" {
			fields["timestamp_raw"] = matches
			break
		}
	}

	return fields
}

// LogLevelExtractor 日志级别提取器
type LogLevelExtractor struct{}

// Extract 提取日志级别
func (e *LogLevelExtractor) Extract(content string) map[string]interface{} {
	fields := make(map[string]interface{})

	levelPattern := regexp.MustCompile(`\b(DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|CRITICAL|TRACE)\b`)
	match := levelPattern.FindString(content)

	if match != "" {
		fields["detected_level"] = match
	}

	return fields
}

// IPAddressExtractor IP 地址提取器
type IPAddressExtractor struct{}

// Extract 提取 IP 地址
func (e *IPAddressExtractor) Extract(content string) map[string]interface{} {
	fields := make(map[string]interface{})

	ipPattern := regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\b`)
	matches := ipPattern.FindAllString(content, -1)

	if len(matches) > 0 {
		fields["detected_ips"] = matches
		if len(matches) == 1 {
			fields["source_ip"] = matches[0]
		}
	}

	return fields
}

// URLEXtractor URL 提取器
type URLEXtractor struct{}

// Extract 提取 URL
func (e *URLEXtractor) Extract(content string) map[string]interface{} {
	fields := make(map[string]interface{})

	urlPattern := regexp.MustCompile(`(https?://[^\s<>"{}|^\[\]]+)`)
	matches := urlPattern.FindAllString(content, -1)

	if len(matches) > 0 {
		fields["detected_urls"] = matches
	}

	return fields
}

// KeyValueExtractor KeyValue 提取器
type KeyValueExtractor struct{}

// Extract 提取键值对
func (e *KeyValueExtractor) Extract(content string) map[string]interface{} {
	fields := make(map[string]interface{})

	// 匹配 key=value 或 key: value 格式
	kvPattern := regexp.MustCompile(`(\w+)[=:]\s*([^\s,;]+|"[^"]*")`)
	matches := kvPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		key := match[1]
		value := strings.Trim(match[2], `"`)

		// 跳过已知的标准字段
		if key == "timestamp" || key == "time" || key == "level" || key == "msg" || key == "message" {
			continue
		}

		fields["kv_"+key] = value
	}

	return fields
}
