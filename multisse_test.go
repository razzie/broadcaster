package broadcaster_test

import (
	"errors"
	"net/http"
	"testing"

	. "github.com/razzie/broadcaster"

	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, "data: 1\n\n", <-resp1)
	assert.Equal(t, "data: 2\n\n", <-resp2)
}

func TestMultiSSEBroadcasterBadKey(t *testing.T) {
	b := NewMultiSSEBroadcaster(new(multiEventSource))

	resp := runRequest(b, "/")
	assert.Equal(t, noKeyErrText+"\n", <-resp)
}

func TestMultiSSEBroadcasterSourceError(t *testing.T) {
	b := NewMultiSSEBroadcaster(new(multiEventSource), WithBlocking(true))
	mux := http.NewServeMux()
	mux.Handle("GET /sse/{key}", b)

	resp := runRequest(mux, "/sse/"+invalidKey)
	assert.Equal(t, invalidKeyErrText+"\n", <-resp)
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
