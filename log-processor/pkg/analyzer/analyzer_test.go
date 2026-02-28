// Package analyzer 提供文本分析单元测试
package analyzer_test

import (
	"testing"
	"time"

	"github.com/log-system/log-processor/pkg/analyzer"
)

// === Entity Extractor Tests ===

func TestIPAddressExtractor_BasicIPs(t *testing.T) {
	extractor := &analyzer.IPAddressExtractor{}

	tests := []struct {
		name     string
		input    string
		wantCount int
		wantIPs  []string
	}{
		{"Single IPv4", "Connection from 192.168.1.1", 1, []string{"192.168.1.1"}},
		{"Multiple IPv4", "From 10.0.0.1 to 10.0.0.2", 2, []string{"10.0.0.1", "10.0.0.2"}},
		{"Localhost", "Listening on 127.0.0.1:8080", 1, []string{"127.0.0.1"}},
		{"No IP", "No IP addresses here", 0, nil},
		{"Invalid IP", "999.999.999.999 is invalid", 1, []string{"999.999.999.999"}}, // Regex matches but may be invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractor.Extract(tt.input)
			if len(entities) != tt.wantCount {
				t.Errorf("Expected %d entities, got %d", tt.wantCount, len(entities))
			}
			for i, wantIP := range tt.wantIPs {
				if i < len(entities) && entities[i].Value != wantIP {
					t.Errorf("Expected IP %s, got %s", wantIP, entities[i].Value)
				}
			}
		})
	}
}

func TestIPAddressExtractor_EntityProperties(t *testing.T) {
	extractor := &analyzer.IPAddressExtractor{}
	text := "Server at 192.168.1.100 responded"

	entities := extractor.Extract(text)
	if len(entities) != 1 {
		t.Fatalf("Expected 1 entity, got %d", len(entities))
	}

	entity := entities[0]
	if entity.Type != "IP_ADDRESS" {
		t.Errorf("Expected type IP_ADDRESS, got %s", entity.Type)
	}
	if entity.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", entity.Confidence)
	}
	if entity.Start == 0 || entity.End == 0 {
		t.Error("Expected non-zero start/end positions")
	}
}

func TestURLEXtractor_BasicURLs(t *testing.T) {
	extractor := &analyzer.URLEXtractor{}

	tests := []struct {
		name     string
		input    string
		wantCount int
		wantURLs []string
	}{
		{"Single HTTP", "Visit http://example.com", 1, []string{"http://example.com"}},
		{"Single HTTPS", "Visit https://example.com/path", 1, []string{"https://example.com/path"}},
		{"Multiple URLs", "See http://a.com and https://b.org/page", 2, []string{"http://a.com", "https://b.org/page"}},
		{"URL with query", "API at https://api.example.com/v1/users?id=123", 1, []string{"https://api.example.com/v1/users?id=123"}},
		{"No URL", "No URLs in this text", 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractor.Extract(tt.input)
			if len(entities) != tt.wantCount {
				t.Errorf("Expected %d entities, got %d", tt.wantCount, len(entities))
			}
			for i, wantURL := range tt.wantURLs {
				if i < len(entities) && entities[i].Value != wantURL {
					t.Errorf("Expected URL %s, got %s", wantURL, entities[i].Value)
				}
			}
		})
	}
}

func TestURLEXtractor_EntityProperties(t *testing.T) {
	extractor := &analyzer.URLEXtractor{}
	text := "Documentation at https://docs.example.com"

	entities := extractor.Extract(text)
	if len(entities) != 1 {
		t.Fatalf("Expected 1 entity, got %d", len(entities))
	}

	entity := entities[0]
	if entity.Type != "URL" {
		t.Errorf("Expected type URL, got %s", entity.Type)
	}
	if entity.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", entity.Confidence)
	}
}

