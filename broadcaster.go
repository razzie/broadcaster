package broadcaster

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var ErrBroadcasterClosed = fmt.Errorf("broadcaster is closed")

type Broadcaster[T any] struct {
	broadcasterOptions
	input              <-chan T
	listeners          map[chan<- T]bool
	reg                chan chan<- T
	unreg              chan chan<- T
	closed             chan struct{}
	numListeners       atomic.Int32
	numDroppedMessages atomic.Int32
	numTimeouts        atomic.Int32
}

type broadcasterOptions struct {
	timeout time.Duration
}

type BroadcasterOption func(*broadcasterOptions)

func WithTimeout(timeout time.Duration) BroadcasterOption {
	return func(bo *broadcasterOptions) {
		bo.timeout = timeout
	}
}

func NewBroadcaster[T any](input <-chan T, options ...BroadcasterOption) *Broadcaster[T] {
	b := &Broadcaster[T]{
		broadcasterOptions: broadcasterOptions{
			timeout: -1,
		},
		input:     input,
		listeners: make(map[chan<- T]bool),
		reg:       make(chan chan<- T),
		unreg:     make(chan chan<- T),
		closed:    make(chan struct{}),
	}
	for _, opt := range options {
		opt(&b.broadcasterOptions)
	}
	go b.run()
	return b
}

func (b *Broadcaster[T]) Listen() (ch <-chan T, closer func(), err error) {
	listener := make(chan T)
	if err := b.register(listener); err != nil {
		return nil, nil, err
	}
	return listener, func() { b.unregister(listener) }, nil
}

func (b *Broadcaster[T]) IsClosed() bool {
	select {
	case <-b.closed:
		return true
	default:
		return false
	}
}

func (b *Broadcaster[T]) NumListeners() int {
	return int(b.numListeners.Load())
}

func (b *Broadcaster[T]) NumDroppedMessages() int {
	return int(b.numDroppedMessages.Load())
}

func (b *Broadcaster[T]) NumTimeouts() int {
	return int(b.numTimeouts.Load())
}

func (b *Broadcaster[T]) register(listener chan<- T) error {
	select {
	case <-b.closed:
		return ErrBroadcasterClosed
	case b.reg <- listener:
		b.numListeners.Add(1)
		return nil
	}
}

func (b *Broadcaster[T]) unregister(listener chan<- T) error {
	select {
	case <-b.closed:
		return ErrBroadcasterClosed
	case b.unreg <- listener:
		return nil
	}
}

func (b *Broadcaster[T]) run() {
	defer b.close()
	for {
		select {
		case m, ok := <-b.input:
			if !ok {
				return
			}
			b.broadcast(m)

		case listener := <-b.reg:
			b.listeners[listener] = true

		case listener := <-b.unreg:
			if _, ok := b.listeners[listener]; ok {
				delete(b.listeners, listener)
				close(listener)
				b.numListeners.Add(-1)
			}
		}
	}
}

func (b *Broadcaster[T]) broadcast(m T) {
	if len(b.listeners) == 0 {
		b.numDroppedMessages.Add(1)
		return
	}

	if b.timeout < 0 {
		for listener := range b.listeners {
			listener <- m
		}
		return
	}

	timeout := make(chan struct{})
	time.AfterFunc(b.timeout, func() { close(timeout) })

	unreg := make(chan chan<- T, len(b.listeners))
	var wg sync.WaitGroup
	wg.Add(len(b.listeners))
	for listener := range b.listeners {
		listener := listener
		go func() {
			defer wg.Done()
			select { // try non-blocking first
			case listener <- m:
				return
			default:
			}
			select {
			case listener <- m:
			case <-timeout:
				unreg <- listener
			}
		}()
	}
	wg.Wait()
	close(unreg)
	numTimeouts := int32(len(unreg))
	for listener := range unreg {
		delete(b.listeners, listener)
		close(listener)
	}
	b.numListeners.Add(-numTimeouts)
	b.numTimeouts.Add(numTimeouts)
}

func (b *Broadcaster[T]) close() {
	close(b.closed)
	for listener := range b.listeners {
		close(listener)
	}
	clear(b.listeners)
}
