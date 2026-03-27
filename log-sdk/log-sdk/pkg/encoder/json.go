package encoder

import (
	"bytes"
	"io"
	"strconv"
	"sync"
	"time"
	"github.com/bytedance/sonic"
)

type JSONEncoder struct {
	config Config
	pool   sync.Pool
}

func NewJSONEncoder(cfg Config) *JSONEncoder {
	return &JSONEncoder{
		config: cfg,
		pool: sync.Pool{
			New: func() interface{} {
				// Pre-allocate buffer to avoid reallocation
				buf := bytes.NewBuffer(make([]byte, 0, 1024))
				return buf
			},
		},
	}
}

func DefaultJSONEncoder() *JSONEncoder {
	return NewJSONEncoder(DefaultConfig())
}

func (e *JSONEncoder) ContentType() string {
	return "application/json"
}

// Encode serializes a LogEntry to JSON without map allocation
func (e *JSONEncoder) Encode(entry LogEntry, w io.Writer) error {
	buf := e.pool.Get().(*bytes.Buffer)
	buf.Reset()
	defer e.pool.Put(buf)

	e.encodeToBuffer(entry, buf)
	_, err := w.Write(buf.Bytes())
	return err
}

func (e *JSONEncoder) EncodeToBytes(entry LogEntry) ([]byte, error) {
	buf := e.pool.Get().(*bytes.Buffer)
	buf.Reset()
	defer e.pool.Put(buf)

	e.encodeToBuffer(entry, buf)
	
	// Must copy bytes because buffer will be returned to pool
	res := make([]byte, buf.Len())
	copy(res, buf.Bytes())
	return res, nil
}

func (e *JSONEncoder) encodeToBuffer(entry LogEntry, buf *bytes.Buffer) {
	buf.WriteByte('{')
	
	// Timestamp
	buf.WriteString(`"timestamp":"`)
	if e.config.TimeFormat == time.RFC3339Nano {
		buf.WriteString(entry.Timestamp.Format(time.RFC3339Nano))
	} else {
		buf.WriteString(entry.Timestamp.Format(e.config.TimeFormat))
	}
	buf.WriteString(`",`)

	// Required fields
	buf.WriteString(`"level":"`)
	buf.WriteString(entry.Level)
	buf.WriteString(`","message":`)
	
	// Escape message string using sonic
	msgBytes, _ := sonic.Marshal(entry.Message)
	buf.Write(msgBytes)
	
	buf.WriteString(`,"service":"`)
	buf.WriteString(entry.Service)
	buf.WriteByte('"')

	// Optional fields
	if entry.Cluster != "" {
		buf.WriteString(`,"cluster":"`)
		buf.WriteString(entry.Cluster)
		buf.WriteByte('"')
	}
	if entry.Pod != "" {
		buf.WriteString(`,"pod":"`)
		buf.WriteString(entry.Pod)
		buf.WriteByte('"')
	}
	if entry.TraceID != "" {
		buf.WriteString(`,"trace_id":"`)
		buf.WriteString(entry.TraceID)
		buf.WriteByte('"')
	}
	if entry.SpanID != "" {
		buf.WriteString(`,"span_id":"`)
		buf.WriteString(entry.SpanID)
		buf.WriteByte('"')
	}

	// Custom fields
	for i := 0; i < entry.FieldsLen; i++ {
		f := &entry.Fields[i]
		buf.WriteString(`,"`)
		buf.WriteString(f.Key)
		buf.WriteString(`":`)
		
		switch f.Type {
		case IntType:
			buf.WriteString(strconv.FormatInt(f.Int, 10))
		case FloatType:
			buf.WriteString(strconv.FormatFloat(f.Float, 'f', -1, 64))
		case StringType:
			valBytes, _ := sonic.Marshal(f.Str)
			buf.Write(valBytes)
		case BoolType:
			if f.Int == 1 {
				buf.WriteString("true")
			} else {
				buf.WriteString("false")
			}
		case InterfaceType:
			valBytes, _ := sonic.Marshal(f.Obj)
			buf.Write(valBytes)
		}
	}

	buf.WriteByte('}')
}
