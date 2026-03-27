package logger

import (
	"testing"
	"github.com/log-system/log-sdk/pkg/encoder"
)

func TestLogEntryPool(t *testing.T) {
	// Test acquiring from pool
	entry1 := acquireLogEntry()
	if entry1 == nil {
		t.Fatal("acquireLogEntry returned nil")
	}

	if entry1.Fields == nil {
		t.Error("Fields map should be initialized")
	}

	// Modify entry and release
	entry1.Message = "test message"
	entry1.Level = "INFO"
	if len(entry1.Fields) == 0 {
		entry1.Fields = make([]encoder.Field, 1)
	}
	entry1.Fields[0] = F("key", "value")
	entry1.FieldsLen = 1

	releaseLogEntry(entry1)

	// Acquire again, should be reset
	entry2 := acquireLogEntry()

	if entry2.Message != "" {
		t.Errorf("Message should be empty after acquire, got %s", entry2.Message)
	}
	if entry2.Level != "" {
		t.Errorf("Level should be empty after acquire, got %s", entry2.Level)
	}

	// FieldsLen should be 0
	if entry2.FieldsLen != 0 {
		t.Errorf("FieldsLen should be 0 after acquire, got %d", entry2.FieldsLen)
	}

	releaseLogEntry(entry2)
}

func TestLogEntryPool_MultipleAcquire(t *testing.T) {
	entries := make([]*encoder.LogEntry, 10)

	// Acquire multiple
	for i := 0; i < 10; i++ {
		entries[i] = acquireLogEntry()
		if len(entries[i].Fields) == 0 {
			entries[i].Fields = make([]encoder.Field, 1)
		}
		entries[i].Fields[0] = F("index", i)
		entries[i].FieldsLen = 1
	}

	// Release all
	for i := 0; i < 10; i++ {
		releaseLogEntry(entries[i])
	}

	// Acquire again and verify
	for i := 0; i < 10; i++ {
		entry := acquireLogEntry()
		if entry.FieldsLen != 0 {
			t.Error("Fields should be reset")
		}
		releaseLogEntry(entry)
	}
}

// BenchmarkLogEntryPool benchmarks pool performance
func BenchmarkLogEntryPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := acquireLogEntry()
		entry.Fields[0] = F("key", "value"); entry.FieldsLen=1
		releaseLogEntry(entry)
	}
}

// BenchmarkLogEntryWithoutPool benchmarks allocation without pool
func BenchmarkLogEntryWithoutPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := &encoder.LogEntry{
			Fields: make([]encoder.Field, 8),
		}
		entry.Fields[0] = F("key", "value"); entry.FieldsLen=1
		_ = entry
	}
}
