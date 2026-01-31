package multiplexer

import (
	"sync"
)

// RingBuffer is a thread-safe circular buffer for storing stream data
// during reconnection periods. It maintains the last N bytes written
// to allow new clients to receive recent data even during upstream reconnection.
type RingBuffer struct {
	data     []byte
	size     int
	writePos int
	readPos  int
	full     bool
	mu       sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the specified size in bytes
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Write appends data to the ring buffer
// If the buffer is full, it will overwrite the oldest data
func (rb *RingBuffer) Write(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	written := 0
	for _, b := range data {
		rb.data[rb.writePos] = b

		// Calculate next write position
		nextWritePos := (rb.writePos + 1) % rb.size

		// If buffer is full or next position would overwrite read position, move read forward
		if rb.full {
			rb.readPos = (rb.readPos + 1) % rb.size
		} else if nextWritePos == rb.readPos {
			// Buffer just became full
			rb.full = true
		}

		rb.writePos = nextWritePos
		written++
	}

	return written
}

// Read reads up to len(p) bytes from the buffer
// Returns the number of bytes read (0 if buffer is empty)
func (rb *RingBuffer) Read(p []byte) int {
	if len(p) == 0 {
		return 0
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	// Check if buffer is empty
	if rb.readPos == rb.writePos && !rb.full {
		return 0
	}

	read := 0
	for read < len(p) {
		// Stop if we've read all available data
		if rb.readPos == rb.writePos && !rb.full {
			break
		}

		p[read] = rb.data[rb.readPos]
		rb.readPos = (rb.readPos + 1) % rb.size
		read++
		rb.full = false
	}

	return read
}

// ReadAll reads all available data from the buffer
// Returns a slice containing all available data
func (rb *RingBuffer) ReadAll() []byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	// Check if buffer is empty
	if rb.readPos == rb.writePos && !rb.full {
		return nil
	}

	var result []byte
	var available int

	if rb.full {
		// Buffer is full, entire buffer has data
		available = rb.size
	} else if rb.readPos < rb.writePos {
		// Normal case: data is contiguous from readPos to writePos
		available = rb.writePos - rb.readPos
	} else {
		// Wrapped around: data from readPos to end, then from 0 to writePos
		available = (rb.size - rb.readPos) + rb.writePos
	}

	result = make([]byte, available)

	if rb.readPos < rb.writePos || (rb.readPos == rb.writePos && rb.full && rb.size == available) {
		// Contiguous read or full buffer starting at position 0
		if rb.full && rb.readPos == rb.writePos {
			// Full buffer wraps around - read from readPos to end, then from 0 to writePos
			firstPart := rb.size - rb.readPos
			copy(result, rb.data[rb.readPos:])
			if rb.writePos > 0 {
				copy(result[firstPart:], rb.data[:rb.writePos])
			}
		} else {
			// Normal contiguous case
			copy(result, rb.data[rb.readPos:rb.writePos])
		}
	} else {
		// Wrapped around case: copy from readPos to end, then from 0 to writePos
		firstPart := rb.size - rb.readPos
		copy(result, rb.data[rb.readPos:])
		if rb.writePos > 0 {
			copy(result[firstPart:], rb.data[:rb.writePos])
		}
	}

	// Update read position to consume data
	rb.readPos = rb.writePos
	rb.full = false

	return result
}

// PeekAll returns all available data without consuming it
// This is useful for sending buffer contents to new clients during reconnection
func (rb *RingBuffer) PeekAll() []byte {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	// Check if buffer is empty
	if rb.readPos == rb.writePos && !rb.full {
		return nil
	}

	var result []byte
	var available int

	if rb.full {
		// Buffer is full, entire buffer has data
		available = rb.size
	} else if rb.readPos < rb.writePos {
		// Normal case: data is contiguous from readPos to writePos
		available = rb.writePos - rb.readPos
	} else {
		// Wrapped around: data from readPos to end, then from 0 to writePos
		available = (rb.size - rb.readPos) + rb.writePos
	}

	result = make([]byte, available)

	if rb.readPos < rb.writePos || (rb.readPos == rb.writePos && rb.full && rb.size == available) {
		// Contiguous read or full buffer starting at position 0
		if rb.full && rb.readPos == rb.writePos {
			// Full buffer wraps around - read from readPos to end, then from 0 to writePos
			firstPart := rb.size - rb.readPos
			copy(result, rb.data[rb.readPos:])
			if rb.writePos > 0 {
				copy(result[firstPart:], rb.data[:rb.writePos])
			}
		} else {
			// Normal contiguous case
			copy(result, rb.data[rb.readPos:rb.writePos])
		}
	} else {
		// Wrapped around case: copy from readPos to end, then from 0 to writePos
		firstPart := rb.size - rb.readPos
		copy(result, rb.data[rb.readPos:])
		if rb.writePos > 0 {
			copy(result[firstPart:], rb.data[:rb.writePos])
		}
	}

	return result
}

// Available returns the number of bytes available to read
func (rb *RingBuffer) Available() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.full {
		return rb.size
	}

	if rb.writePos >= rb.readPos {
		return rb.writePos - rb.readPos
	}

	return (rb.size - rb.readPos) + rb.writePos
}

// Reset clears the buffer
func (rb *RingBuffer) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.readPos = 0
	rb.writePos = 0
	rb.full = false
}

// Size returns the total capacity of the buffer
func (rb *RingBuffer) Size() int {
	return rb.size
}
