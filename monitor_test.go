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
	if stats.PID.PID == 0 {
		t.Fatal("expected process id")
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
	if got := m.Current().HTTP.StatusCodes.Status2xx; got != 1 {
		t.Fatalf("2xx responses = %d, want 1", got)
	}
	if got := m.Current().HTTP.StatusCodes.Status4xx; got != 0 {
		t.Fatalf("4xx responses = %d, want 0", got)
	}
	if got := m.Current().HTTP.InFlightRequests; got != 0 {
		t.Fatalf("in-flight requests = %d, want 0", got)
	}
}

func TestMonitorCollectsHTTPStatusLatencyAndInFlight(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	m := NewMonitor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/created":
			w.WriteHeader(http.StatusCreated)
		case "/redirect":
			w.WriteHeader(http.StatusFound)
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
		case "/slow":
			close(entered)
			<-release
			w.WriteHeader(http.StatusAccepted)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}), Config{Refresh: time.Hour})
	defer m.Stop()

	for _, path := range []string{"/created", "/redirect", "/missing", "/error"} {
		m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}

	done := make(chan struct{})
	go func() {
		m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/slow", nil))
		close(done)
	}()
	<-entered
	m.collectOnce()

	if got := m.Current().HTTP.InFlightRequests; got != 1 {
		t.Fatalf("in-flight requests = %d, want 1", got)
	}

	time.Sleep(2 * time.Millisecond)
	close(release)
	<-done
	m.collectOnce()

	httpStats := m.Current().HTTP
	if httpStats.TotalRequests != 5 {
		t.Fatalf("total requests = %d, want 5", httpStats.TotalRequests)
	}
	if got := httpStats.InFlightRequests; got != 0 {
		t.Fatalf("in-flight requests after release = %d, want 0", got)
	}
	if got := httpStats.StatusCodes.Status2xx; got != 2 {
		t.Fatalf("2xx responses = %d, want 2", got)
	}
	if got := httpStats.StatusCodes.Status3xx; got != 1 {
		t.Fatalf("3xx responses = %d, want 1", got)
	}
	if got := httpStats.StatusCodes.Status4xx; got != 1 {
		t.Fatalf("4xx responses = %d, want 1", got)
	}
	if got := httpStats.StatusCodes.Status5xx; got != 1 {
		t.Fatalf("5xx responses = %d, want 1", got)
	}
	if httpStats.Latency.LastNS == 0 {
		t.Fatalf("last latency = %d, want > 0", httpStats.Latency.LastNS)
	}
	if httpStats.Latency.RecentNS <= 0 {
		t.Fatalf("recent latency = %f, want > 0", httpStats.Latency.RecentNS)
	}
	if float64(httpStats.Latency.MaxNS) < httpStats.Latency.RecentNS {
		t.Fatalf("max latency = %d, want >= recent %f", httpStats.Latency.MaxNS, httpStats.Latency.RecentNS)
	}
}

func TestMonitorRecordsMinimumNanosecondLatency(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{Refresh: time.Hour})
	defer m.Stop()

	m.recordBusinessRequest(http.StatusNoContent, 0)
	m.collectOnce()

	latency := m.Current().HTTP.Latency
	if latency.LastNS != uint64(time.Nanosecond) {
		t.Fatalf("last latency = %d, want %d", latency.LastNS, time.Nanosecond)
	}
	if latency.RecentNS <= 0 {
		t.Fatalf("recent latency = %f, want > 0", latency.RecentNS)
	}
	if latency.MaxNS != uint64(time.Nanosecond) {
		t.Fatalf("max latency = %d, want %d", latency.MaxNS, time.Nanosecond)
	}
}

func TestMonitorCollectsGCPauseStats(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{Refresh: time.Hour})
	defer m.Stop()

	m.gcPauseTotal.Store(0)
	m.gcPauseSeen.Store(true)

	stats := m.collectRuntime()
	if stats.GoroutinePeak < stats.Goroutines {
		t.Fatalf("goroutine peak = %d, want >= current %d", stats.GoroutinePeak, stats.Goroutines)
	}
	if stats.HeapObjects == 0 {
		t.Fatal("expected heap object count")
	}
	if stats.NextGCBytes == 0 {
		t.Fatal("expected next GC target")
	}
	if stats.Mallocs == 0 {
		t.Fatal("expected malloc count")
	}
	if stats.GCPauseRecentNS != stats.GCPauseTotalNS {
		t.Fatalf("gc pause recent = %d, want %d", stats.GCPauseRecentNS, stats.GCPauseTotalNS)
	}
	if stats.NumGC == 0 && stats.GCPauseLastNS != 0 {
		t.Fatalf("gc pause last = %d, want 0 when no GC has run", stats.GCPauseLastNS)
	}
}

