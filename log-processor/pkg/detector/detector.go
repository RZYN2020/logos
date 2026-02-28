// Package detector 提供日志格式检测功能
package detector

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
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

// DetectionResult 检测结果
type DetectionResult struct {
	Format     FormatType
	Confidence float64
	Metadata   map[string]interface{}
}

// FormatDetector 格式检测器接口
type FormatDetector interface {
	Detect(log []byte) *DetectionResult
	RegisterFormat(format FormatType, detector DetectorFunc)
}

// DetectorFunc 检测函数类型
type DetectorFunc func([]byte) *DetectionResult

// FormatDetectorImpl 格式检测器实现
type FormatDetectorImpl struct {
	detectors map[FormatType]DetectorFunc
	order     []FormatType
}

// NewFormatDetector 创建格式检测器
func NewFormatDetector() *FormatDetectorImpl {
	d := &FormatDetectorImpl{
		detectors: make(map[FormatType]DetectorFunc),
		order:     []FormatType{},
	}

	// 注册内置检测器
	d.registerBuiltInDetectors()

	return d
}

// registerBuiltInDetectors 注册内置检测器
func (d *FormatDetectorImpl) registerBuiltInDetectors() {
	d.RegisterFormat(FormatJSON, detectJSON)
	d.RegisterFormat(FormatKeyValue, detectKeyValue)
	d.RegisterFormat(FormatSyslog, detectSyslog)
	d.RegisterFormat(FormatApache, detectApache)
	d.RegisterFormat(FormatNginx, detectNginx)
}

// RegisterFormat 注册格式检测器
func (d *FormatDetectorImpl) RegisterFormat(format FormatType, detector DetectorFunc) {
	d.detectors[format] = detector
	d.order = append(d.order, format)
}

// Detect 检测日志格式
func (d *FormatDetectorImpl) Detect(log []byte) *DetectionResult {
	// 按顺序尝试所有检测器
	for _, format := range d.order {
		if detector, ok := d.detectors[format]; ok {
			result := detector(log)
			if result != nil && result.Confidence > 0.8 {
				return result
			}
		}
	}

	// 默认返回非结构化
	return &DetectionResult{
		Format:     FormatUnstructured,
		Confidence: 0.5,
		Metadata:   make(map[string]interface{}),
	}
}

// detectJSON 检测 JSON 格式
func detectJSON(log []byte) *DetectionResult {
	// 快速检查：首尾字符
	trimmed := bytes.TrimSpace(log)
	if len(trimmed) < 2 {
		return nil
	}

	if trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return nil
	}

	// 尝试解析
	var data map[string]interface{}
	if err := json.Unmarshal(log, &data); err != nil {
		return nil
	}

	// 计算置信度
	confidence := 1.0

	// 检查常见日志字段
	commonFields := []string{"timestamp", "level", "message", "service", "trace_id"}
	matchCount := 0
	for _, field := range commonFields {
		if _, ok := data[field]; ok {
			matchCount++
		}
	}

	if matchCount >= 3 {
		confidence = 1.0
	} else if matchCount >= 1 {
		confidence = 0.9
	}

	return &DetectionResult{
		Format:     FormatJSON,
		Confidence: confidence,
		Metadata: map[string]interface{}{
			"fields": len(data),
		},
	}
}

// detectKeyValue 检测 KeyValue 格式
func detectKeyValue(log []byte) *DetectionResult {
	content := string(log)

	// KeyValue 格式：key=value 或 key: value
	kvPattern := regexp.MustCompile(`[\w_-]+[=:]\s*[\S]+`)
	matches := kvPattern.FindAllString(content, -1)

	if len(matches) < 2 {
		return nil
	}

	// 计算 KV 对比例
	ratio := float64(len(matches)) / float64(len(strings.Fields(content)))

	if ratio < 0.5 {
		return nil
	}

	confidence := 0.7 + ratio*0.2

	return &DetectionResult{
		Format:     FormatKeyValue,
		Confidence: confidence,
		Metadata: map[string]interface{}{
			"kv_count": len(matches),
		},
	}
}