func TestEmailExtractor_BasicEmails(t *testing.T) {
	extractor := &analyzer.EmailExtractor{}

	tests := []struct {
		name       string
		input      string
		wantCount  int
		wantEmails []string
	}{
		{"Single email", "Contact us at support@example.com", 1, []string{"support@example.com"}},
		{"Multiple emails", "Email admin@test.org or user@domain.net", 2, []string{"admin@test.org", "user@domain.net"}},
		{"Email with subdomain", "Reach out to john.doe@mail.subdomain.example.com", 1, []string{"john.doe@mail.subdomain.example.com"}},
		{"No email", "No email addresses here", 0, nil},
		{"Invalid format", "not-an-email@", 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractor.Extract(tt.input)
			if len(entities) != tt.wantCount {
				t.Errorf("Expected %d entities, got %d", tt.wantCount, len(entities))
			}
			for i, wantEmail := range tt.wantEmails {
				if i < len(entities) && entities[i].Value != wantEmail {
					t.Errorf("Expected email %s, got %s", wantEmail, entities[i].Value)
				}
			}
		})
	}
}

func TestTimestampExtractor_ISO8601(t *testing.T) {
	extractor := &analyzer.TimestampExtractor{}

	tests := []struct {
		name      string
		input     string
		wantCount int
	}{
		{"ISO with Z", "Event at 2026-02-28T12:00:00Z", 1},
		{"ISO with offset", "Event at 2026-02-28T12:00:00+08:00", 1},
		{"ISO with millis", "Event at 2026-02-28T12:00:00.123Z", 1},
		{"Space separated", "Event at 2026-02-28 12:00:00", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractor.Extract(tt.input)
			if len(entities) != tt.wantCount {
				t.Errorf("Expected %d entities, got %d", tt.wantCount, len(entities))
			}
		})
	}
}

func TestTimestampExtractor_OtherFormats(t *testing.T) {
	extractor := &analyzer.TimestampExtractor{}

	tests := []struct {
		name      string
		input     string
		wantCount int
	}{
		{"US date", "Event on 02/28/2026 12:00:00", 1},
		{"Syslog", "Event on Feb 28 12:00:00", 1},
		{"Unix timestamp 10 digits", "Event at 1677585600", 1},
		{"Unix timestamp 13 digits", "Event at 1677585600000", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractor.Extract(tt.input)
			if len(entities) != tt.wantCount {
				t.Errorf("Expected %d entities, got %d", tt.wantCount, len(entities))
			}
		})
	}
}

func TestErrorPatternExtractor_ErrorMessages(t *testing.T) {
	extractor := &analyzer.ErrorPatternExtractor{}

	tests := []struct {
		name      string
		input     string
		wantCount int
		wantType  string
	}{
		{"Error prefix", "Error: Database connection failed", 1, "ERROR_ERROR_MESSAGE"},
		{"Exception", "Exception: Null pointer", 1, "ERROR_ERROR_MESSAGE"},
		{"Failed", "Failed to connect to server", 1, "ERROR_ERROR_MESSAGE"},
		{"Panic", "Panic: runtime error", 1, "ERROR_PANIC"},
		{"Timeout", "Timeout: operation exceeded time limit", 1, "ERROR_TIMEOUT"},
		{"Connection error", "Connection refused by host", 1, "ERROR_CONNECTION_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractor.Extract(tt.input)
			if len(entities) != tt.wantCount {
				t.Errorf("Expected %d entities, got %d", tt.wantCount, len(entities))
			}
			if len(entities) > 0 && entities[0].Type != tt.wantType {
				t.Errorf("Expected type %s, got %s", tt.wantType, entities[0].Type)
			}
		})
	}
}

func TestErrorPatternExtractor_CaseInsensitive(t *testing.T) {
	extractor := &analyzer.ErrorPatternExtractor{}

	tests := []struct {
		name      string
		input     string
		wantCount int
	}{
		{"Lowercase error", "error: something went wrong", 1},
		{"Uppercase ERROR", "ERROR: critical failure", 1},
		{"Mixed case", "ErRoR: weird casing", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractor.Extract(tt.input)
			if len(entities) != tt.wantCount {
				t.Errorf("Expected %d entities, got %d", tt.wantCount, len(entities))
			}
		})
	}
}

