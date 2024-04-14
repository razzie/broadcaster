package broadcaster

import (
	"errors"
	"sync"
	"time"
)

var ErrBroadcasterClosed = errors.New("broadcaster is closed")

type CancelFunc func()

type Converter[In, Out any] func(In) (Out, bool)

type Broadcaster[T any] interface {
	Listen(opts ...ListenerOption) (<-chan T, CancelFunc, error)
	IsClosed() bool
	Done() <-chan struct{}
}

type broadcaster[In, Out any] struct {
	broadcasterOptions
	input     <-chan In
	convert   Converter[In, Out]
	listeners map[chan<- Out]listenerOptions
	reg       chan listenerRequest[Out]
	unreg     chan listenerRequest[Out]
	closed    chan struct{}
}

type listenerRequest[T any] struct {
	channel chan<- T
	opts    listenerOptions
	done    chan struct{}
}

func NewBroadcaster[T any](input <-chan T, opts ...BroadcasterOption) Broadcaster[T] {
	return NewConverterBroadcaster(input, noConversion, opts...)
}

func NewConverterBroadcaster[In, Out any](input <-chan In, convert Converter[In, Out], opts ...BroadcasterOption) Broadcaster[Out] {
	b := &broadcaster[In, Out]{
		broadcasterOptions: defaultBroadcasterOptions,
		input:              input,
		convert:            convert,
		listeners:          make(map[chan<- Out]listenerOptions),
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

func (b *broadcaster[In, Out]) Listen(opts ...ListenerOption) (<-chan Out, CancelFunc, error) {
	lisOpts := listenerOptions{
		bufSize: b.lisBufSize,
	}
	for _, opt := range opts {
		opt(&lisOpts)
	}

	listener := make(chan Out, lisOpts.bufSize)
	if err := b.register(listener, lisOpts); err != nil {
		return nil, nil, err
	}

	cancel := sync.OnceFunc(func() { b.unregister(listener) })
	if lisOpts.ctx != nil {
		go func() {
			<-lisOpts.ctx.Done()
			cancel()
		}()
	}

	return listener, cancel, nil
}

func (b *broadcaster[In, Out]) IsClosed() bool {
	select {
	case <-b.closed:
		return true
	default:
		return false
	}
}

func (b *broadcaster[In, Out]) Done() <-chan struct{} {
	return b.closed
}

func (b *broadcaster[In, Out]) register(listener chan<- Out, opts listenerOptions) error {
	req := listenerRequest[Out]{
		channel: listener,
		opts:    opts,
		done:    make(chan struct{}),
	}
	select {
	case <-b.closed:
		return ErrBroadcasterClosed
	case b.reg <- req:
		<-req.done
		return nil
	}
}

func (b *broadcaster[In, Out]) unregister(listener chan<- Out) {
	req := listenerRequest[Out]{
		channel: listener,
		done:    make(chan struct{}),
	}
	select {
	case <-b.closed:
	case b.unreg <- req:
		<-req.done
	}
}

func (b *broadcaster[In, Out]) run() {
	defer b.close()

	var idleTimer *time.Timer
	var idle <-chan time.Time
	if b.idleTimeout > 0 {
		idleTimer = time.NewTimer(b.idleTimeout)
		idle = idleTimer.C
	}

	for {
		if idleTimer != nil {
			if !idleTimer.Stop() {
				<-idleTimer.C
			}
			idleTimer.Reset(b.idleTimeout)
		}

		if b.blocking && len(b.listeners) == 0 {
			select {
			case req := <-b.reg:
				b.listeners[req.channel] = req.opts
				close(req.done)
			case <-idle:
				return
			}
		}

		select {
		case m, ok := <-b.input:
			if !ok {
				return
			}
			b.broadcast(m)

		case req := <-b.reg:
			b.listeners[req.channel] = req.opts
			close(req.done)

		case req := <-b.unreg:
			if _, ok := b.listeners[req.channel]; ok {
				delete(b.listeners, req.channel)
				close(req.channel)
			}
			close(req.done)

		case <-idle:
			return
		}
	}
}

func (b *broadcaster[In, Out]) broadcast(in In) {
	if len(b.listeners) == 0 {
		return
	}

	out, ok := b.convert(in)
	if !ok {
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
	if b.timeout == 0 {
		close(timeout)
	} else {
		time.AfterFunc(b.timeout, func() { close(timeout) })
	}

	unreg := make(chan chan<- Out, len(b.listeners))
	for listener, opts := range b.listeners {
		listener := listener
		onTimeout := opts.onTimeout
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
				if onTimeout != nil {
					go onTimeout()
				}
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

func (b *broadcaster[In, Out]) close() {
	close(b.closed)
	for listener := range b.listeners {
		close(listener)
	}
	clear(b.listeners)
}

func noConversion[T any](t T) (T, bool) {
	return t, true
}
