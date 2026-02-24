package encoder

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestJSONEncoder_Encode(t *testing.T) {
	encoder := DefaultJSONEncoder()

	entry := LogEntry{
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:     "INFO",
		Message:   "Test message",
		Service:   "test-service",
		Cluster:   "test-cluster",
		Pod:       "test-pod",
		TraceID:   "trace-123",
		SpanID:    "span-456",
		Fields: map[string]interface{}{
			"custom_field": "custom_value",
			"count":        42,
		},
	}

	var buf bytes.Buffer
	if err := encoder.Encode(entry, &buf); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Verify JSON is valid
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Check fields
	if result["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", result["level"])
	}
	if result["message"] != "Test message" {
		t.Errorf("message = %v, want 'Test message'", result["message"])
	}
	if result["service"] != "test-service" {
		t.Errorf("service = %v, want 'test-service'", result["service"])
	}
	if result["custom_field"] != "custom_value" {
		t.Errorf("custom_field = %v, want 'custom_value'", result["custom_field"])
	}
}

func TestJSONEncoder_EncodeToBytes(t *testing.T) {
	encoder := DefaultJSONEncoder()

	entry := LogEntry{
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:     "ERROR",
		Message:   "Error occurred",
		Service:   "test-service",
		Fields:    map[string]interface{}{},
	}

	data, err := encoder.EncodeToBytes(entry)
	if err != nil {
		t.Fatalf("EncodeToBytes failed: %v", err)
	}

	// Verify JSON is valid
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if result["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR", result["level"])
	}
}

func TestJSONEncoder_ContentType(t *testing.T) {
	encoder := DefaultJSONEncoder()
	if encoder.ContentType() != "application/json" {
		t.Errorf("ContentType = %v, want application/json", encoder.ContentType())
	}
}

func TestJSONEncoder_PrettyPrint(t *testing.T) {
	cfg := Config{PrettyPrint: true}
	encoder := NewJSONEncoder(cfg)

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Test",
		Service:   "test",
		Fields:    map[string]interface{}{},
	}

	var buf bytes.Buffer
	if err := encoder.Encode(entry, &buf); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\n") {
		t.Error("Pretty print should contain newlines")
	}
	if !strings.Contains(output, "  ") {
		t.Error("Pretty print should contain indentation")
	}
}

func TestJSONEncoder_FieldOverride(t *testing.T) {
	encoder := DefaultJSONEncoder()

	// Custom fields should not override standard fields
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Test",
		Service:   "test-service",
		Fields: map[string]interface{}{
			"level":   "DEBUG",    // Should not override
			"service": "override", // Should not override
			"custom":  "value",    // Should be included
		},
	}

	data, err := encoder.EncodeToBytes(entry)
	if err != nil {
		t.Fatalf("EncodeToBytes failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Standard fields should not be overridden
	if result["level"] != "INFO" {
		t.Errorf("level was overridden: got %v, want INFO", result["level"])
	}
	if result["service"] != "test-service" {
		t.Errorf("service was overridden: got %v, want test-service", result["service"])
	}
	// Custom field should be included
	if result["custom"] != "value" {
		t.Errorf("custom field missing: got %v, want value", result["custom"])
	}
}

func TestJSONEncoder_EmptyFields(t *testing.T) {
	encoder := DefaultJSONEncoder()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Test",
		Service:   "test",
		Cluster:   "", // Empty - should not be included
		Pod:       "", // Empty - should not be included
		TraceID:   "", // Empty - should not be included
		SpanID:    "", // Empty - should not be included
		Fields:    map[string]interface{}{},
	}

	data, err := encoder.EncodeToBytes(entry)
	if err != nil {
		t.Fatalf("EncodeToBytes failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Empty optional fields should not be in output
	if _, ok := result["cluster"]; ok {
		t.Error("cluster should not be in output when empty")
	}
	if _, ok := result["pod"]; ok {
		t.Error("pod should not be in output when empty")
	}
}

// BenchmarkJSONEncoder_Encode benchmarks the encoder
func BenchmarkJSONEncoder_Encode(b *testing.B) {
	encoder := DefaultJSONEncoder()
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Benchmark message",
		Service:   "benchmark",
		Cluster:   "cluster-1",
		Pod:       "pod-1",
		Fields: map[string]interface{}{
			"user_id": "user-123",
			"action":  "login",
			"count":   42,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encoder.EncodeToBytes(entry)
		if err != nil {
			b.Fatal(err)
		}
	}
}
