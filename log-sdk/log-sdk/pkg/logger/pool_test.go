package logger

import (
	"testing"
	"time"
)

func TestLogEntryPool(t *testing.T) {
	// Test acquiring from pool
	entry1 := acquireLogEntry()
	if entry1 == nil {
		t.Fatal("acquireLogEntry() returned nil")
	}
	if entry1.Fields == nil {
		t.Error("Fields map should be initialized")
	}

	// Set some data
	entry1.Level = "INFO"
	entry1.Message = "test message"
	entry1.Fields["key"] = "value"
	entry1.Timestamp = time.Now()

	// Release back to pool
	releaseLogEntry(entry1)

	// Acquire again - should be reset
	entry2 := acquireLogEntry()
	if entry2 == nil {
		t.Fatal("acquireLogEntry() returned nil on second call")
	}

	// Fields should be empty after reset
	if len(entry2.Fields) != 0 {
		t.Errorf("Fields should be empty after acquire, got %d fields", len(entry2.Fields))
	}
	if entry2.Level != "" {
		t.Errorf("Level should be empty, got %s", entry2.Level)
	}
	if entry2.Message != "" {
		t.Errorf("Message should be empty, got %s", entry2.Message)
	}
}

func TestLogEntryPool_MultipleAcquire(t *testing.T) {
	// Test acquiring multiple entries
	entries := make([]*LogEntry, 10)
	for i := 0; i < 10; i++ {
		entries[i] = acquireLogEntry()
		entries[i].Fields["index"] = i
	}

	// Release all
	for _, entry := range entries {
		releaseLogEntry(entry)
	}

	// Acquire again - might get reused entries
	for i := 0; i < 10; i++ {
		entry := acquireLogEntry()
		if entry == nil {
			t.Fatal("acquireLogEntry() returned nil")
		}
		if len(entry.Fields) != 0 {
			t.Error("Fields should be reset")
		}
		releaseLogEntry(entry)
	}
}

func TestLogEntryPool_NilRelease(t *testing.T) {
	// Should not panic
	releaseLogEntry(nil)
}

// BenchmarkLogEntryPool benchmarks pool performance
func BenchmarkLogEntryPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := acquireLogEntry()
		entry.Level = "INFO"
		entry.Message = "benchmark"
		entry.Fields["key"] = "value"
		releaseLogEntry(entry)
	}
}

// BenchmarkLogEntryWithoutPool benchmarks without pool
func BenchmarkLogEntryWithoutPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := &LogEntry{
			Fields: make(map[string]interface{}, 8),
		}
		entry.Level = "INFO"
		entry.Message = "benchmark"
		entry.Fields["key"] = "value"
		// Let GC handle it
	}
}
