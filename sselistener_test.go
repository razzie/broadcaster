package broadcaster_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/razzie/broadcaster"
)

func TestListenSSE(t *testing.T) {
	const resp = "event: event1\ndata: 1\n\ndata: 2\n\n"
	const respEventNum = 2
	server := httptest.NewServer(dummySSEHandler(resp))

	events, err := ListenSSE(context.Background(), server.URL,
		WithEventsBufferSize(respEventNum))
	if err != nil {
		t.Fatalf("unexpected error listening to server sent events: %v", err)
	}

	event1, data1 := (<-events).Read()
	if event1 != "event1" {
		t.Errorf("expected first event name to be \"event1\", got %q", event1)
	}
	if data1 != "1" {
		t.Errorf("expected first event data to be \"1\", got %q", data1)
	}

	event2, data2 := (<-events).Read()
	if event2 != "" {
		t.Errorf("expected second event name to be empty, got %q", event1)
	}
	if data2 != "2" {
		t.Errorf("expected second event data to be \"2\", got %q", data1)
	}
}

func dummySSEHandler(response string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})
}
