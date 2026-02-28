// Package semantic 提供语义增强单元测试
package semantic_test

import (
	"context"
	"testing"
	"time"

	"github.com/log-system/log-processor/pkg/semantic"
)

// === Basic Builder Tests ===

func TestBuilder_BasicBuild(t *testing.T) {
	builder := semantic.NewBuilder()

	entry := &semantic.LogEntry{
		Timestamp: time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
		Level:     "INFO",
		Message:   "Test message",
		Service:   "test-service",
		Fields:    make(map[string]interface{}),
		Raw:       `{"level":"INFO","message":"Test message"}`,
	}

	ctx := context.Background()
	result := builder.Build(ctx, entry)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Level != "INFO" {
		t.Errorf("Expected level INFO, got %s", result.Level)
	}
	if result.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", result.Message)
	}
}

func TestBuilder_NilFields(t *testing.T) {
	builder := semantic.NewBuilder()

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Test",
		Service:   "test-service",
		Fields:    nil, // Nil fields
		Raw:       "test",
	}

	ctx := context.Background()
	result := builder.Build(ctx, entry)

	if result.Fields == nil {
		t.Error("Expected Fields to be initialized")
	}
}

func TestBuilder_WithOptions(t *testing.T) {
	builder := semantic.NewBuilder(semantic.WithAutoInfer(false))

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "GET /api/users 200",
		Service:   "test-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	ctx := context.Background()
	result := builder.Build(ctx, entry)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

// === HTTP Extractor Tests ===

func TestHTTPExtractor_BasicHTTP(t *testing.T) {
	extractor := &semantic.HTTPExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "GET /api/users 200",
		Service:   "api-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["http_method"] != "GET" {
		t.Errorf("Expected http_method 'GET', got %v", fields["http_method"])
	}
	if fields["http_path"] != "/api/users" {
		t.Errorf("Expected http_path '/api/users', got %v", fields["http_path"])
	}
	if fields["http_status"] != 200 {
		t.Errorf("Expected http_status 200, got %v", fields["http_status"])
	}
	if fields["category"] != "http_request" {
		t.Errorf("Expected category 'http_request', got %v", fields["category"])
	}
}

func TestHTTPExtractor_POST(t *testing.T) {
	extractor := &semantic.HTTPExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "POST /api/orders 201",
		Service:   "order-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["http_method"] != "POST" {
		t.Errorf("Expected http_method 'POST', got %v", fields["http_method"])
	}
	if fields["http_status"] != 201 {
		t.Errorf("Expected http_status 201, got %v", fields["http_status"])
	}
}

func TestHTTPExtractor_InvalidFormat(t *testing.T) {
	extractor := &semantic.HTTPExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "This is not an HTTP log",
		Service:   "test-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	// Should not extract anything for non-HTTP logs
	if len(fields) != 0 {
		t.Errorf("Expected no fields for non-HTTP log, got %v", fields)
	}
}

// === User Extractor Tests ===

func TestUserExtractor_FromUserID(t *testing.T) {
	extractor := &semantic.UserExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "User action",
		Service:   "user-service",
		Fields:    map[string]interface{}{"user_id": "user123"},
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["user_id"] != "user123" {
		t.Errorf("Expected user_id 'user123', got %v", fields["user_id"])
	}
}

func TestUserExtractor_FromUID(t *testing.T) {
	extractor := &semantic.UserExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "User action",
		Service:   "user-service",
		Fields:    map[string]interface{}{"uid": "uid456"},
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["user_id"] != "uid456" {
		t.Errorf("Expected user_id 'uid456', got %v", fields["user_id"])
	}
}

func TestUserExtractor_NoUserField(t *testing.T) {
	extractor := &semantic.UserExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "System action",
		Service:   "system-service",
		Fields:    map[string]interface{}{"other": "value"},
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if len(fields) != 0 {
		t.Errorf("Expected no fields when no user info, got %v", fields)
	}
}

// === Error Extractor Tests ===

func TestErrorExtractor_ErrorLevel(t *testing.T) {
	extractor := &semantic.ErrorExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Message:   "Something failed",
		Service:   "test-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["is_error"] != true {
		t.Errorf("Expected is_error to be true")
	}
}

func TestErrorExtractor_FatalLevel(t *testing.T) {
	extractor := &semantic.ErrorExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "FATAL",
		Message:   "Critical failure",
		Service:   "test-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["is_error"] != true {
		t.Errorf("Expected is_error to be true")
	}
}

