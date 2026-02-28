// Package detector 提供格式检测单元测试
package detector_test

import (
	"strings"
	"testing"

	"github.com/log-system/log-processor/pkg/detector"
)

func TestFormatDetectorJSON(t *testing.T) {
	d := detector.NewFormatDetector()

	// JSON 日志
	logData := []byte(`{"timestamp": "2026-02-28T12:00:00Z", "level": "INFO", "message": "test"}`)
	result := d.Detect(logData)

	if result.Format != detector.FormatJSON {
		t.Errorf("Expected FormatJSON, got %v", result.Format)
	}
	if result.Confidence < 0.8 {
		t.Errorf("Expected confidence >= 0.8, got %f", result.Confidence)
	}
}

func TestFormatDetectorKeyValue(t *testing.T) {
	d := detector.NewFormatDetector()

	// KeyValue 日志
	logData := []byte(`timestamp=2026-02-28T12:00:00Z level=INFO message="test" service=api`)
	result := d.Detect(logData)

	if result.Format != detector.FormatKeyValue {
		t.Errorf("Expected FormatKeyValue, got %v", result.Format)
	}
}

func TestFormatDetectorSyslog(t *testing.T) {
	d := detector.NewFormatDetector()

	// Syslog 日志
	logData := []byte(`<34>Feb 28 12:00:00 myhost myservice[1234]: Test message`)
	result := d.Detect(logData)

	if result.Format != detector.FormatSyslog {
		t.Errorf("Expected FormatSyslog, got %v", result.Format)
	}
}

func TestFormatDetectorApache(t *testing.T) {
	d := detector.NewFormatDetector()

	// Apache 日志
	logData := []byte(`127.0.0.1 - frank [28/Feb/2026:12:00:00 +0000] "GET /api HTTP/1.1" 200 1234`)
	result := d.Detect(logData)

	if result.Format != detector.FormatApache {
		t.Errorf("Expected FormatApache, got %v", result.Format)
	}
}

func TestFormatDetectorNginx(t *testing.T) {
	d := detector.NewFormatDetector()

	// Nginx 日志
	logData := []byte(`127.0.0.1 - - [28/Feb/2026:12:00:00 +0000] "GET /api HTTP/1.1" 200 1234 "-" "Mozilla/5.0"`)
	result := d.Detect(logData)

	if result.Format != detector.FormatNginx {
		t.Errorf("Expected FormatNginx, got %v", result.Format)
	}
}

func TestFormatDetectorUnstructured(t *testing.T) {
	d := detector.NewFormatDetector()

	// 非结构化日志
	logData := []byte(`2026-02-28 12:00:00 ERROR Database connection failed: timeout after 30s`)
	result := d.Detect(logData)

	// 可能检测为 Unstructured 或其他格式
	if result.Confidence < 0.5 {
		t.Errorf("Expected confidence >= 0.5, got %f", result.Confidence)
	}
}

