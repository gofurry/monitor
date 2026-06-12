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
	assertMonitorHeaders(t, rec)

	var stats Stats
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if stats.Runtime.Goroutines == 0 {
		t.Fatal("expected runtime goroutine count")
	}
}

func TestMonitorDoesNotCountMonitorRequests(t *testing.T) {
	var ignoredCalls int
	m := NewMonitor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), Config{
		Refresh: time.Hour,
		IgnoreRequest: func(r *http.Request) bool {
			ignoredCalls++
			return false
		},
	})
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
	if ignoredCalls != 0 {
		t.Fatalf("ignore callback calls after monitor requests = %d, want 0", ignoredCalls)
	}

	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/hello", nil))
	m.collectOnce()
	if got := m.Current().HTTP.TotalRequests; got != 1 {
		t.Fatalf("business requests = %d, want 1", got)
	}
	if ignoredCalls != 1 {
		t.Fatalf("ignore callback calls after business request = %d, want 1", ignoredCalls)
	}
}

func TestMonitorIgnoreRequestExcludesConfiguredRequests(t *testing.T) {
	var served int
	m := NewMonitor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served++
		w.WriteHeader(http.StatusNoContent)
	}), Config{
		Refresh: time.Hour,
		IgnoreRequest: func(r *http.Request) bool {
			return r.URL.Path == "/healthz" || r.URL.Path == "/readyz"
		},
	})
	defer m.Stop()

	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))
	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/readyz", nil))
	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/users", nil))
	m.collectOnce()

	if served != 3 {
		t.Fatalf("served requests = %d, want 3", served)
	}
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
	assertMonitorHeaders(t, rec)
}

func TestMonitorHTMLIncludesEnhancedUI(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{Refresh: time.Hour})
	defer m.Stop()

	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/monitor", nil))

	body := rec.Body.String()
	for _, want := range []string{
		`data-lang="zh-CN"`,
		`id="theme-toggle"`,
		`data-status="live"`,
		`id="cpu-chart"`,
		`id="memory-chart"`,
		`id="goroutine-chart"`,
		`id="request-chart"`,
		`storageSet("monitor.theme"`,
		`storageSet("monitor.lang"`,
		`getContext("2d")`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("HTML body does not contain %q", want)
		}
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

func TestMonitorAllowsOnlyGetAndHead(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{Refresh: time.Hour})
	defer m.Stop()

	head := httptest.NewRecorder()
	m.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/monitor", nil))
	if head.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want %d", head.Code, http.StatusOK)
	}

	post := httptest.NewRecorder()
	m.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/monitor", nil))
	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want %d", post.Code, http.StatusMethodNotAllowed)
	}
	if got := post.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("Allow = %q, want %q", got, "GET, HEAD")
	}
	assertMonitorHeaders(t, post)
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

func assertMonitorHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want %q", got, "no-store")
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}
