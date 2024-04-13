package broadcaster

import (
	"sync"
)

type Source[T any] func() (<-chan T, error)

type ondemandBroadcaster[In, Out any] struct {
	mu      sync.Mutex
	b       Broadcaster[Out]
	src     Source[In]
	convert Converter[In, Out]
	opts    []BroadcasterOption
}

func NewOndemandBroadcaster[T any](src Source[T], opts ...BroadcasterOption) Broadcaster[T] {
	return &ondemandBroadcaster[T, T]{
		src:     src,
		convert: noConversion[T],
		opts:    opts,
	}
}

func NewOndemandConverterBroadcaster[In, Out any](src Source[In], convert Converter[In, Out], opts ...BroadcasterOption) Broadcaster[Out] {
	return &ondemandBroadcaster[In, Out]{
		src:     src,
		convert: convert,
		opts:    opts,
	}
}

func (ob *ondemandBroadcaster[In, Out]) Listen(opts ...ListenerOption) (ch <-chan Out, closer func(), err error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	if ob.b == nil || ob.b.IsClosed() {
		in, err := ob.src()
		if err != nil {
			return nil, nil, err
		}
		ob.b = NewConverterBroadcaster(in, ob.convert, ob.opts...)
	}
	return ob.b.Listen(opts...)
}

func (ob *ondemandBroadcaster[In, Out]) IsClosed() bool {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	if ob.b != nil {
		return ob.b.IsClosed()
	}
	return true
}

func (ob *ondemandBroadcaster[In, Out]) Done() <-chan struct{} {
	return nil
}