func TestCollectOSIncludesDiskStats(t *testing.T) {
	m := &Monitor{cfg: DefaultConfig()}

	stats := m.collectOS()
	if stats.DiskTotalBytes == 0 {
		t.Fatal("expected disk total bytes")
	}
	if stats.DiskUsedBytes == 0 {
		t.Fatal("expected disk used bytes")
	}
	if stats.DiskUsedPercent <= 0 {
		t.Fatalf("disk used percent = %f, want > 0", stats.DiskUsedPercent)
	}
	if len(stats.Disks) == 0 {
		t.Fatal("expected disk list")
	}
	if stats.Disks[0].TotalBytes != stats.DiskTotalBytes {
		t.Fatalf("first disk total = %d, want summary total %d", stats.Disks[0].TotalBytes, stats.DiskTotalBytes)
	}
}

func TestMonitorServesHTMLByDefault(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{
		Title:       "My App",
		Description: "Production edge monitor",
		Footer:      "Copyright 2026 Example",
		Refresh:     time.Hour,
	})
	defer m.Stop()

	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/monitor", nil))

	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("content type = %q, want text/html", got)
	}
	body := rec.Body.String()
	for _, want := range []string{"My App", "Production edge monitor", "Copyright 2026 Example"} {
		if !strings.Contains(body, want) {
			t.Fatalf("HTML body does not contain %q", want)
		}
	}
	assertMonitorHeaders(t, rec)
}

func TestMonitorHTMLEscapesConfiguredContent(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{
		Title:       `<script>alert("title")</script>`,
		Description: `<img src=x onerror=alert("description")>`,
		Footer:      `<b>footer</b>`,
		Refresh:     time.Hour,
	})
	defer m.Stop()

	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/monitor", nil))

	body := rec.Body.String()
	for _, unwanted := range []string{
		`<script>alert("title")</script>`,
		`<img src=x onerror=alert("description")>`,
		`<b>footer</b>`,
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("HTML body contains unescaped content %q", unwanted)
		}
	}
	for _, want := range []string{
		`&lt;script&gt;alert(&#34;title&#34;)&lt;/script&gt;`,
		`&lt;img src=x onerror=alert(&#34;description&#34;)&gt;`,
		`&lt;b&gt;footer&lt;/b&gt;`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("HTML body does not contain escaped content %q", want)
		}
	}
}

func TestMonitorHTMLIncludesEnhancedUI(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{Refresh: time.Hour})
	defer m.Stop()

	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/monitor", nil))

	body := rec.Body.String()
	for _, want := range []string{
		`id="lang-toggle"`,
		`id="theme-toggle"`,
		`class="control-dot"`,
		`class="status-state"`,
		`class="header-divider"`,
		`class="description-card"`,
		`class="metric-card"`,
		`class="metric-pager"`,
		`metric-pager__prev`,
		`metric-pager__next`,
		`const metricsPerPage = 5`,
		`function initMetricPagination`,
		`row.hidden`,
		`Page description`,
		`Powered by github.com/gofurry/monitor - MIT License.`,
		`grid-template-columns: minmax(0, 1fr) max-content`,
		`grid-template-areas:`,
		`"controls state"`,
		`"updated response"`,
		`@keyframes moveGlow`,
		`--divider-line`,
		`#f28c28`,
		`#38bdf8`,
		`width: min(1360px`,
		`min-width: 250px`,
		`width: 250px`,
		`width: 100%`,
		`class="sample-toggle"`,
		`class="sample-option"`,
		`id="pid-threads"`,
		`id="pid-id"`,
		`id="pid-fds"`,
		`id="heap-details-button"`,
		`id="rt-next-gc"`,
		`id="gc-details-button"`,
		`id="heap-modal"`,
		`id="heap-modal-list"`,
		`id="gc-modal"`,
		`id="gc-modal-list"`,
		`id="disk-details-button"`,
		`id="disk-modal"`,
		`id="disk-modal-list"`,
		`class="metric-action"`,
		`class="modal"`,
		`class="detail-list"`,
		`id="http-in-flight"`,
		`id="http-latency-recent"`,
		`id="http-latency-max"`,
		`id="http-status-button"`,
		`id="http-status-modal"`,
		`id="http-status-modal-list"`,
		`id="latency-chart"`,
		`id="in-flight-chart"`,
		`id="heap-gc-chart"`,
		`id="gc-pause-chart"`,
		`in_flight_requests`,
		`"pid"`,
		`threads`,
		`fds`,
		`goroutine_peak`,
		`heap_objects`,
		`next_gc_bytes`,
		`mallocs`,
		`frees`,
		`gc_pause_last_ns`,
		`gc_pause_recent_ns`,
		`gc_pause_total_ns`,
		`disks`,
		`total_bytes`,
		`used_bytes`,
		`free_bytes`,
		`used_percent`,
		`function renderDiskList`,
		`function updateRuntimeDetailUI`,
		`function updateHTTPStatusUI`,
		`latencyRecentNS`,
		`latencyMaxNS`,
		`nextGCMiB`,
		`gcPauseRecentNS`,
		`gcPauseLastNS`,
		`visibleSamples(history.latencyRecentNS)`,
		`visibleSamples(history.inFlight)`,
		`visibleSamples(history.nextGCMiB)`,
		`visibleSamples(history.gcPauseRecentNS)`,
		`function durationAxisNS`,
		`status_codes`,
		`recent_ns`,
		`max_ns`,
		`id="page-scroll-dock"`,
		`class="page-scroll-dock"`,
		`page-scroll-dock--visible`,
		`scrollbar-width: none`,
		`::-webkit-scrollbar`,
		`function updateScrollOrb`,
		`function scrollUpQuarter`,
		`--scroll-progress`,
		`aria-valuenow`,
		`data-samples="30"`,
		`data-samples="60" aria-pressed="true"`,
		`data-samples="90"`,
		`window.monitorConfig =`,
		`const maxPoints = 90`,
		`"defaultLanguage":"en"`,
		`"defaultSampleWindow":60`,
		`const defaultLanguage = monitorConfig.defaultLanguage`,
		`const defaultSampleWindow = monitorConfig.defaultSampleWindow`,
		`let currentLang = defaultLanguage`,
		`let currentSampleWindow = defaultSampleWindow`,
		`function applySampleWindow`,
		`visibleSamples(history.pidCPU)`,
		`function durationNS`,
		`return Math.max(1, Math.round(n)) + " ns"`,
		`@media (max-width: 980px)`,
		`#9b8ae3`,
		`rgba(155, 138, 227, 0.22)`,
		`#d96f72`,
		`data-status="live"`,
		`id="cpu-chart"`,
		`id="memory-chart"`,
		`id="goroutine-chart"`,
		`id="request-chart"`,
		`class="legend-dot"`,
		`unit: "%"`,
		`unit: "MiB"`,
		`unit: "req"`,
		`storageSet("monitor.theme"`,
		`storageSet("monitor.lang"`,
		`getContext("2d")`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("HTML body does not contain %q", want)
		}
	}
	for _, unwanted := range []string{
		`Real-time Go service status`,
		`JSON via Accept: application/json · in-browser history only`,
		`Updated</span>`,
		`Response</span>`,
		`Last 60 samples`,
		`最近 60 个采样点`,
		`navigator.language`,
		"</html>`",
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("HTML body contains unwanted %q", unwanted)
		}
	}
}

