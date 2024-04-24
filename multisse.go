package broadcaster

import (
	"net/http"
)

type MultiEventSource[K comparable] interface {
	GetKey(*http.Request) (K, error)
	GetEventSource(K) (EventSource, CancelFunc, error)
}

func NewMultiSSEBroadcaster[K comparable](src MultiEventSource[K], opts ...BroadcasterOption) http.Handler {
	source := func(key K) (<-chan Event, CancelFunc, error) {
		src, cancel, err := src.GetEventSource(key)
		if err != nil {
			return nil, nil, err
		}
		return src.run(), cancel, nil
	}
	b := NewMultiConverterBroadcaster(source, marshalEvent, opts...)
	listen := func(r *http.Request) (<-chan string, error) {
		key, err := src.GetKey(r)
		if err != nil {
			return nil, err
		}
		l, _, err := b.Listen(key, WithContext(r.Context()))
		return l, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveSSE(listen, w, r)
	})
}
