package broadcaster

import (
	"net/http"
)

type OndemandEventSource func() (<-chan Event, error)

func NewOndemandSSEBroadcaster(src OndemandEventSource, opts ...BroadcasterOption) http.Handler {
	source := func() (<-chan Event, error) {
		events, err := src()
		if err != nil {
			return nil, err
		}
		return events, nil
	}
	b := NewOndemandConverterBroadcaster(source, marshalEvent, opts...)
	listen := func(r *http.Request) (<-chan string, error) {
		l, _, err := b.Listen(WithContext(r.Context()))
		return l, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveSSE(listen, w, r)
	})
}
