package broadcaster

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var ErrBroadcasterClosed = fmt.Errorf("broadcaster is closed")

type Broadcaster[T any] interface {
	Listen(opts ...ListenerOption) (ch <-chan T, closer func(), err error)
	IsClosed() bool
}

type broadcaster[In, Out any] struct {
	broadcasterOptions
	input     <-chan In
	convert   func(In) (Out, error)
	listeners map[chan<- Out]bool
	reg       chan listenerRequest[Out]
	unreg     chan listenerRequest[Out]
	closed    chan struct{}
}

type listenerRequest[T any] struct {
	channel chan<- T
	done    chan struct{}
}

func NewBroadcaster[T any](input <-chan T, opts ...BroadcasterOption) Broadcaster[T] {
	convert := func(t T) (T, error) {
		return t, nil
	}
	return NewConverterBroadcaster(input, convert, opts...)
}

func NewConverterBroadcaster[In, Out any](input <-chan In, convert func(In) (Out, error), opts ...BroadcasterOption) Broadcaster[Out] {
	b := &broadcaster[In, Out]{
		broadcasterOptions: defaultBroadcasterOptions,
		input:              input,
		convert:            convert,
		listeners:          make(map[chan<- Out]bool),
		reg:                make(chan listenerRequest[Out]),
		unreg:              make(chan listenerRequest[Out]),
		closed:             make(chan struct{}),
	}
	for _, opt := range opts {
		opt(&b.broadcasterOptions)
	}
	go b.run()
	return b
}

func (b *broadcaster[In, Out]) Listen(opts ...ListenerOption) (ch <-chan Out, closer func(), err error) {
	lisOpts := listenerOptions{
		bufSize: b.lisBufSize,
	}
	for _, opt := range opts {
		opt(&lisOpts)
	}

	listener := make(chan Out, lisOpts.bufSize)
	if err := b.register(listener); err != nil {
		return nil, nil, err
	}

	closer = sync.OnceFunc(func() { b.unregister(listener) })
	if lisOpts.ctx != nil {
		go func() {
			<-lisOpts.ctx.Done()
			closer()
		}()
	}

	return listener, closer, nil
}

func (b *broadcaster[In, Out]) IsClosed() bool {
	select {
	case <-b.closed:
		return true
	default:
		return false
	}
}

func (b *broadcaster[In, Out]) register(listener chan<- Out) error {
	req := listenerRequest[Out]{channel: listener, done: make(chan struct{})}
	select {
	case <-b.closed:
		return ErrBroadcasterClosed
	case b.reg <- req:
		select {
		case <-req.done:
		case <-b.closed:
		}
		return nil
	}
}

func (b *broadcaster[In, Out]) unregister(listener chan<- Out) error {
	req := listenerRequest[Out]{channel: listener, done: make(chan struct{})}
	select {
	case <-b.closed:
		return ErrBroadcasterClosed
	case b.unreg <- req:
		select {
		case <-req.done:
		case <-b.closed:
		}
		return nil
	}
}

func (b *broadcaster[In, Out]) run() {
	defer b.close()
	for {
		select {
		case m, ok := <-b.input:
			if !ok {
				return
			}
			if b.blocking && len(b.listeners) == 0 {
				req := <-b.reg
				b.listeners[req.channel] = true
				close(req.done)
			}
			b.broadcast(m)

		case req := <-b.reg:
			b.listeners[req.channel] = true
			close(req.done)

		case req := <-b.unreg:
			if _, ok := b.listeners[req.channel]; ok {
				delete(b.listeners, req.channel)
				close(req.channel)
			}
			close(req.done)
		}
	}
}

func (b *broadcaster[In, Out]) broadcast(in In) {
	if len(b.listeners) == 0 {
		return
	}

	out, err := b.convert(in)
	if err != nil {
		b.logger.Error("conversion failed", slog.Any("msg", in), slog.Any("err", err))
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
	if num_listeners := len(unreg); num_listeners > 0 {
		b.logger.Info("listener timeout", slog.Int("num_listeners", num_listeners))
	}
	for listener := range unreg {
		delete(b.listeners, listener)
		close(listener)
	}
}

func (b *broadcaster[In, Out]) close() {
	close(b.closed)
	for listener := range b.listeners {
		close(listener)
	}
	clear(b.listeners)
}
