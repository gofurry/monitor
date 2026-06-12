package monitor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
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

	requests      atomic.Uint64
	inFlight      atomic.Uint64
	status1xx     atomic.Uint64
	status2xx     atomic.Uint64
	status3xx     atomic.Uint64
	status4xx     atomic.Uint64
	status5xx     atomic.Uint64
	latencyLastNS atomic.Uint64
	latencyRecent atomic.Uint64
	latencyMaxNS  atomic.Uint64
	gcPauseTotal  atomic.Uint64
	gcPauseSeen   atomic.Bool
	goroutinePeak atomic.Uint64
	snapshot      atomic.Value // stores Stats

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
	m.latencyRecent.Store(ewmaUninitializedBits)
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
		m.serveBusiness(w, r)
		return
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

func (m *Monitor) serveBusiness(w http.ResponseWriter, r *http.Request) {
	m.requests.Add(1)
	m.inFlight.Add(1)
	started := time.Now()
	rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	defer func() {
		m.inFlight.Add(^uint64(0))
		m.recordBusinessRequest(rw.status, time.Since(started))
	}()

	m.next.ServeHTTP(rw, r)
}

func (m *Monitor) recordBusinessRequest(status int, duration time.Duration) {
	if status < 100 {
		status = http.StatusOK
	}
	switch {
	case status < 200:
		m.status1xx.Add(1)
	case status < 300:
		m.status2xx.Add(1)
	case status < 400:
		m.status3xx.Add(1)
	case status < 500:
		m.status4xx.Add(1)
	default:
		m.status5xx.Add(1)
	}

	if duration <= 0 {
		duration = time.Nanosecond
	}
	ns := uint64(duration.Nanoseconds())
	m.latencyLastNS.Store(ns)
	m.updateRecentLatency(ns)
	updateMaxUint64(&m.latencyMaxNS, ns)
}

const (
	latencyEWMAAlpha      = 0.2
	ewmaUninitializedBits = 0x7ff8_0000_0000_0001
)

func (m *Monitor) updateRecentLatency(ns uint64) {
	updateEWMA(&m.latencyRecent, float64(ns))
}

func (m *Monitor) loadRecentLatencyNS() float64 {
	return loadEWMA(&m.latencyRecent)
}

func updateEWMA(value *atomic.Uint64, sample float64) {
	for {
		currentBits := value.Load()
		next := sample
		if currentBits != ewmaUninitializedBits {
			current := math.Float64frombits(currentBits)
			next = current + latencyEWMAAlpha*(sample-current)
		}
		if value.CompareAndSwap(currentBits, math.Float64bits(next)) {
			return
		}
	}
}

func loadEWMA(value *atomic.Uint64) float64 {
	currentBits := value.Load()
	if currentBits == ewmaUninitializedBits {
		return 0
	}
	return math.Float64frombits(currentBits)
}

func updateMaxUint64(value *atomic.Uint64, next uint64) {
	for {
		current := value.Load()
		if next <= current || value.CompareAndSwap(current, next) {
			return
		}
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(p)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		if !r.wroteHeader {
			r.WriteHeader(http.StatusOK)
		}
		flusher.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}

func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	pusher, ok := r.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (r *statusRecorder) ReadFrom(src io.Reader) (int64, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	if readerFrom, ok := r.ResponseWriter.(io.ReaderFrom); ok {
		return readerFrom.ReadFrom(src)
	}
	return io.Copy(r.ResponseWriter, src)
}

func setMonitorHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
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
