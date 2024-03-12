package broadcaster

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBroadcast(t *testing.T) {
	const numMessages = 5

	ch := make(chan int, numMessages)
	b := NewBroadcaster(ch)

	l1, _, err := b.Listen()
	assert.NoError(t, err)
	require.NotNil(t, l1)

	l2, _, err := b.Listen()
	assert.NoError(t, err)
	require.NotNil(t, l2)

	for i := 1; i <= numMessages; i++ {
		ch <- i
	}

	for i := 1; i <= numMessages; i++ {
		select {
		case m := <-l1:
			assert.Equal(t, i, m)
			assert.Equal(t, i, <-l2)
		case m := <-l2:
			assert.Equal(t, i, m)
			assert.Equal(t, i, <-l1)
		}
	}
}

func TestTimeout(t *testing.T) {
	const timeout = 100 * time.Millisecond

	ch := make(chan int, 1)
	b := NewBroadcaster(ch, WithTimeout(timeout))

	l, _, err := b.Listen()
	assert.NoError(t, err)
	require.NotNil(t, l)

	ch <- 1
	assert.Equal(t, 1, <-l)

	ch <- 2
	time.Sleep(timeout * 2)
	_, ok := <-l
	assert.False(t, ok)
}

func TestBlocking(t *testing.T) {
	ch := make(chan int)
	b := NewBroadcaster(ch, WithBlocking(true))

	select {
	case ch <- 1:
		t.Fatal("channel should be blocked")
	default:
	}

	l, _, err := b.Listen(WithBufferSize(1))
	assert.NoError(t, err)
	require.NotNil(t, l)

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
	assert.NoError(t, err)
	require.NotNil(t, l)
	require.NotNil(t, cancel)

	ch <- 1
	assert.Equal(t, 1, <-l)

	cancel()
	ch <- 2
	_, ok := <-l
	assert.False(t, ok)
}

func TestListenerContext(t *testing.T) {
	ch := make(chan int)
	b := NewBroadcaster(ch)

	ctx, cancel := context.WithCancel(context.Background())
	l, _, err := b.Listen(WithContext(ctx), WithBufferSize(1))
	assert.NoError(t, err)
	require.NotNil(t, l)

	ch <- 1

	res, ok := <-l
	assert.Equal(t, 1, res)
	assert.True(t, ok)

	cancel()

	_, ok = <-l
	assert.False(t, ok)
}