func TestUnstructuredDetectorBasic(t *testing.T) {
	d := detector.NewUnstructuredDetector(nil)

	// 非结构化文本
	text := []byte(`Some random text without clear structure`)
	result := d.Detect(text)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestUnstructuredDetectorStructured(t *testing.T) {
	d := detector.NewUnstructuredDetector(nil)

	// JSON 格式（不应该被检测为非结构化）
	text := []byte(`{"key": "value"}`)
	result := d.Detect(text)

	// JSON 不应该被检测为非结构化 - 应该返回 nil
	if result != nil {
		t.Errorf("JSON should return nil, got Format=%v with confidence=%f", result.Format, result.Confidence)
	}
}

func TestUnstructuredDetectorAnalyzeContent(t *testing.T) {
	d := detector.NewUnstructuredDetector(nil)

	text := `Error connecting to 192.168.1.1:8080. Visit https://example.com for help.`
	analysis := d.AnalyzeContent(text)

	if len(analysis.Entities) == 0 {
		t.Error("Expected entities to be extracted")
	}

	// 检查 IP 地址
	hasIP := false
	hasURL := false
	for _, entity := range analysis.Entities {
		if entity.Type == "IP_ADDRESS" {
			hasIP = true
		}
		if entity.Type == "URL" {
			hasURL = true
		}
	}

	if !hasIP {
		t.Error("Expected IP address to be extracted")
	}
	if !hasURL {
		t.Error("Expected URL to be extracted")
	}
}

func TestUnstructuredDetectorKeyPhrases(t *testing.T) {
	d := detector.NewUnstructuredDetector(nil)

	text := `Database connection failed timeout error occurred`
	analysis := d.AnalyzeContent(text)

	if len(analysis.KeyPhrases) == 0 {
		t.Error("Expected key phrases to be extracted")
	}
}

func TestUnstructuredDetectorSentences(t *testing.T) {
	d := detector.NewUnstructuredDetector(nil)

	text := `First sentence. Second sentence! Third sentence?`
	analysis := d.AnalyzeContent(text)

	if len(analysis.Sentences) != 3 {
		t.Errorf("Expected 3 sentences, got %d", len(analysis.Sentences))
	}
}

func TestDetectorTrainer(t *testing.T) {
	trainer := detector.NewDetectorTrainer()

	// 添加训练数据
	samples := []detector.TrainingSample{
		{
			Log:          `{"timestamp": "2026-02-28T12:00:00Z", "level": "INFO"}`,
			ExpectedFormat: detector.FormatJSON,
		},
		{
			Log:          `127.0.0.1 - - [28/Feb/2026:12:00:00 +0000] "GET /api HTTP/1.1" 200`,
			ExpectedFormat: detector.FormatNginx,
		},
	}

	for _, sample := range samples {
		trainer.AddTrainingData(sample)
	}
}

func TestFormatTypeString(t *testing.T) {
	tests := []struct {
		format detector.FormatType
		want   string
	}{
		{detector.FormatJSON, "json"},
		{detector.FormatKeyValue, "key_value"},
		{detector.FormatSyslog, "syslog"},
		{detector.FormatApache, "apache"},
		{detector.FormatNginx, "nginx"},
		{detector.FormatUnstructured, "unstructured"},
	}

	for _, test := range tests {
		// FormatType 是 string 类型，直接使用
		if string(test.format) != test.want {
			t.Errorf("Expected format %v to be '%s', got '%s'", test.format, test.want, test.format)
		}
	}
}

func TestDetectionResult(t *testing.T) {
	result := detector.DetectionResult{
		Format:     detector.FormatJSON,
		Confidence: 0.95,
		Metadata: map[string]interface{}{
			"fields": 5,
		},
	}

	if result.Format != detector.FormatJSON {
		t.Errorf("Expected FormatJSON, got %v", result.Format)
	}
	if result.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", result.Confidence)
	}
	if result.Metadata["fields"] != 5 {
		t.Errorf("Expected fields 5, got %v", result.Metadata["fields"])
	}
}

// === Edge Cases and Additional Tests ===

func TestFormatDetectorEmptyInput(t *testing.T) {
	d := detector.NewFormatDetector()

	// Empty input
	result := d.Detect([]byte{})
	if result == nil {
		t.Error("Expected non-nil result for empty input")
	}

	// Very short input
	result = d.Detect([]byte(`{`))
	if result == nil {
		t.Error("Expected non-nil result for short input")
	}
}

func TestFormatDetectorJSONEdgeCases(t *testing.T) {
	d := detector.NewFormatDetector()

	tests := []struct {
		name     string
		input    string
		wantOK   bool
		minConf  float64
	}{
		{"Minimal JSON", `{"a":"b"}`, true, 0.8},
		{"Nested JSON", `{"outer":{"inner":"value"}}`, true, 0.8},
		{"JSON Array", `[{"a":"b"}]`, false, 0},
		{"Invalid JSON", `{"a":"b"`, false, 0},
		{"JSON with common fields", `{"timestamp":"2026-02-28T12:00:00Z","level":"INFO","message":"test"}`, true, 0.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.Detect([]byte(tt.input))
			if result == nil {
				if tt.wantOK {
					t.Errorf("Expected non-nil result")
				}
				return
			}
			if tt.wantOK && result.Format != detector.FormatJSON {
				t.Errorf("Expected FormatJSON, got %v", result.Format)
			}
			if result.Confidence < tt.minConf {
				t.Errorf("Expected confidence >= %f, got %f", tt.minConf, result.Confidence)
			}
		})
	}
}

