package broadcaster_test

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	. "github.com/razzie/broadcaster"

	"github.com/stretchr/testify/assert"
)

func TestSSEBroadcast(t *testing.T) {
	ch := make(chan int)
	b := NewSSEBroadcaster(NewEventSource(ch, "", nil))

	var res1, res2 []byte
	var wg sync.WaitGroup
	wg.Add(2)
	go runRequest(b, &res1, &wg)
	go runRequest(b, &res2, &wg)

	time.Sleep(time.Millisecond)

	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)
	wg.Wait()

	expected := "data: 1\n\ndata: 2\n\ndata: 3\n\n"
	assert.Equal(t, expected, string(res1))
	assert.Equal(t, expected, string(res2))
}

func TestSSEBroadcastWithEventName(t *testing.T) {
	ch := make(chan int)
	b := NewSSEBroadcaster(NewEventSource(ch, "a", nil))

	var res1, res2 []byte
	var wg sync.WaitGroup
	wg.Add(2)
	go runRequest(b, &res1, &wg)
	go runRequest(b, &res2, &wg)

	time.Sleep(time.Millisecond)

	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)
	wg.Wait()

	expected := "event: a\ndata: 1\n\nevent: a\ndata: 2\n\nevent: a\ndata: 3\n\n"
	assert.Equal(t, expected, string(res1))
	assert.Equal(t, expected, string(res2))
}

func TestMultilineEvent(t *testing.T) {
	ch := make(chan string)
	b := NewSSEBroadcaster(NewTextEventSource(ch, ""))

	var res []byte
	var wg sync.WaitGroup
	wg.Add(1)
	go runRequest(b, &res, &wg)

	time.Sleep(time.Millisecond)

	ch <- "ab\ncd"
	close(ch)
	wg.Wait()

	expected := "data: ab\ndata: cd\n\n"
	assert.Equal(t, expected, string(res))
}

func TestTextEventSource(t *testing.T) {
	ch := make(chan string)
	b := NewSSEBroadcaster(NewTextEventSource(ch, ""))

	var res []byte
	var wg sync.WaitGroup
	wg.Add(1)
	go runRequest(b, &res, &wg)

	time.Sleep(time.Millisecond)

	ch <- "a b c d"
	close(ch)
	wg.Wait()

	expected := "data: a b c d\n\n"
	assert.Equal(t, expected, string(res))
}

func TestTemplateEventSource(t *testing.T) {
	ch := make(chan string)
	tmpl := template.Must(template.New("").Parse("<span>Hello {{.}}</span>"))
	b := NewSSEBroadcaster(NewTemplateEventSource(ch, "", tmpl, ""))

	var res []byte
	var wg sync.WaitGroup
	wg.Add(1)
	go runRequest(b, &res, &wg)

	time.Sleep(time.Millisecond)

	ch <- "World"
	close(ch)
	wg.Wait()

	expected := "data: <span>Hello World</span>\n\n"
	assert.Equal(t, expected, string(res))
}

func TestBundleEventSources(t *testing.T) {
	ch1 := make(chan int)
	ch2 := make(chan int)
	event1 := NewEventSource(ch1, "event1", nil)
	event2 := NewEventSource(ch2, "event2", nil)
	b := NewSSEBroadcaster(BundleEventSources(event1, event2))

	var res []byte
	var wg sync.WaitGroup
	wg.Add(1)
	go runRequest(b, &res, &wg)

	time.Sleep(time.Millisecond)
	ch1 <- 1
	time.Sleep(time.Millisecond)
	ch2 <- 2
	close(ch1)
	close(ch2)
	wg.Wait()

	expected := "event: event1\ndata: 1\n\nevent: event2\ndata: 2\n\n"
	assert.Equal(t, expected, string(res))
}

func runRequest(h http.Handler, p *[]byte, wg *sync.WaitGroup) {
	defer wg.Done()

	req := httptest.NewRequest("GET", "/events", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	*p, _ = io.ReadAll(rec.Body)
}
