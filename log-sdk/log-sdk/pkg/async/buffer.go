package async

import (
	"errors"
	"sync/atomic"
)

// RingBuffer is a lock-free circular buffer for LogMessage
// It supports multiple producers and single consumer (MPSC)
type RingBuffer struct {
	buffer   []LogMessage
	size     uint64
	mask     uint64
	head     atomic.Uint64 // read position
	tail     atomic.Uint64 // write position
}

// ErrBufferFull is returned when the buffer is full
var ErrBufferFull = errors.New("ring buffer full")

// ErrBufferEmpty is returned when the buffer is empty
var ErrBufferEmpty = errors.New("ring buffer empty")

// NewRingBuffer creates a new ring buffer with the given capacity
// Capacity must be a power of 2
func NewRingBuffer(capacity int) *RingBuffer {
	// Round up to power of 2
	capacity = nextPowerOf2(capacity)

	return &RingBuffer{
		buffer: make([]LogMessage, capacity),
		size:   uint64(capacity),
		mask:   uint64(capacity - 1),
	}
}

// nextPowerOf2 returns the next power of 2 >= n
func nextPowerOf2(n int) int {
	if n <= 1 {
		return 2
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// Push adds a message to the ring buffer
// Returns ErrBufferFull if the buffer is full
func (rb *RingBuffer) Push(msg LogMessage) error {
	tail := rb.tail.Load()
	head := rb.head.Load()

	// Check if buffer is full
	if tail-head >= rb.size {
		return ErrBufferFull
	}

	// Write to buffer
	idx := tail & rb.mask
	rb.buffer[idx] = msg

	// Update tail
	rb.tail.Store(tail + 1)

	return nil
}

// PushBatch adds multiple messages to the ring buffer
// Returns the number of messages successfully added
func (rb *RingBuffer) PushBatch(msgs []LogMessage) int {
	if len(msgs) == 0 {
		return 0
	}

	tail := rb.tail.Load()
	head := rb.head.Load()

	// Calculate available space
	available := rb.size - (tail - head)
	if available == 0 {
		return 0
	}

	// Limit to available space
	toWrite := len(msgs)
	if uint64(toWrite) > available {
		toWrite = int(available)
	}

	// Write messages
	for i := 0; i < toWrite; i++ {
		idx := (tail + uint64(i)) & rb.mask
		rb.buffer[idx] = msgs[i]
	}

	// Update tail
	rb.tail.Store(tail + uint64(toWrite))

	return toWrite
}

// Pop removes and returns a message from the ring buffer
// Returns ErrBufferEmpty if the buffer is empty
func (rb *RingBuffer) Pop() (LogMessage, error) {
	head := rb.head.Load()
	tail := rb.tail.Load()

	// Check if buffer is empty
	if head == tail {
		return LogMessage{}, ErrBufferEmpty
	}

	// Read from buffer
	idx := head & rb.mask
	msg := rb.buffer[idx]

	// Clear the slot to help GC
	rb.buffer[idx] = LogMessage{}

	// Update head
	rb.head.Store(head + 1)

	return msg, nil
}

// PopBatch removes and returns up to n messages from the ring buffer
// Returns the number of messages popped
func (rb *RingBuffer) PopBatch(msgs []LogMessage) int {
	if len(msgs) == 0 {
		return 0
	}

	head := rb.head.Load()
	tail := rb.tail.Load()

	// Calculate available messages
	available := tail - head
	if available == 0 {
		return 0
	}

	// Limit to requested size
	toRead := len(msgs)
	if uint64(toRead) > available {
		toRead = int(available)
	}

	// Read messages
	for i := 0; i < toRead; i++ {
		idx := (head + uint64(i)) & rb.mask
		msgs[i] = rb.buffer[idx]
		// Clear the slot
		rb.buffer[idx] = LogMessage{}
	}

	// Update head
	rb.head.Store(head + uint64(toRead))

	return toRead
}

// Len returns the number of messages in the buffer
func (rb *RingBuffer) Len() int {
	tail := rb.tail.Load()
	head := rb.head.Load()
	return int(tail - head)
}

// Cap returns the capacity of the buffer
func (rb *RingBuffer) Cap() int {
	return int(rb.size)
}

// IsEmpty returns true if the buffer is empty
func (rb *RingBuffer) IsEmpty() bool {
	return rb.Len() == 0
}

// IsFull returns true if the buffer is full
func (rb *RingBuffer) IsFull() bool {
	return rb.Len() >= int(rb.size)
}