func TestFormatDetectorKeyValueEdgeCases(t *testing.T) {
	d := detector.NewFormatDetector()

	tests := []struct {
		name     string
		input    string
		wantKV   bool
		minConf  float64
	}{
		{"Single KV", `key=value`, false, 0},
		{"Multiple KV", `key1=value1 key2=value2`, true, 0.7},
		{"Mixed KV", `foo=bar baz:qux test=value`, true, 0.7},
		{"KV with quotes", `message="hello world" status=ok`, true, 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.Detect([]byte(tt.input))
			if result == nil {
				if tt.wantKV {
					t.Errorf("Expected non-nil result")
				}
				return
			}
			if tt.wantKV && result.Format != detector.FormatKeyValue {
				t.Errorf("Expected FormatKeyValue, got %v", result.Format)
			}
			if result.Confidence < tt.minConf {
				t.Errorf("Expected confidence >= %f, got %f", tt.minConf, result.Confidence)
			}
		})
	}
}

func TestFormatDetectorSyslogEdgeCases(t *testing.T) {
	d := detector.NewFormatDetector()

	tests := []struct {
		name       string
		input      string
		wantSyslog bool
		minConf    float64
	}{
		{"BSD Syslog", `Feb 28 12:00:00 myhost myservice: Test`, true, 0.6},
		{"RFC3164 Syslog", `<34>Feb 28 12:00:00 myhost myservice[1234]: Test`, true, 0.8},
		{"RFC5424 Syslog", `<34>1 2026-02-28T12:00:00.000Z myhost myservice - - Test`, false, 0},
		{"ISO Syslog", `2026-02-28T12:00:00.123456 myhost myservice: Test`, true, 0.8},
		{"Not Syslog", `random text without syslog pattern`, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.Detect([]byte(tt.input))
			if result == nil {
				if tt.wantSyslog {
					t.Errorf("Expected non-nil result")
				}
				return
			}
			if tt.wantSyslog && result.Format != detector.FormatSyslog {
				t.Errorf("Expected FormatSyslog, got %v", result.Format)
			}
		})
	}
}

func TestFormatDetectorApacheVsNginx(t *testing.T) {
	d := detector.NewFormatDetector()

	// Apache Common Log
	apacheCommon := `127.0.0.1 - frank [28/Feb/2026:12:00:00 +0000] "GET /api HTTP/1.1" 200 1234`
	result := d.Detect([]byte(apacheCommon))
	if result.Format != detector.FormatApache {
		t.Errorf("Apache Common should be detected as Apache, got %v", result.Format)
	}

	// Nginx Combined Log
	nginxCombined := `127.0.0.1 - - [28/Feb/2026:12:00:00 +0000] "GET /api HTTP/1.1" 200 1234 "-" "Mozilla/5.0"`
	result = d.Detect([]byte(nginxCombined))
	if result.Format != detector.FormatNginx {
		t.Errorf("Nginx Combined should be detected as Nginx, got %v", result.Format)
	}
}

func TestFormatDetectorConfidenceLevels(t *testing.T) {
	d := detector.NewFormatDetector()

	// High confidence JSON
	highConfJSON := `{"timestamp":"2026-02-28T12:00:00Z","level":"ERROR","message":"Database connection failed","service":"api","trace_id":"abc123"}`
	result := d.Detect([]byte(highConfJSON))
	if result.Confidence < 0.95 {
		t.Errorf("Expected high confidence for JSON with common fields, got %f", result.Confidence)
	}

	// Lower confidence JSON (fewer common fields)
	lowConfJSON := `{"foo":"bar","baz":"qux"}`
	result = d.Detect([]byte(lowConfJSON))
	if result.Confidence < 0.8 {
		t.Errorf("Expected at least 0.8 confidence for valid JSON, got %f", result.Confidence)
	}
}

