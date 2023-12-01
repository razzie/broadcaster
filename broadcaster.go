package broadcaster

import (
	"fmt"
	"sync"
	"time"
)

var ErrBroadcasterClosed = fmt.Errorf("broadcaster is closed")

type Broadcaster[T any] struct {
	broadcasterOptions
	input     <-chan T
	listeners map[chan<- T]bool
	reg       chan chan<- T
	unreg     chan chan<- T
	closed    chan struct{}
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

func (b *Broadcaster[T]) register(listener chan<- T) error {
	select {
	case <-b.closed:
		return ErrBroadcasterClosed
	case b.reg <- listener:
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
			}
		}
	}
}

func (b *Broadcaster[T]) broadcast(m T) {
	if len(b.listeners) == 0 {
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
	var wg sync.WaitGroup
	wg.Add(len(b.listeners))
	for listener := range b.listeners {
		listener := listener
		go func() {
			defer wg.Done()
			select {
			case listener <- m:
			case <-timeout:
				go func() { b.unreg <- listener }()
			}
		}()
	}
	wg.Wait()
}

func (b *Broadcaster[T]) close() {
	close(b.closed)
	for listener := range b.listeners {
		close(listener)
	}
	clear(b.listeners)
}
