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

		a := 1
		for {
			a += 1
			if a > 100000000 {
				break
			}
		}

		_, _ = w.Write([]byte("hello"))
	})

	handler := monitor.New(mux, monitor.Config{
		Path:                "/monitor",
		Title:               "Example Monitor",
		Footer:              "Powered by github.com/gofurry/monitor - MIT License.",
		Description:         "Live process, runtime, system, and HTTP metrics for this Go service.",
		DefaultLanguage:     "en",
		DefaultSampleWindow: 60,
		Refresh:             2 * time.Second,
	})

	log.Println("listening on http://localhost:18848")
	log.Fatal(http.ListenAndServe(":18848", handler))
}