func TestFormatDetectorRegistration(t *testing.T) {
	d := detector.NewFormatDetector()

	// Register custom detector
	customFormat := detector.FormatType("custom")
	customDetector := func(log []byte) *detector.DetectionResult {
		if len(log) > 100 {
			return &detector.DetectionResult{
				Format:     customFormat,
				Confidence: 0.9,
			}
		}
		return nil
	}
	d.RegisterFormat(customFormat, customDetector)

	// Long input should trigger custom detector
	longInput := make([]byte, 101)
	result := d.Detect(longInput)
	if result.Format != customFormat {
		t.Errorf("Expected custom format for long input, got %v", result.Format)
	}
}

func TestUnstructuredDetectorEntityExtraction(t *testing.T) {
	d := detector.NewUnstructuredDetector(nil)

	tests := []struct {
		name         string
		input        string
		expectedType string
	}{
		{"IP Address", "Connection from 192.168.1.100 refused", "IP_ADDRESS"},
		{"URL", "Visit https://example.com/api for docs", "URL"},
		{"Email", "Contact admin@example.com for help", "EMAIL"},
		{"Multiple IPs", "From 10.0.0.1 to 10.0.0.2", "IP_ADDRESS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := d.AnalyzeContent(tt.input)
			found := false
			for _, entity := range analysis.Entities {
				if entity.Type == tt.expectedType {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected to find entity type %s", tt.expectedType)
			}
		})
	}
}

func TestUnstructuredDetectorSentenceSplitting(t *testing.T) {
	d := detector.NewUnstructuredDetector(nil)

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"Simple sentences", "Hello world. How are you?", 2},
		{"Exclamation", "Stop! Don't do that", 2},
		{"Question", "What? Why?", 2},
		{"Mixed punctuation", "First. Second! Third?", 3},
		{"Abbreviations", "Mr. Smith went home.", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := d.AnalyzeContent(tt.input)
			if len(analysis.Sentences) != tt.expected {
				t.Errorf("Expected %d sentences, got %d", tt.expected, len(analysis.Sentences))
			}
		})
	}
}

func TestUnstructuredDetectorKeyPhraseExtraction(t *testing.T) {
	d := detector.NewUnstructuredDetector(nil)

	text := "The database connection failed due to network timeout error"
	analysis := d.AnalyzeContent(text)

	// Should extract meaningful phrases, filtering stop words
	if len(analysis.KeyPhrases) == 0 {
		t.Error("Expected key phrases to be extracted")
	}

	// Check that common stop words are filtered
	for _, phrase := range analysis.KeyPhrases {
		if phrase == "the" || phrase == "to" {
			t.Errorf("Stop word '%s' should be filtered", phrase)
		}
	}
}

func TestUnstructuredDetectorFeatureExtraction(t *testing.T) {
	d := detector.NewUnstructuredDetector(nil)

	// High text ratio
	highRatio := "This is a long piece of text with many words and no special characters"
	result := d.Detect([]byte(highRatio))
	if result == nil {
		t.Error("Expected result for high text ratio input")
	}

	// Low text ratio (many special chars)
	lowRatio := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
	result = d.Detect([]byte(lowRatio))
	if result != nil && result.Format == detector.FormatUnstructured && result.Confidence > 0.8 {
		t.Error("Low text ratio should have lower confidence")
	}
}

func TestDetectorTrainerTraining(t *testing.T) {
	trainer := detector.NewDetectorTrainer()
	detectorInst := detector.NewFormatDetector()

	// Add comprehensive training data
	samples := []detector.TrainingSample{
		{Log: `{"timestamp":"2026-02-28T12:00:00Z","level":"INFO","message":"test"}`, ExpectedFormat: detector.FormatJSON},
		{Log: `{"timestamp":"2026-02-28T12:00:01Z","level":"ERROR","message":"error"}`, ExpectedFormat: detector.FormatJSON},
		{Log: `127.0.0.1 - - [28/Feb/2026:12:00:00 +0000] "GET /api HTTP/1.1" 200 1234 "-" "Mozilla/5.0"`, ExpectedFormat: detector.FormatNginx},
		{Log: `127.0.0.1 - - [28/Feb/2026:12:00:01 +0000] "POST /submit HTTP/1.1" 201 567 "-" "curl/7.64"`, ExpectedFormat: detector.FormatNginx},
		{Log: `<34>Feb 28 12:00:00 myhost myservice[1234]: Test message`, ExpectedFormat: detector.FormatSyslog},
		{Log: `Database connection failed timeout error`, ExpectedFormat: detector.FormatUnstructured},
	}

	for _, sample := range samples {
		trainer.AddTrainingData(sample)
	}

	// Train and verify result
	result := trainer.Train(detectorInst)

	if result.Accuracy == 0 {
		t.Error("Expected non-zero accuracy")
	}

	if len(result.ConfusionMatrix) == 0 {
		t.Error("Expected confusion matrix to be populated")
	}
}

