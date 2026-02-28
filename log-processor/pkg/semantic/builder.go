// Package semantic 提供日志语义化处理
// 从 Kafka 消费原始日志，提取和增强语义信息
package semantic

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Service     string                 `json:"service"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	Raw         string                 `json:"raw"`
	Environment string                 `json:"environment,omitempty"`
	Host        string                 `json:"host,omitempty"`
}

// EnrichedLog 增强后的日志
type EnrichedLog struct {
	LogEntry
	// 提取的语义字段
	HTTPMethod   string `json:"http_method,omitempty"`
	HTTPPath     string `json:"http_path,omitempty"`
	HTTPStatus   int    `json:"http_status,omitempty"`
	ResponseTime int64  `json:"response_time_ms,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	Action       string `json:"action,omitempty"`
	Category     string `json:"category,omitempty"`

	// 业务上下文
	BusinessDomain string `json:"business_domain,omitempty"`
	TenantID       string `json:"tenant_id,omitempty"`

	// 异常标记
	IsError   bool   `json:"is_error"`
	ErrorType string `json:"error_type,omitempty"`
}

// Builder 语义构建器
type Builder struct {
	extractors      []FieldExtractor
	enrichers       []ContextEnricher
	autoInferEnabled bool
}

// FieldExtractor 字段提取器接口
type FieldExtractor interface {
	Extract(entry *LogEntry) map[string]interface{}
}

// ContextEnricher 上下文增强器接口
type ContextEnricher interface {
	Enrich(ctx context.Context, entry *LogEntry) map[string]interface{}
}

// BuilderOption 构建器选项
type BuilderOption func(*Builder)

// WithAutoInfer 启用自动推断
func WithAutoInfer(enabled bool) BuilderOption {
	return func(b *Builder) {
		b.autoInferEnabled = enabled
	}
}

