package sandbox

import (
	"sync/atomic"
	"testing"
)

func TestLimitedBuffer_WriteWithinLimit(t *testing.T) {
	called := int32(0)
	buf := newLimitedBuffer(100, func() {
		atomic.AddInt32(&called, 1)
	})

	n, err := buf.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if n != 11 {
		t.Errorf("expected 11 bytes written, got %d", n)
	}
	if buf.String() != "hello world" {
		t.Errorf("expected 'hello world', got %q", buf.String())
	}
	if buf.Exceeded() {
		t.Errorf("expected Exceeded() = false")
	}
	if atomic.LoadInt32(&called) != 0 {
		t.Errorf("expected callback not called")
	}
}

func TestLimitedBuffer_ExceedLimit(t *testing.T) {
	called := int32(0)
	limit := 10
	buf := newLimitedBuffer(limit, func() {
		atomic.AddInt32(&called, 1)
	})

	n, err := buf.Write([]byte("1234567890EXTRA"))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if n != 15 {
		t.Errorf("expected write to return full length 15, got %d", n)
	}
	if buf.String() != "1234567890" {
		t.Errorf("expected buffer truncated at limit '1234567890', got %q", buf.String())
	}
	if !buf.Exceeded() {
		t.Errorf("expected Exceeded() = true")
	}
	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("expected callback called exactly once, got %d", atomic.LoadInt32(&called))
	}

	// Additional write should not trigger callback again
	_, _ = buf.Write([]byte("MORE"))
	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("callback called again on subsequent writes!")
	}
}
