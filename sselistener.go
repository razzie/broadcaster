package broadcaster

import (
	"bufio"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

func ListenSSE(ctx context.Context, url string, opts ...SSEListenerOption) (<-chan Event, error) {
	slo := defaultSSEListenerOptions
	for _, opt := range opts {
		opt(&slo)
	}

	r, err := doRequest(ctx, url, &slo)
	if err != nil {
		return nil, err
	}

	events := make(chan Event, slo.bufSize)
	go func() {
		defer r.Close()
		defer close(events)

		var builder sseEventBuilder

		scanner := bufio.NewScanner(r)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			line := scanner.Text()
			if e := builder.addLine(line); e != nil {
				events <- e
			}
		}

		if err := scanner.Err(); err != nil {
			events <- errorEvent(err.Error())
		}
	}()
	return events, nil
}

type sseEventBuilder struct {
	name string
	data string
}

func (b *sseEventBuilder) addLine(line string) Event {
	if len(line) == 0 {
		if len(b.name) > 0 || len(b.data) > 0 {
			e := &textEvent{name: b.name, data: b.data}
			b.reset()
			return e
		}
		return nil
	}
	prefix, value, ok := strings.Cut(line, ": ")
	if !ok {
		return errorEvent("malformed line: " + line)
	}
	switch prefix {
	case "event":
		if len(b.data) > 0 {
			return errorEvent("event name sent after data: " + value)
		}
		b.name = value
	case "", "id", "retry":
	case "data":
		if len(b.data) > 0 {
			b.data += "\n" + value
		} else {
			b.data = value
		}
	default:
		return errorEvent("malformed line: " + line)
	}
	return nil
}

func (b *sseEventBuilder) reset() {
	b.name = ""
	b.data = ""
}

type textEvent struct {
	name string
	data string
}

func (e textEvent) Read() (name, data string) {
	return e.name, e.data
}

func errorEvent(data string) Event {
	return &textEvent{name: "error", data: data}
}

func doRequest(ctx context.Context, url string, slo *sseListenerOptions) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, slo.method, url, slo.body)
	if err != nil {
		return nil, err
	}
	for key, values := range slo.header {
		req.Header[key] = values
	}
	if len(slo.bodyContentType) > 0 {
		req.Header.Set("Content-Type", slo.bodyContentType)
	}

	resp, err := slo.client.Do(req)
	if err != nil {
		return nil, err
	}
	respCType := resp.Header.Get("Content-Type")
	if parsed, _, _ := mime.ParseMediaType(respCType); parsed != "text/event-stream" {
		resp.Body.Close()
		return nil, errors.New("bad content type: " + respCType)
	}
	return resp.Body, nil
}