// NewBuilder 创建语义构建器
func NewBuilder(opts ...BuilderOption) *Builder {
	b := &Builder{
		extractors: []FieldExtractor{
			&HTTPExtractor{},
			&UserExtractor{},
			&ErrorExtractor{},
			&TextAnalysisExtractor{},
		},
		enrichers: []ContextEnricher{
			&DomainEnricher{},
			&TenantEnricher{},
			&BusinessAttributeEnricher{},
		},
		autoInferEnabled: true,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Build 构建语义化日志
func (b *Builder) Build(ctx context.Context, entry *LogEntry) *EnrichedLog {
	enriched := &EnrichedLog{
		LogEntry: *entry,
	}

	// 初始化 Fields（如果为空）
	if enriched.Fields == nil {
		enriched.Fields = make(map[string]interface{})
	}

	// 应用字段提取器
	for _, extractor := range b.extractors {
		fields := extractor.Extract(entry)
		for k, v := range fields {
			enriched.Fields[k] = v
		}
	}

	// 应用上下文增强器
	for _, enricher := range b.enrichers {
		fields := enricher.Enrich(ctx, entry)
		for k, v := range fields {
			enriched.Fields[k] = v
		}
	}

	// 从提取的字段中设置结构化字段
	b.populateFields(enriched)

	return enriched
}

// populateFields 从 Fields 填充结构化字段
func (b *Builder) populateFields(enriched *EnrichedLog) {
	if v, ok := enriched.Fields["http_method"]; ok {
		enriched.HTTPMethod, _ = v.(string)
	}
	if v, ok := enriched.Fields["http_path"]; ok {
		enriched.HTTPPath, _ = v.(string)
	}
	if v, ok := enriched.Fields["http_status"]; ok {
		enriched.HTTPStatus, _ = v.(int)
	}
	if v, ok := enriched.Fields["user_id"]; ok {
		enriched.UserID, _ = v.(string)
	}
	if v, ok := enriched.Fields["error_type"]; ok {
		enriched.ErrorType, _ = v.(string)
		enriched.IsError = true
	}
}

// HTTPExtractor HTTP 信息提取器
type HTTPExtractor struct{}

// Extract 提取 HTTP 相关信息
func (e *HTTPExtractor) Extract(entry *LogEntry) map[string]interface{} {
	fields := make(map[string]interface{})

	// 从 message 中提取 HTTP 信息
	// 示例: "GET /api/users 200 45ms"
	var method, path string
	var status int
	if _, err := fmt.Sscanf(entry.Message, "%s %s %d", &method, &path, &status); err == nil {
		fields["http_method"] = method
		fields["http_path"] = path
		fields["http_status"] = status
		fields["category"] = "http_request"
	}

	return fields
}

// UserExtractor 用户信息提取器
type UserExtractor struct{}

// Extract 提取用户信息
func (e *UserExtractor) Extract(entry *LogEntry) map[string]interface{} {
	fields := make(map[string]interface{})

	// 从 Fields 中提取用户 ID
	if userID, ok := entry.Fields["user_id"]; ok {
		fields["user_id"] = userID
	}
	if userID, ok := entry.Fields["uid"]; ok {
		fields["user_id"] = userID
	}

	return fields
}

// ErrorExtractor 错误信息提取器
type ErrorExtractor struct{}

// Extract 提取错误信息
func (e *ErrorExtractor) Extract(entry *LogEntry) map[string]interface{} {
	fields := make(map[string]interface{})

	// 根据日志级别标记错误
	if entry.Level == "ERROR" || entry.Level == "FATAL" {
		fields["is_error"] = true

		// 尝试提取错误类型
		if errType, ok := entry.Fields["error_type"]; ok {
			fields["error_type"] = errType
		} else if errType, ok := entry.Fields["exception"]; ok {
			fields["error_type"] = errType
		}
	}

	return fields
}

// DomainEnricher 业务域增强器
type DomainEnricher struct{}

// Enrich 增强业务域信息
func (e *DomainEnricher) Enrich(ctx context.Context, entry *LogEntry) map[string]interface{} {
	fields := make(map[string]interface{})

	// 根据服务名推断业务域
	switch entry.Service {
	case "order-service", "payment-service":
		fields["business_domain"] = "commerce"
	case "user-service", "auth-service":
		fields["business_domain"] = "identity"
	case "content-service":
		fields["business_domain"] = "content"
	default:
		fields["business_domain"] = "general"
	}

	return fields
}

// TenantEnricher 租户信息增强器
type TenantEnricher struct{}

// Enrich 增强租户信息
func (e *TenantEnricher) Enrich(ctx context.Context, entry *LogEntry) map[string]interface{} {
	fields := make(map[string]interface{})

	// 从 Fields 或上下文提取租户 ID
	if tenantID, ok := entry.Fields["tenant_id"]; ok {
		fields["tenant_id"] = tenantID
	}
	if tenantID, ok := entry.Fields["org_id"]; ok {
		fields["tenant_id"] = tenantID
	}

	return fields
}

// TextAnalysisExtractor 文本分析结果提取器
type TextAnalysisExtractor struct{}

// Extract 从文本分析结果中提取字段
func (e *TextAnalysisExtractor) Extract(entry *LogEntry) map[string]interface{} {
	fields := make(map[string]interface{})

	// 从 Fields 中提取分析结果
	if v, ok := entry.Fields["sentiment_score"]; ok {
		fields["sentiment_score"] = v
	}
	if v, ok := entry.Fields["sentiment_label"]; ok {
		fields["sentiment_label"] = v
	}
	if v, ok := entry.Fields["language"]; ok {
		fields["language"] = v
	}
	if v, ok := entry.Fields["category"]; ok {
		fields["analysis_category"] = v
	}
	if v, ok := entry.Fields["keywords"]; ok {
		fields["keywords"] = v
	}
	if v, ok := entry.Fields["entities"]; ok {
		fields["entities"] = v
	}

	// 从消息中提取额外信息
	e.extractFromMessage(entry.Message, fields)

	return fields
}

// extractFromMessage 从消息中提取信息
func (e *TextAnalysisExtractor) extractFromMessage(message string, fields map[string]interface{}) {
	// 提取 IP 地址
	ipPattern := regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	if ip := ipPattern.FindString(message); ip != "" {
		fields["detected_ip"] = ip
	}

	// 提取 URL
	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	if url := urlPattern.FindString(message); url != "" {
		fields["detected_url"] = url
	}

	// 提取错误类型
	errorPattern := regexp.MustCompile(`(?i)(error|exception|failed|failure)[:\s]+([^\n]+)`)
	if matches := errorPattern.FindStringSubmatch(message); len(matches) > 2 {
		fields["error_detail"] = strings.TrimSpace(matches[2])
	}
}

// BusinessAttributeEnricher 业务属性增强器
type BusinessAttributeEnricher struct{}

// Enrich 增强业务属性
func (e *BusinessAttributeEnricher) Enrich(ctx context.Context, entry *LogEntry) map[string]interface{} {
	fields := make(map[string]interface{})

	// 基于服务名和消息内容推断业务属性
	service := strings.ToLower(entry.Service)
	message := strings.ToLower(entry.Message)

	// 推断 API 类型
	if strings.Contains(service, "api") || strings.Contains(service, "gateway") {
		fields["api_type"] = "gateway"
	} else if strings.Contains(service, "internal") {
		fields["api_type"] = "internal"
	}

	// 推断请求类型
	if strings.Contains(message, "get ") {
		fields["request_type"] = "read"
	} else if strings.Contains(message, "post ") || strings.Contains(message, "put ") || strings.Contains(message, "create") {
		fields["request_type"] = "write"
	}

	// 推断是否为关键业务
	criticalServices := []string{"payment", "order", "auth", "user"}
	for _, cs := range criticalServices {
		if strings.Contains(service, cs) {
			fields["is_critical"] = true
			break
		}
	}

	// 推断数据敏感性
	sensitiveKeywords := []string{"password", "token", "secret", "key", "credential", "privacy"}
	for _, kw := range sensitiveKeywords {
		if strings.Contains(message, kw) {
			fields["is_sensitive"] = true
			break
		}
	}

	return fields
}
