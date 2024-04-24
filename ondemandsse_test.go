package broadcaster_test

import (
	"errors"
	"testing"

	. "github.com/razzie/broadcaster"
)

func TestOndemandSSEBroadcaster(t *testing.T) {
	src := func() (<-chan Event, error) {
		ch := make(chan string, 1)
		ch <- "a"
		close(ch)
		return NewTextEventSource(ch, ""), nil
	}
	b := NewOndemandSSEBroadcaster(src, WithBlocking(true))

	resp := runRequest(b, "/")
	if expected, got := "data: a\n\n", <-resp; expected != got {
		t.Errorf("expected <-resp == %q, got %q", expected, got)
	}
}

func TestOndemandSSEBroadcasterError(t *testing.T) {
	const errorText = "source failed"
	src := func() (<-chan Event, error) {
		return nil, errors.New(errorText)
	}
	b := NewOndemandSSEBroadcaster(src, WithBlocking(true))

	resp := runRequest(b, "/")
	if expected, got := errorText+"\n", <-resp; expected != got {
		t.Errorf("expected <-resp == %q, got %q", expected, got)
	}
}
