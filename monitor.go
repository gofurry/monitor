package monitor

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// Monitor is a net/http middleware that exposes a lightweight status page and
// JSON metrics snapshot.
//
// Monitor is safe for concurrent use.
type Monitor struct {
	next http.Handler
	cfg  Config

	startedAt time.Time
	proc      *process.Process

	requests atomic.Uint64
	snapshot atomic.Value // stores Stats

	stopOnce sync.Once
	stopCh   chan struct{}
}

// New creates a monitor middleware around next.
//
// The returned handler exposes cfg.Path as both a HTML status page and a JSON
// snapshot when the request accepts application/json. Requests to cfg.Path are
// not counted as business requests.
func New(next http.Handler, config ...Config) http.Handler {
	return NewMonitor(next, config...)
}

// NewMonitor creates a Monitor around next.
//
// Use New for the shortest middleware setup. NewMonitor is useful when callers
// want to read Current or stop the background collector explicitly.
func NewMonitor(next http.Handler, config ...Config) *Monitor {
	if next == nil {
		next = http.NotFoundHandler()
	}

	m := &Monitor{
		next:      next,
		cfg:       applyConfig(config),
		startedAt: time.Now(),
		proc:      currentProcess(),
		stopCh:    make(chan struct{}),
	}
	m.start()
	return m
}

// ServeHTTP implements http.Handler.
func (m *Monitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == m.cfg.Path {
		m.serveMonitor(w, r)
		return
	}

	if !m.ignoreRequest(r) {
		m.requests.Add(1)
	}
	m.next.ServeHTTP(w, r)
}

// Current returns the most recently collected metrics snapshot.
func (m *Monitor) Current() Stats {
	v := m.snapshot.Load()
	if v == nil {
		return Stats{}
	}
	stats, ok := v.(Stats)
	if !ok {
		return Stats{}
	}
	return stats
}

// Stop stops the background collector. It is safe to call Stop more than once.
func (m *Monitor) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
	})
}

func (m *Monitor) start() {
	m.collectOnce()

	ticker := time.NewTicker(m.cfg.Refresh)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.collectOnce()
			case <-m.stopCh:
				return
			}
		}
	}()
}

func (m *Monitor) serveMonitor(w http.ResponseWriter, r *http.Request) {
	setMonitorHeaders(w)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if m.cfg.APIOnly || wantsJSON(r) {
		m.serveJSON(w)
		return
	}
	m.serveHTML(w)
}

func (m *Monitor) ignoreRequest(r *http.Request) bool {
	return m.cfg.IgnoreRequest != nil && m.cfg.IgnoreRequest(r)
}

func setMonitorHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func (m *Monitor) serveJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(m.Current())
}

func (m *Monitor) serveHTML(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(renderHTML(m.cfg)))
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}