func TestDetectorTrainerOptimizeThresholds(t *testing.T) {
	trainer := detector.NewDetectorTrainer()
	detectorInst := detector.NewFormatDetector()

	// Add training data
	for i := 0; i < 10; i++ {
		trainer.AddTrainingData(detector.TrainingSample{
			Log:          `{"timestamp":"2026-02-28T12:00:00Z","level":"INFO"}`,
			ExpectedFormat: detector.FormatJSON,
		})
	}

	thresholds := trainer.OptimizeThresholds(detectorInst)

	if len(thresholds) == 0 {
		t.Error("Expected non-empty thresholds map")
	}
}

func TestDetectorTrainerReportGeneration(t *testing.T) {
	trainer := detector.NewDetectorTrainer()
	detectorInst := detector.NewFormatDetector()

	// Add training data
	trainer.AddTrainingData(detector.TrainingSample{
		Log:          `{"timestamp":"2026-02-28T12:00:00Z","level":"INFO"}`,
		ExpectedFormat: detector.FormatJSON,
	})

	result := trainer.Train(detectorInst)
	report := trainer.GenerateReport(result)

	if len(report) == 0 {
		t.Error("Expected non-empty report")
	}

	if !strings.Contains(report, "Total Samples") {
		t.Error("Report should contain total samples info")
	}
}

func TestDetectorTrainerExportMetrics(t *testing.T) {
	trainer := detector.NewDetectorTrainer()
	detectorInst := detector.NewFormatDetector()

	trainer.AddTrainingData(detector.TrainingSample{
		Log:          `{"timestamp":"2026-02-28T12:00:00Z","level":"INFO"}`,
		ExpectedFormat: detector.FormatJSON,
	})

	result := trainer.Train(detectorInst)
	metrics := trainer.ExportMetrics(result)

	if metrics["total_samples"] != 1 {
		t.Errorf("Expected total_samples to be 1, got %v", metrics["total_samples"])
	}
}

func TestUnstructuredDetectorCustomConfig(t *testing.T) {
	customConfig := &detector.UnstructuredConfig{
		MinTextRatio:    0.5,
		MinWordCount:    3,
		MaxPatternScore: 0.5,
		LanguageHints:   []string{"en"},
	}

	d := detector.NewUnstructuredDetector(customConfig)

	// Short text should be detected with lower min word count
	shortText := "Hello world test"
	result := d.Detect([]byte(shortText))
	if result == nil {
		t.Error("Expected result for short text with custom config")
	}
}

func TestFormatDetectorLowConfidence(t *testing.T) {
	d := detector.NewFormatDetector()

	// Ambiguous input
	ambiguous := "some text with 12:34:56 timestamp"
	result := d.Detect([]byte(ambiguous))

	// Should return some result even if low confidence
	if result == nil {
		t.Error("Expected some result for ambiguous input")
	}
}

func TestDetectionResultMetadata(t *testing.T) {
	result := &detector.DetectionResult{
		Format:     detector.FormatJSON,
		Confidence: 0.95,
		Metadata: map[string]interface{}{
			"fields":      5,
			"nested":      map[string]interface{}{"key": "value"},
			"array":       []int{1, 2, 3},
			"empty_array": []int{},
			"nil_value":   nil,
		},
	}

	if result.Metadata["fields"] != 5 {
		t.Errorf("Expected fields 5, got %v", result.Metadata["fields"])
	}

	nested, ok := result.Metadata["nested"].(map[string]interface{})
	if !ok || nested["key"] != "value" {
		t.Error("Expected nested map value")
	}
}
