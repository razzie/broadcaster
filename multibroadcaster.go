package broadcaster

import (
	"sync"
)

type MultiSource[K comparable, T any] func(K) (<-chan T, CancelFunc, error)

type MultiBroadcaster[K comparable, T any] interface {
	Listen(key K, opts ...ListenerOption) (<-chan T, CancelFunc, error)
}

type multiBroadcaster[K comparable, In, Out any] struct {
	mu   sync.Mutex
	bcs  map[K]Broadcaster[Out]
	src  MultiSource[K, In]
	conv Converter[In, Out]
	opts []BroadcasterOption
}

func NewMultiBroadcaster[K comparable, T any](src MultiSource[K, T], opts ...BroadcasterOption) MultiBroadcaster[K, T] {
	return &multiBroadcaster[K, T, T]{
		bcs:  make(map[K]Broadcaster[T]),
		src:  src,
		conv: noConversion[T],
		opts: opts,
	}
}

func NewMultiConverterBroadcaster[K comparable, In, Out any](src MultiSource[K, In], conv Converter[In, Out], opts ...BroadcasterOption) MultiBroadcaster[K, Out] {
	return &multiBroadcaster[K, In, Out]{
		bcs:  make(map[K]Broadcaster[Out]),
		src:  src,
		conv: conv,
		opts: opts,
	}
}

func (mb *multiBroadcaster[K, In, Out]) Listen(key K, opts ...ListenerOption) (<-chan Out, CancelFunc, error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if bc := mb.bcs[key]; bc != nil && !bc.IsClosed() {
		return bc.Listen(opts...)
	}

	input, cancel, err := mb.src(key)
	if err != nil {
		return nil, nil, err
	}

	bc := NewConverterBroadcaster(input, mb.conv, mb.opts...)
	mb.bcs[key] = bc

	go func() {
		<-bc.Done()
		cancel()

		mb.mu.Lock()
		defer mb.mu.Unlock()
		bc = mb.bcs[key]
		if bc.IsClosed() {
			delete(mb.bcs, key)
		}
	}()

	return bc.Listen(opts...)
}
