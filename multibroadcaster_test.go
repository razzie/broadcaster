package broadcaster_test

import (
	"sync"
	"testing"

	. "github.com/razzie/broadcaster"

	"github.com/stretchr/testify/assert"
)

func TestMultisourceBroadcaster(t *testing.T) {
	b := NewMultiBroadcaster(multiSource, WithBlocking(true), WithListenerBufferSize(1))

	l1, _, err := b.Listen("1")
	assert.NoError(t, err)
	assert.Equal(t, 1, <-l1)

	l2, _, err := b.Listen("2")
	assert.NoError(t, err)
	assert.Equal(t, 2, <-l2)

	_, _, err = b.Listen("bad key")
	assert.Equal(t, err, assert.AnError)
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
		return nil, nil, assert.AnError
	}
}
