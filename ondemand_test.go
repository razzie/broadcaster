package broadcaster_test

import (
	"errors"
	"testing"

	. "github.com/razzie/broadcaster"
)

var ErrExpected = errors.New("expected error")

func TestOndemandBroadcaster(t *testing.T) {
	src := func() (<-chan int, error) {
		ch := make(chan int, 1)
		ch <- 1
		return ch, nil
	}
	b := NewOndemandBroadcaster(src, WithBlocking(true))

	l, _, err := b.Listen()
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}

	if val := <-l; val != 1 {
		t.Errorf("expected <-l == 1, but got %d", val)
	}
}

func TestOndemandBroadcasterError(t *testing.T) {
	src := func() (<-chan int, error) {
		return nil, ErrExpected
	}
	b := NewOndemandBroadcaster(src)

	_, _, err := b.Listen()
	if err != ErrExpected {
		t.Errorf("expected err == ErrExpected, got '%v'", err)
	}
}
