package broadcaster

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var (
	ErrBroadcasterClosed = fmt.Errorf("broadcaster is closed")

	defaultOptions = broadcasterOptions{
		logger:  slog.Default(),
		timeout: -1,
	}
)

type Broadcaster[In, Out any] struct {
	broadcasterOptions
	input     <-chan In
	transform func(In) (Out, error)
	listeners map[chan<- Out]bool
	reg       chan chan<- Out
	unreg     chan chan<- Out
	closed    chan struct{}
}

type broadcasterOptions struct {
	logger  *slog.Logger
	timeout time.Duration
}

type BroadcasterOption func(*broadcasterOptions)

func WithTimeout(timeout time.Duration) BroadcasterOption {
	return func(bo *broadcasterOptions) {
		bo.timeout = timeout
	}
}

func NewBroadcaster[T any](input <-chan T, options ...BroadcasterOption) *Broadcaster[T, T] {
	b := &Broadcaster[T, T]{
		broadcasterOptions: defaultOptions,
		input:              input,
		transform:          func(t T) (T, error) { return t, nil },
		listeners:          make(map[chan<- T]bool),
		reg:                make(chan chan<- T),
		unreg:              make(chan chan<- T),
		closed:             make(chan struct{}),
	}
	for _, opt := range options {
		opt(&b.broadcasterOptions)
	}
	go b.run()
	return b
}

func (b *Broadcaster[In, Out]) Listen() (ch <-chan Out, closer func(), err error) {
	listener := make(chan Out)
	if err := b.register(listener); err != nil {
		return nil, nil, err
	}
	return listener, func() { b.unregister(listener) }, nil
}

func (b *Broadcaster[In, Out]) IsClosed() bool {
	select {
	case <-b.closed:
		return true
	default:
		return false
	}
}

func (b *Broadcaster[In, Out]) register(listener chan<- Out) error {
	select {
	case <-b.closed:
		return ErrBroadcasterClosed
	case b.reg <- listener:
		return nil
	}
}

func (b *Broadcaster[In, Out]) unregister(listener chan<- Out) error {
	select {
	case <-b.closed:
		return ErrBroadcasterClosed
	case b.unreg <- listener:
		return nil
	}
}

func (b *Broadcaster[In, Out]) run() {
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

func (b *Broadcaster[In, Out]) broadcast(in In) {
	if len(b.listeners) == 0 {
		return
	}

	out, err := b.transform(in)
	if err != nil {
		b.logger.Error("transform failed", slog.Any("msg", in), slog.Any("err", err))
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(b.listeners))

	if b.timeout < 0 {
		for listener := range b.listeners {
			listener := listener
			go func() {
				defer wg.Done()
				listener <- out
			}()
		}
		wg.Wait()
		return
	}

	timeout := make(chan struct{})
	time.AfterFunc(b.timeout, func() { close(timeout) })

	unreg := make(chan chan<- Out, len(b.listeners))
	for listener := range b.listeners {
		listener := listener
		go func() {
			defer wg.Done()
			select { // try non-blocking first
			case listener <- out:
				return
			default:
			}
			select {
			case listener <- out:
			case <-timeout:
				unreg <- listener
			}
		}()
	}
	wg.Wait()
	close(unreg)
	for listener := range unreg {
		delete(b.listeners, listener)
		close(listener)
	}
}

func (b *Broadcaster[In, Out]) close() {
	close(b.closed)
	for listener := range b.listeners {
		close(listener)
	}
	clear(b.listeners)
}
