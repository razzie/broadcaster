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
	messageEvents := broadcaster.NewTemplateEventSource(messages, "message_event", t, "message")

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
	r.Handle("GET /ssechat", broadcaster.NewSSEBroadcaster(messageEvents))

	http.ListenAndServe(":8080", &r)
}
