// Package encoder provides log entry encoding interfaces and implementations
package encoder

import (
	"io"
	"time"
)

// Encoder is the interface for log entry encoding
type Encoder interface {
	// Encode serializes a log entry to the writer
	Encode(entry LogEntry, w io.Writer) error
	// ContentType returns the content type of the encoded data
	ContentType() string
}

// LogEntry represents a structured log entry for encoding
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Service   string
	Cluster   string
	Pod       string
	TraceID   string
	SpanID    string
	Fields    map[string]interface{}
}

// Config holds encoder configuration
type Config struct {
	// PrettyPrint enables pretty-printed JSON output
	PrettyPrint bool
	// TimeFormat specifies the time format for timestamps
	TimeFormat string
}

// DefaultConfig returns the default encoder configuration
func DefaultConfig() Config {
	return Config{
		PrettyPrint: false,
		TimeFormat:  time.RFC3339Nano,
	}
}
