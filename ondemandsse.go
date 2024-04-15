package broadcaster

import (
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
	listen := func(r *http.Request) (<-chan string, error) {
		l, _, err := b.Listen(WithContext(r.Context()))
		return l, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveSSE(listen, w, r)
	})
}
