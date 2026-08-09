package monitor

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSnapshotMetadataAndPartialCollection(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{
		DiskPaths: []string{filepath.Join(t.TempDir(), "missing")},
		Refresh:   time.Hour,
	})
	defer m.Stop()

	stats := m.Current()
	if stats.SchemaVersion != StatsSchemaVersion {
		t.Fatalf("schema version = %d, want %d", stats.SchemaVersion, StatsSchemaVersion)
	}
	if stats.CollectedAt.IsZero() {
		t.Fatal("collected_at is zero")
	}
	if stats.Collection.DurationNS == 0 {
		t.Fatal("collection duration is zero")
	}
	if !stats.Collection.Partial || !containsString(stats.Collection.Errors, "os.disk") {
		t.Fatalf("collection = %+v, want partial os.disk error", stats.Collection)
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "file does not exist") {
		t.Fatalf("snapshot leaked an operating system error: %s", data)
	}
}

func TestHTTPWindowRatesAndLatency(t *testing.T) {
	var m Monitor
	m.latencyRecent.Store(ewmaUninitializedBits)
	for i := 0; i < 10; i++ {
		status := http.StatusOK
		if i < 2 {
			status = http.StatusBadRequest
		} else if i == 2 {
			status = http.StatusInternalServerError
		}
		m.ObserveRequest(status, time.Duration(i+1)*time.Millisecond)
	}

	stats := m.collectHTTP(2 * time.Second)
	if stats.RPS != 5 {
		t.Fatalf("RPS = %v, want 5", stats.RPS)
	}
	assertFloat(t, stats.Rates.Status4xxRate, 0.2)
	assertFloat(t, stats.Rates.Status5xxRate, 0.1)
	assertFloat(t, stats.Rates.ErrorRate, 0.3)
	if stats.Latency.P50NS != 5*uint64(time.Millisecond) {
		t.Fatalf("P50 = %d, want 5ms", stats.Latency.P50NS)
	}
	if stats.Latency.P95NS != 10*uint64(time.Millisecond) || stats.Latency.P99NS != 10*uint64(time.Millisecond) {
		t.Fatalf("P95/P99 = %d/%d, want 10ms/10ms", stats.Latency.P95NS, stats.Latency.P99NS)
	}
	if stats.Latency.RecentMaxNS != 10*uint64(time.Millisecond) {
		t.Fatalf("recent max = %d, want 10ms", stats.Latency.RecentMaxNS)
	}

	empty := m.collectHTTP(time.Second)
	if empty.RPS != 0 || empty.Rates.ErrorRate != 0 || empty.Latency.P50NS != 0 || empty.Latency.RecentMaxNS != 0 {
		t.Fatalf("empty window contains values: %+v", empty)
	}
}

func TestLatencyHistogramBoundaries(t *testing.T) {
	values := []uint64{
		0,
		uint64(time.Millisecond),
		5 * uint64(time.Millisecond),
		10 * uint64(time.Millisecond),
		25 * uint64(time.Millisecond),
		50 * uint64(time.Millisecond),
		100 * uint64(time.Millisecond),
		250 * uint64(time.Millisecond),
		500 * uint64(time.Millisecond),
		uint64(time.Second),
		2500 * uint64(time.Millisecond),
		5 * uint64(time.Second),
		6 * uint64(time.Second),
	}
	var histogram latencyHistogram
	for _, value := range values {
		histogram.observe(value)
	}
	window := histogram.snapshotAndReset()
	if window.buckets[0] != 2 {
		t.Fatalf("first bucket = %d, want 2", window.buckets[0])
	}
	for i := 1; i < len(window.buckets); i++ {
		if window.buckets[i] != 1 {
			t.Fatalf("bucket %d = %d, want 1", i, window.buckets[i])
		}
	}
	if window.maxNS != 6*uint64(time.Second) {
		t.Fatalf("window max = %d, want 6s", window.maxNS)
	}
	empty := histogram.snapshotAndReset()
	if empty.percentile(50) != 0 || empty.maxNS != 0 {
		t.Fatalf("reset histogram is not empty: %+v", empty)
	}
}

func TestLatencyHistogramConcurrent(t *testing.T) {
	var histogram latencyHistogram
	var wg sync.WaitGroup
	for shard := range uint64(32) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				histogram.observeSharded(25*uint64(time.Millisecond), shard)
			}
		}()
	}
	wg.Wait()
	window := histogram.snapshotAndReset()
	var total uint64
	for _, count := range window.buckets {
		total += count
	}
	if total != 32000 {
		t.Fatalf("histogram count = %d, want 32000", total)
	}
}

func TestRequestLifecycleDoesNotUnderflow(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{Refresh: time.Hour})
	defer m.Stop()

	m.RequestFinished(http.StatusOK, time.Millisecond)
	if got := m.inFlight.Load(); got != 0 {
		t.Fatalf("in-flight after unmatched finish = %d, want 0", got)
	}
	m.RequestStarted()
	m.RequestFinished(http.StatusOK, time.Millisecond)
	m.RequestFinished(http.StatusOK, time.Millisecond)
	if got := m.inFlight.Load(); got != 0 {
		t.Fatalf("in-flight after double finish = %d, want 0", got)
	}
}

