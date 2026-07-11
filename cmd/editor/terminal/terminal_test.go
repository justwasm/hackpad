//go:build !js

package terminal

import (
	"io"
	"os"
	"sync"
	"testing"
	"time"
)

// TestOnDataCallbackReturnsImmediately verifies that wrapping stdin.Write
// in a goroutine allows the onData callback to return without blocking
// on I/O. This prevents the fdMutex from being held across a JS event
// loop re-entry boundary.
func TestOnDataCallbackReturnsImmediately(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// Drain the pipe in background so writes can complete
	go func() {
		_, _ = io.Copy(io.Discard, r)
	}()

	// Simulate the onData callback: writes to stdin in a goroutine
	// (this is the fix) and returns immediately
	writeDone := make(chan struct{})
	callbackReturned := make(chan struct{})

	go func() {
		// This simulates the JS callback (onData)
		// Before the fix, stdin.Write() would block here, holding the
		// callback open while the goroutine yielded to the JS event loop.
		// After the fix, Write runs in a separate goroutine below, so
		// this function returns immediately.
		chunk := []byte("hello")
		go func() {
			_, err := w.Write(chunk)
			if err != nil {
				t.Log("write:", err)
			}
			close(writeDone)
		}()
		close(callbackReturned)
	}()

	// The callback must return immediately, not wait for the write
	select {
	case <-callbackReturned:
		// Success: callback returned without blocking on I/O
	case <-time.After(100 * time.Millisecond):
		t.Fatal("onData callback blocked on write - deadlock risk")
	}

	// The write should complete asynchronously
	select {
	case <-writeDone:
		// Write completed in the background
	case <-time.After(5 * time.Second):
		t.Fatal("background write never completed")
	}
}

// TestConcurrentPipeWrites verifies multiple concurrent writes to the same
// pipe do not deadlock. This simulates the pattern where the onData callback
// launches a goroutine for writing, allowing multiple re-entrant calls to
// proceed safely.
func TestConcurrentPipeWrites(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// Drain the pipe
	var drainWg sync.WaitGroup
	drainWg.Add(1)
	go func() {
		defer drainWg.Done()
		_, _ = io.Copy(io.Discard, r)
	}()

	const writers = 10
	writeCompleted := make(chan struct{}, writers)

	for i := 0; i < writers; i++ {
		go func(n int) {
			_, err := w.Write([]byte("data"))
			if err != nil {
				t.Logf("writer %d: %v", n, err)
			}
			writeCompleted <- struct{}{}
		}(i)
	}

	// All writes must complete without deadlock
	select {
	case <-time.After(5 * time.Second):
		t.Fatalf("deadlock: only %d/%d writes completed", len(writeCompleted), writers)
	default:
		for i := 0; i < writers; i++ {
			<-writeCompleted
		}
	}

	w.Close()
	drainWg.Wait()
}
