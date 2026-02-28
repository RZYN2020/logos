// Package analyzer 提供实体提取功能
package analyzer

import (
	"regexp"
	"strings"
	"time"
)

// IPAddressExtractor IP 地址提取器
type IPAddressExtractor struct{}

// Extract 提取 IP 地址
func (e *IPAddressExtractor) Extract(text string) []Entity {
	entities := make([]Entity, 0)

	// IPv4 模式
	ipPattern := regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\b`)
	matches := ipPattern.FindAllStringIndex(text, -1)

	for _, match := range matches {
		entities = append(entities, Entity{
			Type:     "IP_ADDRESS",
			Value:    text[match[0]:match[1]],
			Confidence: 0.95,
			Start:    match[0],
			End:      match[1],
		})
	}

	return entities
}

// URLEXtractor URL 提取器
type URLEXtractor struct{}

// Extract 提取 URL
func (e *URLEXtractor) Extract(text string) []Entity {
	entities := make([]Entity, 0)

	// URL 模式
	urlPattern := regexp.MustCompile(`(https?://[^\s<>"{}|^\[\]]+)`)
	matches := urlPattern.FindAllStringIndex(text, -1)

	for _, match := range matches {
		entities = append(entities, Entity{
			Type:     "URL",
			Value:    text[match[0]:match[1]],
			Confidence: 0.95,
			Start:    match[0],
			End:      match[1],
		})
	}

	return entities
}

// EmailExtractor 邮箱提取器
type EmailExtractor struct{}

// Extract 提取邮箱地址
func (e *EmailExtractor) Extract(text string) []Entity {
	entities := make([]Entity, 0)

	// 邮箱模式
	emailPattern := regexp.MustCompile(`[\w\-.]+@[\w\-.]+\.\w+`)
	matches := emailPattern.FindAllStringIndex(text, -1)

	for _, match := range matches {
		entities = append(entities, Entity{
			Type:     "EMAIL",
			Value:    text[match[0]:match[1]],
			Confidence: 0.9,
			Start:    match[0],
			End:      match[1],
		})
	}

	return entities
}

// TimestampExtractor 时间戳提取器
type TimestampExtractor struct{}

// Extract 提取时间戳
func (e *TimestampExtractor) Extract(text string) []Entity {
	entities := make([]Entity, 0)

	// 多种时间格式模式
	patterns := []struct {
		regex   *regexp.Regexp
		format  string
	}{
		{regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`), "iso8601"},
		{regexp.MustCompile(`\d{2}/\d{2}/\d{4}\s+\d{2}:\d{2}:\d{2}`), "us_date"},
		{regexp.MustCompile(`\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}`), "syslog"},
		{regexp.MustCompile(`\d{10,13}`), "unix"},
	}

	for _, p := range patterns {
		matches := p.regex.FindAllStringIndex(text, -1)
		for _, match := range matches {
			timestampStr := text[match[0]:match[1]]
			if isValidTimestamp(timestampStr, p.format) {
				entities = append(entities, Entity{
					Type:     "TIMESTAMP",
					Value:    timestampStr,
					Confidence: 0.85,
					Start:    match[0],
					End:      match[1],
				})
			}
		}
	}

	return entities
}

// isValidTimestamp 验证是否是有效的时间戳
func isValidTimestamp(s, format string) bool {
	switch format {
	case "iso8601":
		_, err := time.Parse(time.RFC3339, s)
		if err != nil {
			_, err = time.Parse("2006-01-02 15:04:05", s)
		}
		return err == nil
	case "us_date":
		_, err := time.Parse("01/02/2006 15:04:05", s)
		return err == nil
	case "syslog":
		_, err := time.Parse("Jan 2 15:04:05", s)
		return err == nil
	case "unix":
		return len(s) == 10 || len(s) == 13
	}
	return false
}

// ErrorPatternExtractor 错误模式提取器
type ErrorPatternExtractor struct{}

// Extract 提取错误相关实体
func (e *ErrorPatternExtractor) Extract(text string) []Entity {
	entities := make([]Entity, 0)

	// 错误模式
	errorPatterns := []struct {
		pattern *regexp.Regexp
		errType string
	}{
		{regexp.MustCompile(`(?i)(error|exception|failed|failure|fatal|critical)[:\s]+([^\n]+)`), "error_message"},
		{regexp.MustCompile(`(?i)(panic|segfault|segmentation fault)[:\s]*([^\n]*)`), "panic"},
		{regexp.MustCompile(`(?i)(timeout|timed out)[:\s]*([^\n]*)`), "timeout"},
		{regexp.MustCompile(`(?i)(connection refused|connection reset|connection lost)`), "connection_error"},
	}

	for _, p := range errorPatterns {
		matches := p.pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				valueStart, valueEnd := match[2], match[3]
				entities = append(entities, Entity{
					Type:     "ERROR_" + strings.ToUpper(p.errType),
					Value:    strings.TrimSpace(text[valueStart:valueEnd]),
					Confidence: 0.9,
					Start:    match[0],
					End:      match[1],
				})
			}
		}
	}

	return entities
}

// StatusCodeExtractor 状态码提取器
type StatusCodeExtractor struct{}

// Extract 提取 HTTP 状态码
func (e *StatusCodeExtractor) Extract(text string) []Entity {
	entities := make([]Entity, 0)

	// HTTP 状态码模式
	statusPattern := regexp.MustCompile(`\b([1-5][0-9]{2})\b`)
	matches := statusPattern.FindAllStringIndex(text, -1)

	for _, match := range matches {
		code := text[match[0]:match[1]]
		// 检查是否是 HTTP 状态码上下文
		if isHTTPContext(text, match[0]) {
			entities = append(entities, Entity{
				Type:     "HTTP_STATUS",
				Value:    code,
				Confidence: 0.85,
				Start:    match[0],
				End:      match[1],
			})
		}
	}

	return entities
}

// isHTTPContext 检查是否是 HTTP 上下文
func isHTTPContext(text string, pos int) bool {
	// 检查附近是否有 HTTP 相关关键词
	start := pos - 50
	if start < 0 {
		start = 0
	}
	end := pos + 50
	if end > len(text) {
		end = len(text)
	}

	context := strings.ToLower(text[start:end])
	httpKeywords := []string{"http", "get", "post", "put", "delete", "request", "response", "url", "path"}

	for _, keyword := range httpKeywords {
		if strings.Contains(context, keyword) {
			return true
		}
	}

	return false
}

// UserIDExtractor 用户 ID 提取器
type UserIDExtractor struct{}

// Extract 提取用户 ID
func (e *UserIDExtractor) Extract(text string) []Entity {
	entities := make([]Entity, 0)

	// 用户 ID 模式
	userPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:user[_-]?id|uid|user)[=:]\s*([a-zA-Z0-9_-]+)`),
		regexp.MustCompile(`(?:user[_-]?name|username|login)[=:]\s*([a-zA-Z0-9_-]+)`),
	}

	for _, pattern := range userPatterns {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				valueStart, valueEnd := match[2], match[3]
				entities = append(entities, Entity{
					Type:     "USER_ID",
					Value:    text[valueStart:valueEnd],
					Confidence: 0.8,
					Start:    match[0],
					End:      match[1],
				})
			}
		}
	}

	return entities
}