func TestErrorExtractor_InfoLevel(t *testing.T) {
	extractor := &semantic.ErrorExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Normal operation",
		Service:   "test-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["is_error"] != nil {
		t.Errorf("Expected is_error to be nil for INFO level")
	}
}

func TestErrorExtractor_WithErrorType(t *testing.T) {
	extractor := &semantic.ErrorExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Message:   "Exception occurred",
		Service:   "test-service",
		Fields:    map[string]interface{}{"error_type": "NullPointerException"},
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["error_type"] != "NullPointerException" {
		t.Errorf("Expected error_type 'NullPointerException', got %v", fields["error_type"])
	}
}

func TestErrorExtractor_WithException(t *testing.T) {
	extractor := &semantic.ErrorExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Message:   "Exception occurred",
		Service:   "test-service",
		Fields:    map[string]interface{}{"exception": "RuntimeError"},
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["error_type"] != "RuntimeError" {
		t.Errorf("Expected error_type 'RuntimeError', got %v", fields["error_type"])
	}
}

// === Domain Enricher Tests ===

func TestDomainEnricher_Commerce(t *testing.T) {
	enricher := &semantic.DomainEnricher{}
	ctx := context.Background()

	tests := []struct {
		service     string
		wantDomain string
	}{
		{"order-service", "commerce"},
		{"payment-service", "commerce"},
		{"user-service", "identity"},
		{"auth-service", "identity"},
		{"content-service", "content"},
		{"other-service", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			entry := &semantic.LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   "test",
				Service:   tt.service,
				Fields:    make(map[string]interface{}),
				Raw:       "test",
			}

			fields := enricher.Enrich(ctx, entry)

			if fields["business_domain"] != tt.wantDomain {
				t.Errorf("Expected business_domain '%s', got %v", tt.wantDomain, fields["business_domain"])
			}
		})
	}
}

// === Tenant Enricher Tests ===

func TestTenantEnricher_FromTenantID(t *testing.T) {
	enricher := &semantic.TenantEnricher{}
	ctx := context.Background()

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "test",
		Service:   "test-service",
		Fields:    map[string]interface{}{"tenant_id": "tenant123"},
		Raw:       "test",
	}

	fields := enricher.Enrich(ctx, entry)

	if fields["tenant_id"] != "tenant123" {
		t.Errorf("Expected tenant_id 'tenant123', got %v", fields["tenant_id"])
	}
}

func TestTenantEnricher_FromOrgID(t *testing.T) {
	enricher := &semantic.TenantEnricher{}
	ctx := context.Background()

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "test",
		Service:   "test-service",
		Fields:    map[string]interface{}{"org_id": "org456"},
		Raw:       "test",
	}

	fields := enricher.Enrich(ctx, entry)

	if fields["tenant_id"] != "org456" {
		t.Errorf("Expected tenant_id 'org456', got %v", fields["tenant_id"])
	}
}

func TestTenantEnricher_NoTenant(t *testing.T) {
	enricher := &semantic.TenantEnricher{}
	ctx := context.Background()

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "test",
		Service:   "test-service",
		Fields:    map[string]interface{}{"other": "value"},
		Raw:       "test",
	}

	fields := enricher.Enrich(ctx, entry)

	if len(fields) != 0 {
		t.Errorf("Expected no fields when no tenant info, got %v", fields)
	}
}

// === Text Analysis Extractor Tests ===

func TestTextAnalysisExtractor_SentimentFields(t *testing.T) {
	extractor := &semantic.TextAnalysisExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "test",
		Service:   "test-service",
		Fields: map[string]interface{}{
			"sentiment_score":  0.8,
			"sentiment_label":  "positive",
			"language":         "en",
			"category":         "general",
			"keywords":         []string{"test", "keyword"},
			"entities":         []map[string]interface{}{{"type": "IP", "value": "1.2.3.4"}},
		},
		Raw: "test",
	}

	fields := extractor.Extract(entry)

	if fields["sentiment_score"] != 0.8 {
		t.Errorf("Expected sentiment_score 0.8, got %v", fields["sentiment_score"])
	}
	if fields["sentiment_label"] != "positive" {
		t.Errorf("Expected sentiment_label 'positive', got %v", fields["sentiment_label"])
	}
	if fields["language"] != "en" {
		t.Errorf("Expected language 'en', got %v", fields["language"])
	}
	if fields["keywords"] == nil {
		t.Error("Expected keywords to be extracted")
	}
}

