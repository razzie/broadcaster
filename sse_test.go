package broadcaster_test

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/razzie/broadcaster"

	"github.com/stretchr/testify/assert"
)

func TestSSEBroadcast(t *testing.T) {
	ch := make(chan int)
	b := NewSSEBroadcaster(NewJsonEventSource(ch, ""))

	resp1 := runRequest(b, "/")
	resp2 := runRequest(b, "/")

	time.Sleep(time.Millisecond)

	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)

	expected := "data: 1\n\ndata: 2\n\ndata: 3\n\n"
	assert.Equal(t, expected, <-resp1)
	assert.Equal(t, expected, <-resp2)
}

func TestSSEBroadcastWithEventName(t *testing.T) {
	ch := make(chan int)
	b := NewSSEBroadcaster(NewJsonEventSource(ch, "a"))

	resp1 := runRequest(b, "/")
	resp2 := runRequest(b, "/")

	time.Sleep(time.Millisecond)

	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)

	expected := "event: a\ndata: 1\n\nevent: a\ndata: 2\n\nevent: a\ndata: 3\n\n"
	assert.Equal(t, expected, <-resp1)
	assert.Equal(t, expected, <-resp2)
}

func TestMultilineEvent(t *testing.T) {
	ch := make(chan string)
	b := NewSSEBroadcaster(NewTextEventSource(ch, ""))

	resp := runRequest(b, "/")

	time.Sleep(time.Millisecond)

	ch <- "ab\ncd"
	close(ch)

	expected := "data: ab\ndata: cd\n\n"
	assert.Equal(t, expected, <-resp)
}

func TestTextEventSource(t *testing.T) {
	ch := make(chan string)
	b := NewSSEBroadcaster(NewTextEventSource(ch, ""))

	resp := runRequest(b, "/")

	time.Sleep(time.Millisecond)

	ch <- "a b c d"
	close(ch)

	expected := "data: a b c d\n\n"
	assert.Equal(t, expected, <-resp)
}

func TestTemplateEventSource(t *testing.T) {
	ch := make(chan string)
	tmpl := template.Must(template.New("").Parse("<span>Hello {{.}}</span>"))
	b := NewSSEBroadcaster(NewTemplateEventSource(ch, "", tmpl, ""))

	resp := runRequest(b, "/")

	time.Sleep(time.Millisecond)

	ch <- "World"
	close(ch)

	expected := "data: <span>Hello World</span>\n\n"
	assert.Equal(t, expected, <-resp)
}

func TestBundleEventSources(t *testing.T) {
	ch1 := make(chan int)
	ch2 := make(chan int)
	event1 := NewEventSource(ch1, "event1", nil)
	event2 := NewEventSource(ch2, "event2", nil)
	b := NewSSEBroadcaster(BundleEventSources(event1, event2))

	resp := runRequest(b, "/")

	time.Sleep(time.Millisecond)
	ch1 <- 1
	time.Sleep(time.Millisecond)
	ch2 <- 2
	close(ch1)
	close(ch2)

	expected := "event: event1\ndata: 1\n\nevent: event2\ndata: 2\n\n"
	assert.Equal(t, expected, <-resp)
}

func runRequest(h http.Handler, path string) <-chan string {
	resp := make(chan string)
	req := httptest.NewRequest("GET", path, nil)
	rec := httptest.NewRecorder()
	go func() {
		h.ServeHTTP(rec, req)
		res := rec.Result()
		defer res.Body.Close()
		defer close(resp)
		all, _ := io.ReadAll(res.Body)
		resp <- string(all)
	}()
	return resp
}
