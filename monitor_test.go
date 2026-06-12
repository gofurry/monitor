package monitor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewMonitorServesJSONSnapshot(t *testing.T) {
	m := NewMonitor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}), Config{Refresh: time.Hour})
	defer m.Stop()

	req := httptest.NewRequest(http.MethodGet, "/monitor", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("content type = %q, want application/json", got)
	}

	var stats Stats
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if stats.Runtime.Goroutines == 0 {
		t.Fatal("expected runtime goroutine count")
	}
}

func TestMonitorDoesNotCountMonitorRequests(t *testing.T) {
	m := NewMonitor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), Config{Refresh: time.Hour})
	defer m.Stop()

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/monitor", nil)
		req.Header.Set("Accept", "application/json")
		m.ServeHTTP(httptest.NewRecorder(), req)
	}
	m.collectOnce()
	if got := m.Current().HTTP.TotalRequests; got != 0 {
		t.Fatalf("monitor requests = %d, want 0", got)
	}

	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/hello", nil))
	m.collectOnce()
	if got := m.Current().HTTP.TotalRequests; got != 1 {
		t.Fatalf("business requests = %d, want 1", got)
	}
}

func TestMonitorServesHTMLByDefault(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{
		Title:   "My App",
		Refresh: time.Hour,
	})
	defer m.Stop()

	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/monitor", nil))

	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("content type = %q, want text/html", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, "My App") {
		t.Fatalf("HTML body does not contain title: %q", body)
	}
}

func TestAPIOnlyServesJSONWithoutAcceptHeader(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{
		APIOnly: true,
		Refresh: time.Hour,
	})
	defer m.Stop()

	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/monitor", nil))

	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("content type = %q, want application/json", got)
	}
}

func TestConfigDefaultsAndPathNormalization(t *testing.T) {
	cfg := applyConfig([]Config{{Path: "status"}})

	if cfg.Path != "/status" {
		t.Fatalf("path = %q, want /status", cfg.Path)
	}
	if cfg.Title != defaultTitle {
		t.Fatalf("title = %q, want %q", cfg.Title, defaultTitle)
	}
	if cfg.Refresh != defaultRefresh {
		t.Fatalf("refresh = %s, want %s", cfg.Refresh, defaultRefresh)
	}
}