// detectSyslog 检测 Syslog 格式
func detectSyslog(log []byte) *DetectionResult {
	content := string(log)

	// Syslog 格式：<priority>timestamp hostname process[pid]: message
	// 或：Mon DD HH:MM:SS hostname process[pid]: message

	syslogPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^<\d{1,3}>\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}`),
		regexp.MustCompile(`^\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}\s+\w+`),
		regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+`),
	}

	for _, pattern := range syslogPatterns {
		if pattern.Match(log) {
			return &DetectionResult{
				Format:     FormatSyslog,
				Confidence: 0.9,
				Metadata: map[string]interface{}{
					"pattern": "syslog",
				},
			}
		}
	}

	// 检查是否包含 syslog 特征
	if strings.Contains(content, "host") && strings.Contains(content, ":") {
		return &DetectionResult{
			Format:     FormatSyslog,
			Confidence: 0.6,
			Metadata: map[string]interface{}{
				"pattern": "syslog-like",
			},
		}
	}

	return nil
}

// detectApache 检测 Apache 日志格式
func detectApache(log []byte) *DetectionResult {
	content := string(log)

	// Apache Common Log Format:
	// 127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326
	// Apache Combined Log Format has additional fields like referer and user-agent

	apachePattern := regexp.MustCompile(`^\S+\s+\S+\s+\S+\s+\[[^\]]+\]\s+"[A-Z]+\s+\S+\s+\S+"\s+\d{3}\s+\d+\s+"[^"]*"\s+"[^"]*"`)

	// If it matches the combined format with referer and user-agent, it's Nginx
	if apachePattern.Match(log) {
		return nil // Let Nginx detector handle it
	}

	// Basic Apache pattern without referer/user-agent
	basicApachePattern := regexp.MustCompile(`^\S+\s+\S+\s+\S+\s+\[[^\]]+\]\s+"[A-Z]+\s+\S+\s+\S+"\s+\d{3}\s+\d+$`)

	if basicApachePattern.Match(log) {
		return &DetectionResult{
			Format:     FormatApache,
			Confidence: 0.95,
			Metadata: map[string]interface{}{
				"pattern": "apache-common",
			},
		}
	}

	// Apache Combined Log Format (额外字段，但没有完整的 referer + user-agent)
	if strings.Contains(content, "\"") && strings.Contains(content, "HTTP/") {
		return &DetectionResult{
			Format:     FormatApache,
			Confidence: 0.7,
			Metadata: map[string]interface{}{
				"pattern": "apache-like",
			},
		}
	}

	return nil
}

// detectNginx 检测 Nginx 日志格式
func detectNginx(log []byte) *DetectionResult {
	content := string(log)

	// Nginx 日志格式类似 Apache，但有一些特征
	// 127.0.0.1 - - [28/Feb/2026:12:00:00 +0000] "GET /api HTTP/1.1" 200 1234 "-" "Mozilla/5.0"

	nginxPattern := regexp.MustCompile(`^\S+\s+\S+\s+\S+\s+\[[^\]]+\]\s+"[A-Z]+\s+\S+\s+\S+"\s+\d{3}\s+\d+\s+"[^"]*"\s+"[^"]*"`)

	if nginxPattern.Match(log) {
		return &DetectionResult{
			Format:     FormatNginx,
			Confidence: 0.95,
			Metadata: map[string]interface{}{
				"pattern": "nginx-combined",
			},
		}
	}

	// 检查 Nginx 特征
	if strings.Contains(content, "HTTP/") && strings.Contains(content, "\"") {
		return &DetectionResult{
			Format:     FormatNginx,
			Confidence: 0.6,
			Metadata: map[string]interface{}{
				"pattern": "nginx-like",
			},
		}
	}

	return nil
}