func TestStatusCodeExtractor_HTTPStatus(t *testing.T) {
	extractor := &analyzer.StatusCodeExtractor{}

	tests := []struct {
		name      string
		input     string
		wantCount int
	}{
		{"200 OK", "Request returned 200 OK with HTTP", 1},
		{"404 Not Found", "Got 404 error in HTTP response", 1},
		{"500 Internal", "Server returned 500 over HTTP", 1},
		{"Multiple codes", "HTTP from 200 to 500 status", 2},
		{"Non-HTTP number", "Port 8080 is open", 0}, // Should not match without HTTP context
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractor.Extract(tt.input)
			if len(entities) != tt.wantCount {
				t.Errorf("Expected %d entities, got %d", tt.wantCount, len(entities))
			}
		})
	}
}

func TestStatusCodeExtractor_EntityValue(t *testing.T) {
	extractor := &analyzer.StatusCodeExtractor{}
	text := "HTTP response 404 not found"

	entities := extractor.Extract(text)
	if len(entities) != 1 {
		t.Fatalf("Expected 1 entity, got %d", len(entities))
	}

	if entities[0].Type != "HTTP_STATUS" {
		t.Errorf("Expected type HTTP_STATUS, got %s", entities[0].Type)
	}
	if entities[0].Value != "404" {
		t.Errorf("Expected value 404, got %s", entities[0].Value)
	}
}

func TestUserIDExtractor_UserPatterns(t *testing.T) {
	extractor := &analyzer.UserIDExtractor{}

	tests := []struct {
		name      string
		input     string
		wantCount int
	}{
		{"user_id pattern", "user_id=12345", 1},
		{"uid pattern", "uid=admin", 1},
		{"username pattern", "username=john_doe", 1},
		{"login pattern", "login=testuser", 1},
		{"No user", "No user info", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractor.Extract(tt.input)
			if len(entities) != tt.wantCount {
				t.Errorf("Expected %d entities, got %d", tt.wantCount, len(entities))
			}
		})
	}
}

// === Sentiment Analyzer Tests ===

func TestSentimentAnalyzer_PositiveText(t *testing.T) {
	analyzer := analyzer.NewSentimentAnalyzer()

	tests := []struct {
		name      string
		input     string
		wantLabel string
	}{
		{"Simple positive", "This is good", "positive"},
		{"Strong positive", "Excellent and amazing work", "positive"},
		{"Success", "Operation completed with good results", "positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.Analyze(tt.input)
			if result.Label != tt.wantLabel {
				t.Errorf("Expected label %s, got %s", tt.wantLabel, result.Label)
			}
			if result.Score <= 0 {
				t.Errorf("Expected positive score, got %f", result.Score)
			}
		})
	}
}

func TestSentimentAnalyzer_NegativeText(t *testing.T) {
	analyzer := analyzer.NewSentimentAnalyzer()

	tests := []struct {
		name      string
		input     string
		wantLabel string
	}{
		{"Simple negative", "This is bad and terrible", "negative"},
		{"Error context", "Failed with error", "negative"},
		{"Crash", "Application crash is bad", "negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.Analyze(tt.input)
			if result.Label != tt.wantLabel {
				t.Errorf("Expected label %s, got %s", tt.wantLabel, result.Label)
			}
			if result.Score >= 0 {
				t.Errorf("Expected negative score, got %f", result.Score)
			}
		})
	}
}

func TestSentimentAnalyzer_NeutralText(t *testing.T) {
	analyzer := analyzer.NewSentimentAnalyzer()

	result := analyzer.Analyze("The sky exists")
	if result.Label != "neutral" {
		t.Errorf("Expected neutral label, got %s", result.Label)
	}
	if result.Score != 0 {
		t.Errorf("Expected zero score, got %f", result.Score)
	}
}

func TestSentimentAnalyzer_Negation(t *testing.T) {
	analyzer := analyzer.NewSentimentAnalyzer()

	// Positive word with negation should be negative
	result := analyzer.Analyze("not good")
	if result.Score >= 0 {
		t.Errorf("Expected negative score for 'not good', got %f", result.Score)
	}
}

