package main

import (
	"crypto/subtle"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofurry/monitor"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("Try /ok, /slow, /client-error, /server-error, or /monitor\n"))
	})
	mux.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(250 * time.Millisecond)
		_, _ = w.Write([]byte("slow response\n"))
	})
	mux.HandleFunc("/client-error", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "client error", http.StatusBadRequest)
	})
	mux.HandleFunc("/server-error", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	})

	token := os.Getenv("MONITOR_TOKEN")
	var authorize func(*http.Request) bool
	if token != "" {
		expected := []byte("Bearer " + token)
		authorize = func(r *http.Request) bool {
			return subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), expected) == 1
		}
		log.Println("monitor authorization enabled through MONITOR_TOKEN")
	}

	handler := monitor.New(mux, monitor.Config{
		Path:                "/monitor",
		Title:               "Example Monitor",
		Description:         "Live process, runtime, system, container, network, and HTTP metrics.",
		Footer:              "Powered by github.com/gofurry/monitor - MIT License.",
		ServiceName:         "monitor-example",
		Version:             "v1.2.0-demo",
		Environment:         "development",
		DefaultLanguage:     "en",
		DefaultTheme:        "dark",
		DefaultSampleWindow: 60,
		Refresh:             500 * time.Millisecond,
		Authorize:           authorize,
	})

	log.Println("listening on http://localhost:18848")
	log.Fatal(http.ListenAndServe(":18848", handler))
}
