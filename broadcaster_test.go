package broadcaster_test

import (
	"context"
	"testing"
	"time"

	. "github.com/razzie/broadcaster"
)

func TestBroadcast(t *testing.T) {
	const numMessages = 5

	ch := make(chan int, numMessages)
	b := NewBroadcaster(ch)

	l1, _, err := b.Listen()
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}

	l2, _, err := b.Listen()
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}

	for i := 1; i <= numMessages; i++ {
		ch <- i
	}

	for i := 1; i <= numMessages; i++ {
		// the order of listeners receiving the broadcasted value is random
		select {
		case val1 := <-l1:
			val2 := <-l2
			if val1 != val2 {
				t.Errorf("expected <-l1 == <-l2, but got values %d and %d", val1, val2)
			}
		case val2 := <-l2:
			val1 := <-l1
			if val1 != val2 {
				t.Errorf("expected <-l1 == <-l2, but got values %d and %d", val1, val2)
			}
		}
	}
}

func TestTimeout(t *testing.T) {
	const timeout = 10 * time.Millisecond

	ch := make(chan int, 1)
	b := NewBroadcaster(ch, WithTimeout(timeout))

	timeoutCallbackCalled := false
	timeoutCallback := func() {
		timeoutCallbackCalled = true
	}
	l, _, err := b.Listen(WithTimeoutCallback(timeoutCallback))
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}

	ch <- 1
	if val := <-l; val != 1 {
		t.Errorf("expected <-l == 1, but got %d", val)
	}

	ch <- 2
	time.Sleep(timeout * 2)

	if _, ok := <-l; ok {
		t.Errorf("expected l to be closed")
	}
	if !timeoutCallbackCalled {
		t.Error("timeoutCallback should have been called")
	}
}

func TestBlocking(t *testing.T) {
	ch := make(chan int)
	b := NewBroadcaster(ch, WithBlocking(true))

	select {
	case ch <- 1:
		t.Fatal("channel should be blocked")
	default:
	}

	_, _, err := b.Listen(WithBufferSize(1))
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}

	select {
	case ch <- 1:
	default:
		t.Fatal("channel should not block")
	}
}

func TestListenerClose(t *testing.T) {
	ch := make(chan int, 1)
	b := NewBroadcaster(ch)

	l, cancel, err := b.Listen()
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}

	ch <- 1
	if val := <-l; val != 1 {
		t.Errorf("expected <-l == 1, but got %d", val)
	}

	cancel()
	ch <- 2
	if _, ok := <-l; ok {
		t.Errorf("expected l to be closed")
	}
}

func TestListenerContext(t *testing.T) {
	ch := make(chan int)
	b := NewBroadcaster(ch)

	ctx, cancel := context.WithCancel(context.Background())
	l, _, err := b.Listen(WithContext(ctx), WithBufferSize(1))
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}

	ch <- 1
	if val := <-l; val != 1 {
		t.Errorf("expected <-l == 1, but got %d", val)
	}

	cancel()
	if _, ok := <-l; ok {
		t.Errorf("expected l to be closed")
	}
}

func TestBroadcasterClose(t *testing.T) {
	ch := make(chan int)
	b := NewBroadcaster(ch)

	select {
	case <-b.Done():
		t.Error("broadcaster's Done() channel should not be closed")
	default:
	}
	if b.IsClosed() {
		t.Error("broadcaster should not be closed")
	}

	close(ch)
	time.Sleep(time.Millisecond)

	select {
	case <-b.Done():
	default:
		t.Error("broadcaster's Done() channel should be closed")
	}
	if !b.IsClosed() {
		t.Error("broadcaster should be closed")
	}
}

func TestIdleTimeout(t *testing.T) {
	const idleTimeout = 10 * time.Millisecond
	ch := make(chan int)
	b := NewBroadcaster(ch, WithIdleTimeout(idleTimeout))

	if b.IsClosed() {
		t.Error("broadcaster should not be closed")
	}

	time.Sleep(2 * idleTimeout)

	if !b.IsClosed() {
		t.Error("broadcaster should be closed")
	}
}

func TestIdleTimeoutWithBlocking(t *testing.T) {
	const idleTimeout = 10 * time.Millisecond
	ch := make(chan int)
	b := NewBroadcaster(ch, WithIdleTimeout(idleTimeout), WithBlocking(true))

	if b.IsClosed() {
		t.Error("broadcaster should not be closed")
	}

	time.Sleep(2 * idleTimeout)

	if !b.IsClosed() {
		t.Error("broadcaster should be closed")
	}
}
