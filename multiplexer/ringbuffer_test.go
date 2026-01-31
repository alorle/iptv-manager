package multiplexer

import (
	"bytes"
	"sync"
	"testing"
)

func TestRingBuffer_WriteRead(t *testing.T) {
	rb := NewRingBuffer(10)

	// Write some data
	data := []byte("hello")
	n := rb.Write(data)
	if n != len(data) {
		t.Errorf("Write() = %d, want %d", n, len(data))
	}

	// Read it back
	buf := make([]byte, 10)
	n = rb.Read(buf)
	if n != len(data) {
		t.Errorf("Read() = %d, want %d", n, len(data))
	}
	if !bytes.Equal(buf[:n], data) {
		t.Errorf("Read() = %q, want %q", buf[:n], data)
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer(5)

	// Write more data than buffer size
	data1 := []byte("hello")
	rb.Write(data1)

	data2 := []byte("world")
	rb.Write(data2)

	// Should only have last 5 bytes ("world")
	result := rb.ReadAll()
	expected := []byte("world")
	if !bytes.Equal(result, expected) {
		t.Errorf("ReadAll() = %q, want %q", result, expected)
	}
}

func TestRingBuffer_PeekAll(t *testing.T) {
	rb := NewRingBuffer(10)

	data := []byte("hello")
	rb.Write(data)

	// Peek should not consume data
	result1 := rb.PeekAll()
	result2 := rb.PeekAll()

	if !bytes.Equal(result1, result2) {
		t.Errorf("PeekAll() consumed data: first=%q, second=%q", result1, result2)
	}

	if !bytes.Equal(result1, data) {
		t.Errorf("PeekAll() = %q, want %q", result1, data)
	}

	// Read should still work
	buf := make([]byte, 10)
	n := rb.Read(buf)
	if !bytes.Equal(buf[:n], data) {
		t.Errorf("Read() after PeekAll() = %q, want %q", buf[:n], data)
	}
}

func TestRingBuffer_Available(t *testing.T) {
	rb := NewRingBuffer(10)

	// Initially empty
	if rb.Available() != 0 {
		t.Errorf("Available() = %d, want 0", rb.Available())
	}

	// Write 5 bytes
	rb.Write([]byte("hello"))
	if rb.Available() != 5 {
		t.Errorf("Available() = %d, want 5", rb.Available())
	}

	// Write 5 more (fill buffer)
	rb.Write([]byte("world"))
	if rb.Available() != 10 {
		t.Errorf("Available() = %d, want 10", rb.Available())
	}

	// Write 3 more (overflow, should still be 10)
	rb.Write([]byte("abc"))
	if rb.Available() != 10 {
		t.Errorf("Available() = %d, want 10", rb.Available())
	}
}

func TestRingBuffer_Reset(t *testing.T) {
	rb := NewRingBuffer(10)

	rb.Write([]byte("hello"))
	rb.Reset()

	if rb.Available() != 0 {
		t.Errorf("Available() after Reset() = %d, want 0", rb.Available())
	}

	result := rb.ReadAll()
	if result != nil {
		t.Errorf("ReadAll() after Reset() = %q, want nil", result)
	}
}

func TestRingBuffer_Concurrent(t *testing.T) {
	rb := NewRingBuffer(1024)
	var wg sync.WaitGroup

	// Multiple concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			data := []byte{byte(id)}
			for j := 0; j < 100; j++ {
				rb.Write(data)
			}
		}(i)
	}

	// Multiple concurrent readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 10)
			for j := 0; j < 100; j++ {
				rb.Read(buf)
			}
		}()
	}

	wg.Wait()

	// Should not panic and should have some data
	available := rb.Available()
	if available < 0 || available > rb.Size() {
		t.Errorf("Available() = %d, out of valid range [0, %d]", available, rb.Size())
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer(10)

	// Read from empty buffer
	buf := make([]byte, 10)
	n := rb.Read(buf)
	if n != 0 {
		t.Errorf("Read() from empty buffer = %d, want 0", n)
	}

	// ReadAll from empty buffer
	result := rb.ReadAll()
	if result != nil {
		t.Errorf("ReadAll() from empty buffer = %q, want nil", result)
	}

	// PeekAll from empty buffer
	result = rb.PeekAll()
	if result != nil {
		t.Errorf("PeekAll() from empty buffer = %q, want nil", result)
	}
}

func TestRingBuffer_Wraparound(t *testing.T) {
	rb := NewRingBuffer(5)

	// Fill buffer
	rb.Write([]byte("12345"))

	// Read some
	buf := make([]byte, 3)
	rb.Read(buf) // Should read "123"

	// Write more (causing wraparound)
	rb.Write([]byte("67"))

	// Read all
	result := rb.ReadAll()
	expected := []byte("4567")
	if !bytes.Equal(result, expected) {
		t.Errorf("ReadAll() after wraparound = %q, want %q", result, expected)
	}
}

func TestRingBuffer_PartialRead(t *testing.T) {
	rb := NewRingBuffer(10)

	// Write 5 bytes
	rb.Write([]byte("hello"))

	// Read 3 bytes
	buf := make([]byte, 3)
	n := rb.Read(buf)
	if n != 3 {
		t.Errorf("Read() = %d, want 3", n)
	}
	if !bytes.Equal(buf[:n], []byte("hel")) {
		t.Errorf("Read() = %q, want %q", buf[:n], []byte("hel"))
	}

	// Read remaining 2 bytes
	n = rb.Read(buf)
	if n != 2 {
		t.Errorf("Read() = %d, want 2", n)
	}
	if !bytes.Equal(buf[:n], []byte("lo")) {
		t.Errorf("Read() = %q, want %q", buf[:n], []byte("lo"))
	}

	// Buffer should now be empty
	if rb.Available() != 0 {
		t.Errorf("Available() = %d, want 0", rb.Available())
	}
}
