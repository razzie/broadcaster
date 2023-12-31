package broadcaster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBroadcast(t *testing.T) {
	const numMessages = 5

	ch := make(chan int, numMessages)
	b := NewBroadcaster(ch)

	l1, _, err := b.Listen()
	assert.NoError(t, err)

	l2, _, err := b.Listen()
	assert.NoError(t, err)

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

	ch <- 1
	assert.Equal(t, 1, <-l)

	ch <- 2
	time.Sleep(timeout * 2)
	_, ok := <-l
	assert.False(t, ok)
}

func TestListenerClose(t *testing.T) {
	ch := make(chan int, 1)
	b := NewBroadcaster(ch)

	l, close, err := b.Listen()
	assert.NoError(t, err)

	ch <- 1
	assert.Equal(t, 1, <-l)

	close()
	ch <- 2
	_, ok := <-l
	assert.False(t, ok)
}
