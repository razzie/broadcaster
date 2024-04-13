package broadcaster_test

import (
	"testing"

	. "github.com/razzie/broadcaster"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOndemandBroadcaster(t *testing.T) {
	src := func() (<-chan int, error) {
		ch := make(chan int, 1)
		ch <- 1
		return ch, nil
	}
	b := NewOndemandBroadcaster(src, WithBlocking(true))

	l, _, err := b.Listen()
	require.NotNil(t, l)
	assert.NoError(t, err)

	n, ok := <-l
	assert.Equal(t, 1, n)
	assert.True(t, ok)
}

func TestOndemandBroadcasterError(t *testing.T) {
	src := func() (<-chan int, error) {
		return nil, assert.AnError
	}
	b := NewOndemandBroadcaster(src)

	_, _, err := b.Listen()
	assert.Equal(t, assert.AnError, err)
}