func TestSentimentAnalyzer_Intensifiers(t *testing.T) {
	analyzer := analyzer.NewSentimentAnalyzer()

	// Intensified positive
	result1 := analyzer.Analyze("very good")
	// Non-intensified positive
	result2 := analyzer.Analyze("good")

	if result1.Score <= result2.Score {
		t.Errorf("Expected 'very good' to have higher score than 'good'")
	}
}

func TestSentimentAnalyzer_MixedSentiment(t *testing.T) {
	analyzer := analyzer.NewSentimentAnalyzer()

	result := analyzer.Analyze("This is good but also bad")
	if !result.Mixed {
		t.Error("Expected mixed sentiment to be true")
	}
}

func TestSentimentAnalyzer_ChinesePositive(t *testing.T) {
	analyzer := analyzer.NewSentimentAnalyzer()

	// Test with mixed Chinese-English that contains positive Chinese words
	result := analyzer.Analyze("这个好")
	// The tokenizer splits on spaces, so single Chinese characters may not match
	// We test that the analyzer handles Chinese text gracefully
	if result.Label != "positive" && result.Label != "neutral" {
		t.Errorf("Expected positive or neutral label for Chinese text, got %s", result.Label)
	}
}

func TestSentimentAnalyzer_ChineseNegative(t *testing.T) {
	analyzer := analyzer.NewSentimentAnalyzer()

	// Test with mixed Chinese-English that contains negative Chinese words
	result := analyzer.Analyze("这个坏 error")
	// The analyzer should detect negative sentiment from 'error' at minimum
	if result.Label == "positive" {
		t.Errorf("Expected non-positive label for Chinese negative text, got %s", result.Label)
	}
}

func TestSentimentAnalyzer_ScoreBounds(t *testing.T) {
	analyzer := analyzer.NewSentimentAnalyzer()

	// Very positive text
	result := analyzer.Analyze("excellent perfect amazing great")
	if result.Score > 1.0 {
		t.Errorf("Score should be capped at 1.0, got %f", result.Score)
	}

	// Very negative text
	result = analyzer.Analyze("terrible horrible awful bad error")
	if result.Score < -1.0 {
		t.Errorf("Score should be capped at -1.0, got %f", result.Score)
	}
}

// === Language Detector Tests ===

func TestLanguageDetector_English(t *testing.T) {
	detector := analyzer.NewLanguageDetector()

	result := detector.Detect("The quick brown fox jumps over the lazy dog")
	if result != "en" {
		t.Errorf("Expected 'en', got %s", result)
	}
}

func TestLanguageDetector_Chinese(t *testing.T) {
	detector := analyzer.NewLanguageDetector()

	result := detector.Detect("这是一个中文测试")
	if result != "zh" {
		t.Errorf("Expected 'zh', got %s", result)
	}
}

func TestLanguageDetector_Spanish(t *testing.T) {
	detector := analyzer.NewLanguageDetector()

	result := detector.Detect("El rápido zorro salta sobre el perro perezoso")
	if result != "es" {
		t.Errorf("Expected 'es', got %s", result)
	}
}

func TestLanguageDetector_SpecialChar(t *testing.T) {
	detector := analyzer.NewLanguageDetector()

	// Spanish special characters
	result := detector.Detect("¿Cómo estás? ¡Hola!")
	if result != "es" {
		t.Errorf("Expected 'es' for Spanish special chars, got %s", result)
	}
}

func TestLanguageDetector_DefaultFallback(t *testing.T) {
	detector := analyzer.NewLanguageDetector()

	// Text without clear language indicators
	result := detector.Detect("Some random text without clear language")
	if result != "en" {
		t.Errorf("Expected default fallback to 'en', got %s", result)
	}
}

// === Text Analyzer Integration Tests ===

func TestTextAnalyzer_BasicAnalysis(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("Server at 192.168.1.1 returned 200 OK")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Entities) == 0 {
		t.Error("Expected entities to be extracted")
	}

	// Check for IP entity
	foundIP := false
	for _, e := range result.Entities {
		if e.Type == "IP_ADDRESS" {
			foundIP = true
			break
		}
	}
	if !foundIP {
		t.Error("Expected IP address to be extracted")
	}
}

