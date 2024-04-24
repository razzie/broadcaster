package broadcaster_test

import (
	"sync"
	"testing"

	. "github.com/razzie/broadcaster"
)

func TestMultisourceBroadcaster(t *testing.T) {
	b := NewMultiBroadcaster(multiSource, WithBlocking(true), WithListenerBufferSize(1))

	_, _, err := b.Listen("1")
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}

	_, _, err = b.Listen("2")
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}

	_, _, err = b.Listen("bad key")
	if err != ErrExpected {
		t.Errorf("expected err == ErrExpected, got '%v'", err)
	}
}

func multiSource(key string) (<-chan int, CancelFunc, error) {
	switch key {
	case "1":
		ch := make(chan int, 1)
		ch <- 1
		cancel := sync.OnceFunc(func() { close(ch) })
		return ch, cancel, nil

	case "2":
		ch := make(chan int, 1)
		ch <- 2
		cancel := sync.OnceFunc(func() { close(ch) })
		return ch, cancel, nil

	default:
		return nil, nil, ErrExpected
	}
}