func TestBeginRequestIsIdempotent(t *testing.T) {
	m := NewMonitor(http.NotFoundHandler(), Config{Refresh: time.Hour})
	defer m.Stop()

	finish := m.BeginRequest()
	finish(http.StatusCreated)
	finish(http.StatusInternalServerError)
	if got := m.requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
	if got := m.status2xx.Load(); got != 1 {
		t.Fatalf("2xx = %d, want 1", got)
	}
	if got := m.status5xx.Load(); got != 0 {
		t.Fatalf("5xx = %d, want 0", got)
	}
}

func TestMonitorAuthorization(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		acceptJSON bool
		authorized bool
		wantStatus int
	}{
		{name: "HTML allowed", method: http.MethodGet, authorized: true, wantStatus: http.StatusOK},
		{name: "JSON allowed", method: http.MethodGet, acceptJSON: true, authorized: true, wantStatus: http.StatusOK},
		{name: "HEAD denied", method: http.MethodHead, wantStatus: http.StatusUnauthorized},
		{name: "POST denied before method check", method: http.MethodPost, wantStatus: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMonitor(http.NotFoundHandler(), Config{
				Refresh: time.Hour,
				Authorize: func(r *http.Request) bool {
					return r.Header.Get("X-Allow") == "yes"
				},
			})
			defer m.Stop()
			req := httptest.NewRequest(tt.method, "/monitor", nil)
			if tt.acceptJSON {
				req.Header.Set("Accept", "application/json")
			}
			if tt.authorized {
				req.Header.Set("X-Allow", "yes")
			}
			rec := httptest.NewRecorder()
			m.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}

	m := NewMonitor(http.NotFoundHandler(), Config{Refresh: time.Hour})
	defer m.Stop()
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/monitor", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("nil Authorize status = %d, want 200", rec.Code)
	}
}

func TestRefreshIsClamped(t *testing.T) {
	cfg := applyConfig([]Config{{Refresh: time.Millisecond}})
	if cfg.Refresh != minRefresh {
		t.Fatalf("refresh = %s, want %s", cfg.Refresh, minRefresh)
	}
}

func TestServiceStatsOverridesAndPrivacy(t *testing.T) {
	stats := collectServiceStats(Config{
		ServiceName: "example-api",
		Version:     "v1.2.0",
		Environment: "production",
	})
	if stats.Name != "example-api" || stats.Version != "v1.2.0" || stats.Environment != "production" {
		t.Fatalf("service stats = %+v", stats)
	}
	if stats.GoVersion == "" {
		t.Fatal("Go version is empty")
	}
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"executable", "working_directory", "gopath", "remote_url"} {
		if strings.Contains(strings.ToLower(string(data)), forbidden) {
			t.Fatalf("service JSON contains private field %q: %s", forbidden, data)
		}
	}
}

func TestCgroupV2Collection(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "memory.current", "536870912\n")
	writeTestFile(t, root, "memory.max", "1073741824\n")
	writeTestFile(t, root, "cpu.max", "200000 100000\n")

	stats, errors := collectCgroupV2(root, true)
	if len(errors) != 0 {
		t.Fatalf("errors = %v", errors)
	}
	if !stats.Detected || stats.MemoryUsageBytes != 536870912 || stats.MemoryLimitBytes != 1073741824 {
		t.Fatalf("container stats = %+v", stats)
	}
	assertFloat(t, stats.MemoryUsedPercent, 50)
	assertFloat(t, stats.CPUQuotaCores, 2)
}

func TestCgroupV2UnlimitedAndDetection(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "memory.current", "1024")
	writeTestFile(t, root, "memory.max", "max")
	writeTestFile(t, root, "cpu.max", "max 100000")

	stats, errors := collectCgroupV2(root, true)
	if len(errors) != 0 || !stats.Detected || stats.MemoryLimitBytes != 0 || stats.CPUQuotaCores != 0 {
		t.Fatalf("unlimited stats = %+v, errors = %v", stats, errors)
	}
	undetected, errors := collectCgroupV2(root, false)
	if undetected.Detected || len(errors) != 0 {
		t.Fatalf("unhinted unlimited cgroup = %+v, errors = %v", undetected, errors)
	}
}

func TestCgroupV2ErrorsUseStableIdentifiers(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "memory.current", "invalid")
	writeTestFile(t, root, "memory.max", "invalid")
	stats, errors := collectCgroupV2(root, true)
	if !stats.Detected {
		t.Fatal("hinted cgroup was not detected")
	}
	if !containsString(errors, "container.memory") || !containsString(errors, "container.cpu") {
		t.Fatalf("errors = %v", errors)
	}
}

func TestNetworkStatsFromCounters(t *testing.T) {
	stats := networkStatsFromCounters(3000, 5000, 1000, 1000, 2*time.Second)
	if stats.ReceivedBytes != 3000 || stats.SentBytes != 5000 || stats.ReceiveBPS != 1000 || stats.SendBPS != 2000 {
		t.Fatalf("network stats = %+v", stats)
	}
	first := networkStatsFromCounters(3000, 5000, 0, 0, 0)
	if first.ReceiveBPS != 0 || first.SendBPS != 0 {
		t.Fatalf("first network rates = %+v, want zero", first)
	}
	reset := networkStatsFromCounters(100, 100, 3000, 5000, time.Second)
	if reset.ReceiveBPS != 0 || reset.SendBPS != 0 {
		t.Fatalf("reset network rates = %+v, want zero", reset)
	}
}

func assertFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("value = %v, want %v", got, want)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func writeTestFile(t *testing.T, root, name, value string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
