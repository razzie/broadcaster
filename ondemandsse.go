package broadcaster

import (
	"context"
	"net/http"
)

type OndemandEventSource func() (EventSource, error)

func NewOndemandSSEBroadcaster(src OndemandEventSource, opts ...BroadcasterOption) http.Handler {
	source := func() (<-chan event, error) {
		src, err := src()
		if err != nil {
			return nil, err
		}
		return src.run(), nil
	}
	b := NewOndemandConverterBroadcaster(source, marshalEvent, opts...)
	listen := func(ctx context.Context) (<-chan string, error) {
		l, _, err := b.Listen(WithContext(ctx))
		return l, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveSSE(listen, w, r)
	})
}
