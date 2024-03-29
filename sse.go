package broadcaster

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type event struct {
	name      string
	value     any
	marshaler Marshaler
}

type Marshaler func(any) ([]byte, error)

type EventSource func(events chan<- event, wg *sync.WaitGroup)

func NewEventSource[T any](input <-chan T, eventName string, marshaler Marshaler) EventSource {
	if marshaler == nil {
		marshaler = json.Marshal
	}
	return func(events chan<- event, wg *sync.WaitGroup) {
		defer wg.Done()
		for in := range input {
			events <- event{
				name:      eventName,
				value:     in,
				marshaler: marshaler,
			}
		}
	}
}

func BundleEventSources(srcs ...EventSource) EventSource {
	return func(events chan<- event, wg *sync.WaitGroup) {
		defer wg.Done()
		wg.Add(len(srcs))
		for _, src := range srcs {
			go src(events, wg)
		}
	}
}

func NewSSEBroadcaster(src EventSource, opts ...BroadcasterOption) http.Handler {
	events := make(chan event)
	b := NewConverterBroadcaster(events, marshalEvent, opts...)

	var wg sync.WaitGroup
	wg.Add(1)
	go src(events, &wg)
	go func() {
		wg.Wait()
		close(events)
	}()

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
			w.Write([]byte(m))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	})
}

func marshalEvent(e event) (string, error) {
	bytes, err := e.marshaler(e.value)
	if err != nil {
		return fmt.Sprintf("event: error\ndata: failed to serialize event: %v\n\n", err), nil
	}
	str := strings.ReplaceAll(string(bytes), "\n", "\ndata: ")
	if len(e.name) > 0 {
		return fmt.Sprintf("event: %s\ndata: %s\n\n", e.name, str), nil
	}
	return fmt.Sprintf("data: %s\n\n", str), nil
}
