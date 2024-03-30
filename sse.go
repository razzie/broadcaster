package broadcaster

import (
	"bytes"
	"encoding/json"
	"html/template"
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

func NewTextEventSource(input <-chan string, eventName string) EventSource {
	return NewEventSource(input, eventName, marshalText)
}

func NewTemplateEventSource[T any](input <-chan T, eventName string, t *template.Template, templateName string) EventSource {
	return NewEventSource(input, eventName, marshalTemplate(t, templateName))
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

func (src EventSource) run() <-chan event {
	events := make(chan event)
	var wg sync.WaitGroup
	wg.Add(1)
	go src(events, &wg)
	go func() {
		wg.Wait()
		close(events)
	}()
	return events
}

func NewSSEBroadcaster(src EventSource, opts ...BroadcasterOption) http.Handler {
	b := NewConverterBroadcaster(src.run(), marshalEvent, opts...)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Server does not support Flusher!", http.StatusInternalServerError)
			return
		}

		events, _, err := b.Listen(WithContext(r.Context()))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Cache-Control", "no-store")
		w.Header().Add("Content-Type", "text/event-stream")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		for e := range events {
			w.Write([]byte(e))
			flusher.Flush()
		}
	})
}

func marshalEvent(e event) (string, error) {
	bytes, err := e.marshaler(e.value)
	if err != nil {
		return "event: error\ndata: failed to serialize event: " + err.Error() + "\n\n", nil
	}
	str := strings.ReplaceAll(string(bytes), "\n", "\ndata: ")
	if len(e.name) > 0 {
		return "event: " + e.name + "\ndata: " + str + "\n\n", nil
	}
	return "data: " + str + "\n\n", nil
}

func marshalText(text any) ([]byte, error) {
	return []byte(text.(string)), nil
}

func marshalTemplate(t *template.Template, name string) func(any) ([]byte, error) {
	if len(name) == 0 {
		return func(val any) ([]byte, error) {
			var buf bytes.Buffer
			if err := t.Execute(&buf, val); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
	}
	return func(val any) ([]byte, error) {
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, name, val); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
}
