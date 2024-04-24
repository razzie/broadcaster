package broadcaster

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"sync"
)

type Event interface {
	Read() (name, data string)
}

type event struct {
	name      string
	data      any
	marshaler Marshaler
}

func (e event) Read() (name, data string) {
	bytes, err := e.marshaler(e.data)
	if err != nil {
		return "error", "failed to serialize event: " + err.Error()
	}
	return e.name, string(bytes)
}

type Marshaler func(any) ([]byte, error)

type EventSource func(chan<- Event, *sync.WaitGroup)

func NewEventSource[T any](input <-chan T, eventName string, marshaler Marshaler) EventSource {
	if marshaler == nil {
		marshaler = json.Marshal
	}
	return func(events chan<- Event, wg *sync.WaitGroup) {
		defer wg.Done()
		for in := range input {
			events <- event{
				name:      eventName,
				data:      in,
				marshaler: marshaler,
			}
		}
	}
}

func NewJsonEventSource[T any](input <-chan T, eventName string) EventSource {
	return NewEventSource(input, eventName, json.Marshal)
}

func NewTextEventSource(input <-chan string, eventName string) EventSource {
	return NewEventSource(input, eventName, marshalText)
}

func NewTemplateEventSource[T any](input <-chan T, eventName string, t *template.Template, templateName string) EventSource {
	return NewEventSource(input, eventName, marshalTemplate(t, templateName))
}

func BundleEventSources(srcs ...EventSource) EventSource {
	return func(events chan<- Event, wg *sync.WaitGroup) {
		defer wg.Done()
		wg.Add(len(srcs))
		for _, src := range srcs {
			go src(events, wg)
		}
	}
}

func (src EventSource) run() <-chan Event {
	events := make(chan Event)
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
	listen := func(r *http.Request) (<-chan string, error) {
		l, _, err := b.Listen(WithContext(r.Context()))
		return l, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveSSE(listen, w, r)
	})
}

func serveSSE(listen func(*http.Request) (<-chan string, error), w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Server does not support Flusher!", http.StatusInternalServerError)
		return
	}

	events, err := listen(r)
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
}

func marshalEvent(e Event) (string, bool) {
	name, data := e.Read()
	data = strings.ReplaceAll(data, "\n", "\ndata: ")
	if len(name) > 0 {
		return "event: " + name + "\ndata: " + data + "\n\n", true
	}
	return "data: " + data + "\n\n", true
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
