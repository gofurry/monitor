package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gofurry/monitor"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	})

	handler := monitor.New(mux, monitor.Config{
		Path:    "/monitor",
		Title:   "Example Monitor",
		Refresh: 2 * time.Second,
	})

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
