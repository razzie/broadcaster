package main

import (
	"html/template"
	"net/http"

	"github.com/razzie/broadcaster"
)

var t = template.Must(template.ParseGlob("./template/*.html"))

type Message struct {
	Name string
	Text string
}

func main() {
	messages := make(chan Message, 64)
	b := broadcaster.NewBroadcaster(messages)

	var r http.ServeMux
	r.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		t.ExecuteTemplate(w, "index", nil)
	})
	r.HandleFunc("GET /chat", func(w http.ResponseWriter, r *http.Request) {
		view := map[string]string{
			"name": r.URL.Query().Get("name"),
		}
		t.ExecuteTemplate(w, "chat", view)
	})
	r.HandleFunc("POST /chat", func(w http.ResponseWriter, r *http.Request) {
		name := r.FormValue("name")
		message := r.FormValue("message")
		if len(message) == 0 {
			return
		}
		messages <- Message{Name: name, Text: message}
	})
	r.HandleFunc("GET /ssechat", func(w http.ResponseWriter, r *http.Request) {
		l, _, err := b.Listen(broadcaster.WithContext(r.Context()))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Add("Cache-Control", "no-store")
		w.Header().Add("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for m := range l {
			w.Write([]byte("data: "))
			t.ExecuteTemplate(w, "message", m)
			w.Write([]byte("\n\n"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	})

	http.ListenAndServe(":8080", &r)
}
