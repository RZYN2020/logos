package async

import (
	"testing"
)

func TestRingBuffer_PushPop(t *testing.T) {
	rb := NewRingBuffer(16)

	// Push some messages
	for i := 0; i < 5; i++ {
		msg := LogMessage{
			Topic: "test-topic",
			Key:   "key",
			Value: []byte("message"),
		}
		if err := rb.Push(msg); err != nil {
			t.Fatalf("Push failed: %v", err)
		}
	}

	if rb.Len() != 5 {
		t.Errorf("Len = %d, want 5", rb.Len())
	}

	// Pop messages
	for i := 0; i < 5; i++ {
		msg, err := rb.Pop()
		if err != nil {
			t.Fatalf("Pop failed: %v", err)
		}
		if msg.Topic != "test-topic" {
			t.Errorf("Topic = %s, want 'test-topic'", msg.Topic)
		}
	}

	if rb.Len() != 0 {
		t.Errorf("Len after pop = %d, want 0", rb.Len())
	}
}

func TestRingBuffer_PushFull(t *testing.T) {
	rb := NewRingBuffer(4) // capacity will be rounded to 4

	// Fill the buffer
	for i := 0; i < 4; i++ {
		msg := LogMessage{Topic: "test"}
		if err := rb.Push(msg); err != nil {
			t.Fatalf("Push %d failed: %v", i, err)
		}
	}

	// Next push should fail
	msg := LogMessage{Topic: "overflow"}
	if err := rb.Push(msg); err != ErrBufferFull {
		t.Errorf("Expected ErrBufferFull, got %v", err)
	}
}

func TestRingBuffer_PopEmpty(t *testing.T) {
	rb := NewRingBuffer(16)

	_, err := rb.Pop()
	if err != ErrBufferEmpty {
		t.Errorf("Expected ErrBufferEmpty, got %v", err)
	}
}

func TestRingBuffer_PushBatch(t *testing.T) {
	rb := NewRingBuffer(16)

	msgs := make([]LogMessage, 5)
	for i := range msgs {
		msgs[i] = LogMessage{Topic: "batch"}
	}

	n := rb.PushBatch(msgs)
	if n != 5 {
		t.Errorf("PushBatch returned %d, want 5", n)
	}

	if rb.Len() != 5 {
		t.Errorf("Len = %d, want 5", rb.Len())
	}
}

func TestRingBuffer_PushBatchPartial(t *testing.T) {
	rb := NewRingBuffer(4)

	// Fill the buffer partially
	for i := 0; i < 2; i++ {
		rb.Push(LogMessage{Topic: "filler"})
	}

	// Try to push more than available
	msgs := make([]LogMessage, 10)
	for i := range msgs {
		msgs[i] = LogMessage{Topic: "batch"}
	}

	n := rb.PushBatch(msgs)
	if n != 2 { // Only 2 slots available
		t.Errorf("PushBatch returned %d, want 2", n)
	}
}

func TestRingBuffer_PopBatch(t *testing.T) {
	rb := NewRingBuffer(16)

	// Push 5 messages
	for i := 0; i < 5; i++ {
		rb.Push(LogMessage{Topic: "test"})
	}

	// Pop 3 messages
	msgs := make([]LogMessage, 3)
	n := rb.PopBatch(msgs)
	if n != 3 {
		t.Errorf("PopBatch returned %d, want 3", n)
	}

	if rb.Len() != 2 {
		t.Errorf("Len = %d, want 2", rb.Len())
	}
}

func TestRingBuffer_PopBatchMoreThanAvailable(t *testing.T) {
	rb := NewRingBuffer(16)

	// Push 3 messages
	for i := 0; i < 3; i++ {
		rb.Push(LogMessage{Topic: "test"})
	}

	// Try to pop 10
	msgs := make([]LogMessage, 10)
	n := rb.PopBatch(msgs)
	if n != 3 {
		t.Errorf("PopBatch returned %d, want 3", n)
	}
}

func TestRingBuffer_WrapAround(t *testing.T) {
	rb := NewRingBuffer(4)

	// Fill buffer
	for i := 0; i < 4; i++ {
		rb.Push(LogMessage{Topic: "first"})
	}

	// Pop 2 (create space at beginning)
	for i := 0; i < 2; i++ {
		rb.Pop()
	}

	// Push 2 more (should wrap around)
	for i := 0; i < 2; i++ {
		if err := rb.Push(LogMessage{Topic: "second"}); err != nil {
			t.Fatalf("Push after wrap failed: %v", err)
		}
	}

	// Pop all and verify
	count := 0
	for !rb.IsEmpty() {
		msg, _ := rb.Pop()
		if msg.Topic != "first" && msg.Topic != "second" {
			t.Errorf("Unexpected topic: %s", msg.Topic)
		}
		count++
	}

	if count != 4 {
		t.Errorf("Total messages = %d, want 4", count)
	}
}

func TestRingBuffer_IsEmptyIsFull(t *testing.T) {
	rb := NewRingBuffer(4)

	if !rb.IsEmpty() {
		t.Error("New buffer should be empty")
	}
	if rb.IsFull() {
		t.Error("New buffer should not be full")
	}

	// Fill it
	for i := 0; i < 4; i++ {
		rb.Push(LogMessage{})
	}

	if rb.IsEmpty() {
		t.Error("Full buffer should not be empty")
	}
	if !rb.IsFull() {
		t.Error("Full buffer should be full")
	}
}

func TestRingBuffer_Cap(t *testing.T) {
	// Test capacity rounding to power of 2
	rb := NewRingBuffer(5)
	if rb.Cap() != 8 {
		t.Errorf("Cap(5) = %d, want 8", rb.Cap())
	}

	rb = NewRingBuffer(16)
	if rb.Cap() != 16 {
		t.Errorf("Cap(16) = %d, want 16", rb.Cap())
	}

	rb = NewRingBuffer(1)
	if rb.Cap() != 2 {
		t.Errorf("Cap(1) = %d, want 2", rb.Cap())
	}
}

// BenchmarkRingBuffer_Push benchmarks single push
func BenchmarkRingBuffer_Push(b *testing.B) {
	rb := NewRingBuffer(1024)
	msg := LogMessage{Topic: "benchmark", Value: []byte("data")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if rb.IsFull() {
			rb.Pop() // make room
		}
		rb.Push(msg)
	}
}

// BenchmarkRingBuffer_PushBatch benchmarks batch push
func BenchmarkRingBuffer_PushBatch(b *testing.B) {
	rb := NewRingBuffer(1024)
	msgs := make([]LogMessage, 100)
	for i := range msgs {
		msgs[i] = LogMessage{Topic: "benchmark"}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if rb.Len() > 512 {
			// Drain half
			out := make([]LogMessage, 512)
			rb.PopBatch(out)
		}
		rb.PushBatch(msgs)
	}
}