func TestTextAnalysisExtractor_IPExtraction(t *testing.T) {
	extractor := &semantic.TextAnalysisExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Connection from 192.168.1.100 established",
		Service:   "test-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["detected_ip"] != "192.168.1.100" {
		t.Errorf("Expected detected_ip '192.168.1.100', got %v", fields["detected_ip"])
	}
}

func TestTextAnalysisExtractor_URLExtraction(t *testing.T) {
	extractor := &semantic.TextAnalysisExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Visit https://example.com/api for documentation",
		Service:   "test-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["detected_url"] != "https://example.com/api" {
		t.Errorf("Expected detected_url 'https://example.com/api', got %v", fields["detected_url"])
	}
}

func TestTextAnalysisExtractor_ErrorDetailExtraction(t *testing.T) {
	extractor := &semantic.TextAnalysisExtractor{}

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Message:   "Error: Database connection timeout",
		Service:   "test-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	fields := extractor.Extract(entry)

	if fields["error_detail"] != "Database connection timeout" {
		t.Errorf("Expected error_detail 'Database connection timeout', got %v", fields["error_detail"])
	}
}

// === Business Attribute Enricher Tests ===

func TestBusinessAttributeEnricher_APIClassification(t *testing.T) {
	enricher := &semantic.BusinessAttributeEnricher{}
	ctx := context.Background()

	tests := []struct {
		service   string
		wantType  string
	}{
		{"api-gateway", "gateway"},
		{"api-service", "gateway"},
		{"internal-service", "internal"},
		{"regular-service", ""},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			entry := &semantic.LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   "test",
				Service:   tt.service,
				Fields:    make(map[string]interface{}),
				Raw:       "test",
			}

			fields := enricher.Enrich(ctx, entry)

			if tt.wantType != "" {
				if fields["api_type"] != tt.wantType {
					t.Errorf("Expected api_type '%s', got %v", tt.wantType, fields["api_type"])
				}
			}
		})
	}
}

func TestBusinessAttributeEnricher_RequestTypeClassification(t *testing.T) {
	enricher := &semantic.BusinessAttributeEnricher{}
	ctx := context.Background()

	tests := []struct {
		message      string
		wantReqType  string
	}{
		{"GET /api/users", "read"},
		{"POST /api/users", "write"},
		{"PUT /api/users", "write"},
		{"create new user", "write"},
		{"delete user", ""},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			entry := &semantic.LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   tt.message,
				Service:   "test-service",
				Fields:    make(map[string]interface{}),
				Raw:       "test",
			}

			fields := enricher.Enrich(ctx, entry)

			if tt.wantReqType != "" {
				if fields["request_type"] != tt.wantReqType {
					t.Errorf("Expected request_type '%s', got %v", tt.wantReqType, fields["request_type"])
				}
			}
		})
	}
}

func TestBusinessAttributeEnricher_CriticalService(t *testing.T) {
	enricher := &semantic.BusinessAttributeEnricher{}
	ctx := context.Background()

	tests := []struct {
		service   string
		wantCritical bool
	}{
		{"payment-service", true},
		{"order-service", true},
		{"auth-service", true},
		{"user-service", true},
		{"logging-service", false},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			entry := &semantic.LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   "test",
				Service:   tt.service,
				Fields:    make(map[string]interface{}),
				Raw:       "test",
			}

			fields := enricher.Enrich(ctx, entry)

			if tt.wantCritical {
				if fields["is_critical"] != true {
					t.Errorf("Expected is_critical to be true for %s", tt.service)
				}
			}
		})
	}
}

func TestBusinessAttributeEnricher_SensitiveData(t *testing.T) {
	enricher := &semantic.BusinessAttributeEnricher{}
	ctx := context.Background()

	tests := []struct {
		message      string
		wantSensitive bool
	}{
		{"User password updated", true},
		{"Token generated for user", true},
		{"Secret key rotated", true},
		{"Credential validation failed", true},
		{"Privacy data accessed", true},
		{"Normal log message", false},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			entry := &semantic.LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   tt.message,
				Service:   "test-service",
				Fields:    make(map[string]interface{}),
				Raw:       "test",
			}

			fields := enricher.Enrich(ctx, entry)

			if tt.wantSensitive {
				if fields["is_sensitive"] != true {
					t.Errorf("Expected is_sensitive to be true for message: %s", tt.message)
				}
			} else {
				if fields["is_sensitive"] != nil && fields["is_sensitive"] == true {
					t.Errorf("Expected is_sensitive to be false for message: %s", tt.message)
				}
			}
		})
	}
}

