package async

import (
	"errors"
	"sync/atomic"
)

// RingBuffer is a lock-free circular buffer for LogMessage
// It supports multiple producers and single consumer (MPSC)
type RingBuffer struct {
	data []atomic.Pointer[LogMessage]
	cap  uint64
	head atomic.Uint64
	tail atomic.Uint64
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
		data: make([]atomic.Pointer[LogMessage], capacity),
		cap:  uint64(capacity),
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
func (r *RingBuffer) Push(item LogMessage) error {
	for {
		t := r.tail.Load()
		h := r.head.Load()
		if t-h >= r.cap {
			return ErrBufferFull
		}
		if r.tail.CompareAndSwap(t, t+1) {
			r.data[t%r.cap].Store(&item)
			return nil
		}
	}
}

// TryPush is alias for Push
func (r *RingBuffer) TryPush(item LogMessage) bool {
	return r.Push(item) == nil
}

// DrainAll pops all messages currently in the buffer
func (r *RingBuffer) DrainAll() []LogMessage {
	for {
		t := r.tail.Load()
		h := r.head.Load()
		if h == t {
			return nil
		}

		if r.head.CompareAndSwap(h, t) {
			out := make([]LogMessage, 0, t-h)
			for i := h; i < t; i++ {
				// 自旋等待并发的 TryPush 完成写入操作
				for {
					val := r.data[i%r.cap].Swap(nil)
					if val != nil {
						out = append(out, *val)
						break
					}
				}
			}
			return out
		}
	}
}

// PushBatch is not efficiently implementable in lock-free fashion without complex logic,
// but for backward compatibility in tests:
func (r *RingBuffer) PushBatch(msgs []LogMessage) int {
	count := 0
	for _, msg := range msgs {
		if err := r.Push(msg); err != nil {
			break
		}
		count++
	}
	return count
}

// Pop is for test compatibility
func (r *RingBuffer) Pop() (LogMessage, error) {
	for {
		t := r.tail.Load()
		h := r.head.Load()
		if h == t {
			return LogMessage{}, ErrBufferEmpty
		}
		if r.head.CompareAndSwap(h, h+1) {
			for {
				val := r.data[h%r.cap].Swap(nil)
				if val != nil {
					return *val, nil
				}
			}
		}
	}
}

// PopBatch is for test compatibility
func (r *RingBuffer) PopBatch(msgs []LogMessage) int {
	if len(msgs) == 0 {
		return 0
	}
	for {
		t := r.tail.Load()
		h := r.head.Load()
		if h == t {
			return 0
		}
		
		available := t - h
		toRead := uint64(len(msgs))
		if toRead > available {
			toRead = available
		}

		if r.head.CompareAndSwap(h, h+toRead) {
			for i := uint64(0); i < toRead; i++ {
				for {
					val := r.data[(h+i)%r.cap].Swap(nil)
					if val != nil {
						msgs[i] = *val
						break
					}
				}
			}
			return int(toRead)
		}
	}
}

// Len returns the number of messages in the buffer
func (r *RingBuffer) Len() int {
	t := r.tail.Load()
	h := r.head.Load()
	return int(t - h)
}

// Cap returns the capacity of the buffer
func (r *RingBuffer) Cap() int {
	return int(r.cap)
}

// IsEmpty returns true if the buffer is empty
func (r *RingBuffer) IsEmpty() bool {
	return r.Len() == 0
}

// IsFull returns true if the buffer is full
func (r *RingBuffer) IsFull() bool {
	return r.Len() >= int(r.cap)
}
