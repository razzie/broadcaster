package broadcaster_test

import (
	"errors"
	"sync"
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

	var res []byte
	var wg sync.WaitGroup
	wg.Add(1)
	go runRequest(b, &res, &wg)
	wg.Wait()

	expected := "data: a\n\n"
	assert.Equal(t, expected, string(res))
}

func TestOndemandSSEBroadcasterError(t *testing.T) {
	const errorText = "source failed"
	src := func() (EventSource, error) {
		return nil, errors.New(errorText)
	}
	b := NewOndemandSSEBroadcaster(src, WithBlocking(true))

	var res []byte
	var wg sync.WaitGroup
	wg.Add(1)
	go runRequest(b, &res, &wg)
	wg.Wait()

	assert.Equal(t, errorText+"\n", string(res))
}
