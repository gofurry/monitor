package monitor

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var benchmarkTotalRequests uint64

//go:noinline
func benchmarkCount(p *uint64) {
	(*p)++
}

type benchmarkResponseWriter struct {
	header http.Header
	status int
}

func newBenchmarkResponseWriter() *benchmarkResponseWriter {
	return &benchmarkResponseWriter{header: make(http.Header)}
}

func (w *benchmarkResponseWriter) Header() http.Header {
	return w.header
}

func (w *benchmarkResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *benchmarkResponseWriter) WriteHeader(status int) {
	w.status = status
}

func BenchmarkDirectHandler(b *testing.B) {
	var calls uint64
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rw := newBenchmarkResponseWriter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		benchmarkCount(&calls)
		w.WriteHeader(http.StatusNoContent)
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(rw, req)
	}
	benchmarkTotalRequests = calls
}

func BenchmarkMonitorBusinessRequest(b *testing.B) {
	var calls uint64
	m := NewMonitor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		benchmarkCount(&calls)
		w.WriteHeader(http.StatusNoContent)
	}), Config{
		Refresh: time.Hour,
	})
	defer m.Stop()

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rw := newBenchmarkResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ServeHTTP(rw, req)
	}
	benchmarkTotalRequests = calls
}

func BenchmarkMonitorBusinessRequestParallel(b *testing.B) {
	m := NewMonitor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), Config{
		Refresh: time.Hour,
	})
	defer m.Stop()

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rw := newBenchmarkResponseWriter()
		for pb.Next() {
			m.ServeHTTP(rw, req)
		}
	})
}

func BenchmarkMonitorIgnoredRequest(b *testing.B) {
	var calls uint64
	m := NewMonitor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		benchmarkCount(&calls)
		w.WriteHeader(http.StatusNoContent)
	}), Config{
		Refresh: time.Hour,
		IgnoreRequest: func(r *http.Request) bool {
			return r.URL.Path == "/healthz"
		},
	})
	defer m.Stop()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rw := newBenchmarkResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ServeHTTP(rw, req)
	}
	benchmarkTotalRequests = calls
}

func BenchmarkMonitorJSONSnapshot(b *testing.B) {
	m := NewMonitor(http.NotFoundHandler(), Config{Refresh: time.Hour})
	defer m.Stop()

	req := httptest.NewRequest(http.MethodGet, "/monitor", nil)
	req.Header.Set("Accept", "application/json")
	rw := newBenchmarkResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ServeHTTP(rw, req)
	}
}

func BenchmarkMonitorHTMLPage(b *testing.B) {
	m := NewMonitor(http.NotFoundHandler(), Config{Refresh: time.Hour})
	defer m.Stop()

	req := httptest.NewRequest(http.MethodGet, "/monitor", nil)
	rw := newBenchmarkResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ServeHTTP(rw, req)
	}
}

func BenchmarkMonitorCurrent(b *testing.B) {
	m := NewMonitor(http.NotFoundHandler(), Config{Refresh: time.Hour})
	defer m.Stop()

	var total uint64

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats := m.Current()
		total += stats.HTTP.TotalRequests
	}
	benchmarkTotalRequests = total
}
