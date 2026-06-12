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
	TotalRequests uint64 `json:"total_requests"`
}
