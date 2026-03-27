package encoder

import (
	"bytes"
	"io"
	"sync"

	"github.com/bytedance/sonic"
)

// JSONEncoder implements Encoder for JSON format using sonic
type JSONEncoder struct {
	config Config
	pool   sync.Pool
}

// NewJSONEncoder creates a new JSON encoder
func NewJSONEncoder(cfg Config) *JSONEncoder {
	return &JSONEncoder{
		config: cfg,
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

// DefaultJSONEncoder returns a JSON encoder with default configuration
func DefaultJSONEncoder() *JSONEncoder {
	return NewJSONEncoder(DefaultConfig())
}

// ContentType returns the content type for JSON encoding
func (e *JSONEncoder) ContentType() string {
	return "application/json"
}

// Encode serializes a LogEntry to JSON
func (e *JSONEncoder) Encode(entry LogEntry, w io.Writer) error {
	m := e.buildMap(entry)

	// Get buffer from pool
	buf := e.pool.Get().(*bytes.Buffer)
	buf.Reset()
	defer e.pool.Put(buf)

	// Encode to buffer
	var data []byte
	var err error
	if e.config.PrettyPrint {
		data, err = sonic.ConfigDefault.MarshalIndent(m, "", "  ")
	} else {
		data, err = sonic.ConfigDefault.Marshal(m)
	}
	if err != nil {
		return err
	}

	// Write to output
	_, err = w.Write(data)
	return err
}

// EncodeToBytes serializes a LogEntry to JSON bytes
func (e *JSONEncoder) EncodeToBytes(entry LogEntry) ([]byte, error) {
	m := e.buildMap(entry)

	if e.config.PrettyPrint {
		return sonic.ConfigDefault.MarshalIndent(m, "", "  ")
	}
	return sonic.ConfigDefault.Marshal(m)
}

func (e *JSONEncoder) buildMap(entry LogEntry) map[string]interface{} {
	m := make(map[string]interface{}, 10+len(entry.Fields))

	m["timestamp"] = entry.Timestamp.Format(e.config.TimeFormat)
	m["level"] = entry.Level
	m["message"] = entry.Message
	m["service"] = entry.Service

	if entry.Cluster != "" {
		m["cluster"] = entry.Cluster
	}
	if entry.Pod != "" {
		m["pod"] = entry.Pod
	}
	if entry.TraceID != "" {
		m["trace_id"] = entry.TraceID
	}
	if entry.SpanID != "" {
		m["span_id"] = entry.SpanID
	}

	// Merge custom fields
	for k, v := range entry.Fields {
		// Custom fields don't override standard fields
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return m
}