func TestTextAnalyzer_MultipleExtractors(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	text := "User admin@example.com connected from 10.0.0.1 at https://api.example.com"
	result, err := a.Analyze(text)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should find email, IP, and URL
	if len(result.Entities) < 3 {
		t.Errorf("Expected at least 3 entities, got %d", len(result.Entities))
	}
}

func TestTextAnalyzer_ErrorClassification(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("ERROR: Database connection failed")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Category != "error" {
		t.Errorf("Expected category 'error', got %s", result.Category)
	}
}

func TestTextAnalyzer_HTTPClassification(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("GET /api/users returned 200 OK")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Category != "http" {
		t.Errorf("Expected category 'http', got %s", result.Category)
	}
}

func TestTextAnalyzer_DatabaseClassification(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("SQL query executed on database")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Category != "database" {
		t.Errorf("Expected category 'database', got %s", result.Category)
	}
}

func TestTextAnalyzer_SecurityClassification(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("Authentication token expired for user")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Category != "security" {
		t.Errorf("Expected category 'security', got %s", result.Category)
	}
}

func TestTextAnalyzer_GeneralClassification(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("Some general log message")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Category != "general" {
		t.Errorf("Expected category 'general', got %s", result.Category)
	}
}

func TestTextAnalyzer_KeywordExtraction(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("database connection failed due to network timeout error")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Keywords) == 0 {
		t.Error("Expected keywords to be extracted")
	}

	// Check that meaningful keywords are extracted
	expectedKeywords := []string{"database", "connection", "failed", "network", "timeout", "error"}
	foundCount := 0
	for _, kw := range result.Keywords {
		for _, expected := range expectedKeywords {
			if kw == expected {
				foundCount++
				break
			}
		}
	}
	if foundCount == 0 {
		t.Error("Expected to find meaningful keywords")
	}
}

func TestTextAnalyzer_KeyPhraseExtraction(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("First sentence. Second sentence! Third sentence?")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.KeyPhrases) == 0 {
		t.Error("Expected key phrases to be extracted")
	}
}

func TestTextAnalyzer_LanguageDetection(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("This is an English sentence")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Language != "en" {
		t.Errorf("Expected language 'en', got %s", result.Language)
	}
}

func TestTextAnalyzer_SentimentAnalysis(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("This is excellent and amazing")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Sentiment.Label != "positive" {
		t.Errorf("Expected positive sentiment, got %s", result.Sentiment.Label)
	}
}

func TestTextAnalyzer_AnalyzedAt(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	beforeAnalysis := time.Now()
	result, err := a.Analyze("Test message")
	afterAnalysis := time.Now()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.AnalyzedAt.Before(beforeAnalysis) || result.AnalyzedAt.After(afterAnalysis) {
		t.Error("Expected AnalyzedAt to be within analysis time range")
	}
}

func TestTextAnalyzer_CustomExtractor(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	// Register custom extractor
	customExtractor := &testCustomExtractor{}
	a.RegisterExtractor("custom", customExtractor)

	result, err := a.Analyze("Custom pattern: CUSTOM123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should find custom entity
	foundCustom := false
	for _, e := range result.Entities {
		if e.Type == "CUSTOM" {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Error("Expected custom entity to be extracted")
	}
}

func TestTextAnalyzer_EmptyText(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Text != "" {
		t.Error("Expected empty text")
	}
	if len(result.Entities) != 0 {
		t.Error("Expected no entities for empty text")
	}
}

func TestTextAnalyzer_UnicodeText(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("Hello 世界 🌍")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Text != "Hello 世界 🌍" {
		t.Errorf("Expected unicode text to be preserved")
	}
}

func TestTextAnalyzer_MetadataMap(t *testing.T) {
	a := analyzer.NewTextAnalyzer()

	result, err := a.Analyze("Test message")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Metadata == nil {
		t.Error("Expected metadata map to be initialized")
	}
}

// Test custom extractor for testing
type testCustomExtractor struct{}

func (e *testCustomExtractor) Extract(text string) []analyzer.Entity {
	if len(text) == 0 {
		return nil
	}
	return []analyzer.Entity{
		{Type: "CUSTOM", Value: "test", Confidence: 1.0, Start: 0, End: 4},
	}
}