func TestMonitorHTMLUsesConfiguredUIDefaults(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{
		DefaultLanguage:     "zh-CN",
		DefaultSampleWindow: 30,
		Refresh:             time.Hour,
	})
	defer m.Stop()

	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/monitor", nil))

	body := rec.Body.String()
	for _, want := range []string{
		`"defaultLanguage":"zh-CN"`,
		`"defaultSampleWindow":30`,
		`data-samples="30" aria-pressed="true"`,
		`data-samples="60" aria-pressed="false"`,
		`applySampleWindow(defaultSampleWindow)`,
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
	if cfg.Description != defaultDescription {
		t.Fatalf("description = %q, want %q", cfg.Description, defaultDescription)
	}
	if cfg.Footer != defaultFooter {
		t.Fatalf("footer = %q, want %q", cfg.Footer, defaultFooter)
	}
	if cfg.DefaultLanguage != defaultLanguage {
		t.Fatalf("default language = %q, want %q", cfg.DefaultLanguage, defaultLanguage)
	}
	if cfg.DefaultSampleWindow != defaultSampleWindow {
		t.Fatalf("default sample window = %d, want %d", cfg.DefaultSampleWindow, defaultSampleWindow)
	}
	if cfg.Refresh != defaultRefresh {
		t.Fatalf("refresh = %s, want %s", cfg.Refresh, defaultRefresh)
	}
	if len(cfg.DiskPaths) != 0 {
		t.Fatalf("disk paths = %v, want empty", cfg.DiskPaths)
	}
}

func TestConfigValidatesUIDefaults(t *testing.T) {
	valid := applyConfig([]Config{
		{
			DefaultLanguage:     "zh-CN",
			DefaultSampleWindow: 90,
		},
	})
	if valid.DefaultLanguage != "zh-CN" {
		t.Fatalf("default language = %q, want zh-CN", valid.DefaultLanguage)
	}
	if valid.DefaultSampleWindow != 90 {
		t.Fatalf("default sample window = %d, want 90", valid.DefaultSampleWindow)
	}

	invalid := applyConfig([]Config{
		{
			DefaultLanguage:     "fr",
			DefaultSampleWindow: 45,
		},
	})
	if invalid.DefaultLanguage != defaultLanguage {
		t.Fatalf("default language = %q, want %q", invalid.DefaultLanguage, defaultLanguage)
	}
	if invalid.DefaultSampleWindow != defaultSampleWindow {
		t.Fatalf("default sample window = %d, want %d", invalid.DefaultSampleWindow, defaultSampleWindow)
	}
}

func TestConfigCopiesDiskPaths(t *testing.T) {
	paths := []string{"first", "second"}
	cfg := applyConfig([]Config{{DiskPaths: paths}})
	paths[0] = "changed"

	if cfg.DiskPaths[0] != "first" {
		t.Fatalf("disk paths were not copied: %v", cfg.DiskPaths)
	}
}

func assertMonitorHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want %q", got, "no-store")
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q, want %q", got, "no-referrer")
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}
