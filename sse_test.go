package broadcaster

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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

func runRequest(h http.Handler, p *[]byte, wg *sync.WaitGroup) {
	defer wg.Done()

	req := httptest.NewRequest("GET", "/events", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	*p, _ = io.ReadAll(rec.Body)
}
