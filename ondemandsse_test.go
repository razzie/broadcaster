package broadcaster_test

import (
	"errors"
	"testing"

	. "github.com/razzie/broadcaster"

	"github.com/stretchr/testify/assert"
)

func TestOndemandSSEBroadcaster(t *testing.T) {
	src := func() (EventSource, error) {
		ch := make(chan string, 1)
		ch <- "a"
		close(ch)
		return NewTextEventSource(ch, ""), nil
	}
	b := NewOndemandSSEBroadcaster(src, WithBlocking(true))

	resp := runRequest(b, "/")
	expected := "data: a\n\n"
	assert.Equal(t, expected, <-resp)
}

func TestOndemandSSEBroadcasterError(t *testing.T) {
	const errorText = "source failed"
	src := func() (EventSource, error) {
		return nil, errors.New(errorText)
	}
	b := NewOndemandSSEBroadcaster(src, WithBlocking(true))

	resp := runRequest(b, "/")
	expected := errorText + "\n"
	assert.Equal(t, expected, <-resp)
}
