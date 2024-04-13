package broadcaster

import (
	"context"
	"net/http"
	"sync"
)

type OndemandEventSource func() (EventSource, error)

func NewOndemandSSEBroadcaster(src OndemandEventSource, opts ...BroadcasterOption) http.Handler {
	var mu sync.Mutex
	var b Broadcaster[string]

	listen := func(ctx context.Context) (<-chan string, error) {
		mu.Lock()
		defer mu.Unlock()
		if b == nil || b.IsClosed() {
			evSrc, err := src()
			if err != nil {
				return nil, err
			}
			b = NewConverterBroadcaster(evSrc.run(), marshalEvent, opts...)
		}
		l, _, err := b.Listen(WithContext(ctx))
		return l, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveSSE(listen, w, r)
	})
}
