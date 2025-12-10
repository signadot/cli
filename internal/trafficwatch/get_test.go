package trafficwatch

import (
	"sync"
	"testing"
)

func TestSafeCloser(t *testing.T) {
	ch := make(chan struct{})
	sc := &safeCloser{ch: ch}

	// Test concurrent closes
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc.Close()
		}()
	}
	wg.Wait()

	// Verify channel is closed
	select {
	case <-ch:
		// Channel is closed, good
	default:
		t.Error("channel should be closed")
	}

	// Verify we can call Close() again without panic
	sc.Close()
}

func TestSafeCloserNil(t *testing.T) {
	var sc *safeCloser
	// Should not panic
	sc.Close()

	sc = &safeCloser{ch: nil}
	// Should not panic
	sc.Close()
}
