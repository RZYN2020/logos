package encoder

import (
	"io"
	"time"
)

// Encoder is the interface for log entry encoding
type Encoder interface {
	Encode(entry LogEntry, w io.Writer) error
	ContentType() string
}

// Field represents a structured log field
type Field struct {
	Key   string
	Type  FieldType
	Int   int64
	Float float64
	Str   string
	Obj   interface{}
}

type FieldType uint8

const (
	IntType FieldType = iota
	FloatType
	StringType
	BoolType
	InterfaceType
)

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
	// Use slice instead of map to avoid allocation
	Fields    []Field
	FieldsLen int
	
	File      string `json:"-"`
	Line      int    `json:"-"`
	Function  string `json:"-"`
}

// Config holds encoder configuration
type Config struct {
	PrettyPrint bool
	TimeFormat  string
}

// DefaultConfig returns the default encoder configuration
func DefaultConfig() Config {
	return Config{
		PrettyPrint: false,
		TimeFormat:  time.RFC3339Nano,
	}
}