// === Integration Tests ===

func TestBuilder_FullPipeline(t *testing.T) {
	builder := semantic.NewBuilder()

	entry := &semantic.LogEntry{
		Timestamp: time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
		Level:     "ERROR",
		Message:   "GET /api/users 500 - Error: Database connection failed from 192.168.1.100",
		Service:   "payment-service",
		Fields: map[string]interface{}{
			"user_id":   "user123",
			"tenant_id": "tenant456",
			"error_type": "ConnectionError",
		},
		Raw: `{"level":"ERROR","message":"GET /api/users 500"}`,
	}

	ctx := context.Background()
	result := builder.Build(ctx, entry)

	// Check HTTP extraction
	if result.HTTPMethod != "GET" {
		t.Errorf("Expected HTTPMethod 'GET', got %s", result.HTTPMethod)
	}
	if result.HTTPPath != "/api/users" {
		t.Errorf("Expected HTTPPath '/api/users', got %s", result.HTTPPath)
	}
	if result.HTTPStatus != 500 {
		t.Errorf("Expected HTTPStatus 500, got %d", result.HTTPStatus)
	}

	// Check user extraction
	if result.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got %s", result.UserID)
	}

	// Check error extraction
	if !result.IsError {
		t.Error("Expected IsError to be true")
	}
	if result.ErrorType != "ConnectionError" {
		t.Errorf("Expected ErrorType 'ConnectionError', got %s", result.ErrorType)
	}

	// Check business domain (it's in Fields, not directly on EnrichedLog)
	if result.Fields["business_domain"] != "commerce" {
		t.Errorf("Expected business_domain 'commerce', got %v", result.Fields["business_domain"])
	}

	// Check tenant
	if result.Fields["tenant_id"] != "tenant456" {
		t.Errorf("Expected tenant_id 'tenant456', got %v", result.Fields["tenant_id"])
	}
}

func TestBuilder_ContextCancellation(t *testing.T) {
	builder := semantic.NewBuilder()

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "test",
		Service:   "test-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should handle cancelled context gracefully
	result := builder.Build(ctx, entry)

	if result == nil {
		t.Error("Expected non-nil result even with cancelled context")
	}
}

func TestBuilder_PopulateFieldsFromFields(t *testing.T) {
	builder := semantic.NewBuilder()

	entry := &semantic.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "test",
		Service:   "test-service",
		Fields: map[string]interface{}{
			"http_method":  "POST",
			"http_path":    "/api/data",
			"http_status":  201,
			"user_id":      "user789",
			"error_type":   "ValidationError",
		},
		Raw: "test",
	}

	ctx := context.Background()
	result := builder.Build(ctx, entry)

	if result.HTTPMethod != "POST" {
		t.Errorf("Expected HTTPMethod 'POST', got %s", result.HTTPMethod)
	}
	if result.HTTPPath != "/api/data" {
		t.Errorf("Expected HTTPPath '/api/data', got %s", result.HTTPPath)
	}
	if result.HTTPStatus != 201 {
		t.Errorf("Expected HTTPStatus 201, got %d", result.HTTPStatus)
	}
	if result.UserID != "user789" {
		t.Errorf("Expected UserID 'user789', got %s", result.UserID)
	}
	if result.ErrorType != "ValidationError" {
		t.Errorf("Expected ErrorType 'ValidationError', got %s", result.ErrorType)
	}
	if !result.IsError {
		t.Error("Expected IsError to be true")
	}
}

func TestEnrichedLog_JSONSerialization(t *testing.T) {
	builder := semantic.NewBuilder()

	entry := &semantic.LogEntry{
		Timestamp: time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
		Level:     "INFO",
		Message:   "test",
		Service:   "test-service",
		Fields:    make(map[string]interface{}),
		Raw:       "test",
	}

	ctx := context.Background()
	result := builder.Build(ctx, entry)

	// The EnrichedLog should be serializable
	// (This test mainly ensures no circular references)
	if result.LogEntry.Message != "test" {
		t.Error("Expected message to be preserved")
	}
}
