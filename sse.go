package broadcaster

import (
	"encoding/json"
	"net/http"
)

func NewSSEBroadcaster[T any](input <-chan T, event string, opts ...BroadcasterOption) http.Handler {
	b := &Broadcaster[T, []byte]{
		broadcasterOptions: defaultBroadcasterOptions,
		input:              input,
		transform:          func(t T) ([]byte, error) { return json.Marshal(t) },
		listeners:          make(map[chan<- []byte]bool),
		reg:                make(chan chan<- []byte),
		unreg:              make(chan chan<- []byte),
		closed:             make(chan struct{}),
	}
	for _, opt := range opts {
		opt(&b.broadcasterOptions)
	}
	go b.run()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l, _, err := b.Listen(WithContext(r.Context()))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Cache-Control", "no-store")
		w.Header().Add("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for m := range l {
			if len(event) > 0 {
				w.Write([]byte("event: " + event + "\n"))
			}
			w.Write([]byte("data: "))
			w.Write(m)
			w.Write([]byte("\n\n"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	})
}
