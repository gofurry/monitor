package monitor

// Stats is a point-in-time JSON snapshot of process, runtime, system, and HTTP
// metrics.
type Stats struct {
	PID     PIDStats     `json:"pid"`
	Runtime RuntimeStats `json:"runtime"`
	OS      OSStats      `json:"os"`
	HTTP    HTTPStats    `json:"http"`
}

// PIDStats describes the current process.
type PIDStats struct {
	CPUPercent float64 `json:"cpu_percent"`
	RSSBytes   uint64  `json:"rss_bytes"`
}

// RuntimeStats describes the Go runtime in the current process.
type RuntimeStats struct {
	Goroutines     int    `json:"goroutines"`
	HeapAllocBytes uint64 `json:"heap_alloc_bytes"`
	HeapSysBytes   uint64 `json:"heap_sys_bytes"`
	NumGC          uint32 `json:"num_gc"`
	UptimeSeconds  uint64 `json:"uptime_seconds"`
}

// OSStats describes the host operating system.
type OSStats struct {
	CPUPercent        float64 `json:"cpu_percent"`
	MemoryUsedPercent float64 `json:"memory_used_percent"`
	MemoryTotalBytes  uint64  `json:"memory_total_bytes"`
	Load1             float64 `json:"load1"`
}

// HTTPStats describes requests seen by the middleware.
type HTTPStats struct {
	TotalRequests    uint64          `json:"total_requests"`
	InFlightRequests uint64          `json:"in_flight_requests"`
	StatusCodes      StatusCodeStats `json:"status_codes"`
	Latency          LatencyStats    `json:"latency"`
}

// StatusCodeStats groups HTTP responses by status code class.
type StatusCodeStats struct {
	Status1xx uint64 `json:"1xx"`
	Status2xx uint64 `json:"2xx"`
	Status3xx uint64 `json:"3xx"`
	Status4xx uint64 `json:"4xx"`
	Status5xx uint64 `json:"5xx"`
}

// LatencyStats describes business request duration in nanoseconds.
type LatencyStats struct {
	LastNS   uint64  `json:"last_ns"`
	RecentNS float64 `json:"recent_ns"`
	MaxNS    uint64  `json:"max_ns"`
}
