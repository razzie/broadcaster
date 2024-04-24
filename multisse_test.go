package broadcaster_test

import (
	"errors"
	"net/http"
	"testing"

	. "github.com/razzie/broadcaster"
)

const (
	invalidKey        = "invalid"
	invalidKeyErrText = "invalid key"
	noKeyErrText      = "bad key"
)

func TestMultiSSEBroadcaster(t *testing.T) {
	b := NewMultiSSEBroadcaster(new(multiEventSource), WithBlocking(true))
	mux := http.NewServeMux()
	mux.Handle("GET /sse/{key}", b)

	resp1 := runRequest(mux, "/sse/1")
	resp2 := runRequest(mux, "/sse/2")

	if expected, got := "data: 1\n\n", <-resp1; expected != got {
		t.Errorf("expected <-resp1 == %q, got %q", expected, got)
	}
	if expected, got := "data: 2\n\n", <-resp2; expected != got {
		t.Errorf("expected <-resp2 == %q, got %q", expected, got)
	}
}

func TestMultiSSEBroadcasterBadKey(t *testing.T) {
	b := NewMultiSSEBroadcaster(new(multiEventSource))

	resp := runRequest(b, "/")

	if expected, got := noKeyErrText+"\n", <-resp; expected != got {
		t.Errorf("expected <-resp == %q, got %q", expected, got)
	}
}

func TestMultiSSEBroadcasterSourceError(t *testing.T) {
	b := NewMultiSSEBroadcaster(new(multiEventSource), WithBlocking(true))
	mux := http.NewServeMux()
	mux.Handle("GET /sse/{key}", b)

	resp := runRequest(mux, "/sse/"+invalidKey)

	if expected, got := invalidKeyErrText+"\n", <-resp; expected != got {
		t.Errorf("expected <-resp == %q, got %q", expected, got)
	}
}

type multiEventSource struct{}

func (src multiEventSource) GetKey(r *http.Request) (string, error) {
	key := r.PathValue("key")
	if len(key) == 0 {
		return "", errors.New(noKeyErrText)
	}
	return key, nil
}

func (src multiEventSource) GetEventSource(key string) (EventSource, CancelFunc, error) {
	if key == "invalid" {
		return nil, nil, errors.New(invalidKeyErrText)
	}
	ch := make(chan string, 1)
	ch <- key
	close(ch)
	return NewTextEventSource(ch, ""), func() {}, nil
}
