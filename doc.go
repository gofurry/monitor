// Package monitor provides a tiny net/http middleware for exposing a real-time
// Go service status page and JSON metrics snapshot.
//
// The package is intentionally small: it wraps an existing http.Handler,
// exposes one monitor path, and serves metrics from a race-safe background
// snapshot. Requests to the monitor path are excluded from the HTTP request
// count so page refreshes and JSON polling do not inflate business traffic.
//
// Basic usage:
//
//	mux := http.NewServeMux()
//	handler := monitor.New(mux)
//	http.ListenAndServe(":8080", handler)
//
// Open /monitor in a browser for the HTML page, or request the same path with
// Accept: application/json for the raw snapshot.
package monitor
