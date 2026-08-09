package monitor

import "time"

// StatsSchemaVersion identifies the JSON snapshot schema emitted by this
// version of monitor.
const StatsSchemaVersion = 1

// Stats is a point-in-time JSON snapshot of service, process, runtime, system,
// container, and HTTP metrics.
type Stats struct {
	SchemaVersion int             `json:"schema_version"`
	CollectedAt   time.Time       `json:"collected_at"`
	Collection    CollectionStats `json:"collection"`
	Service       ServiceStats    `json:"service"`
	PID           PIDStats        `json:"pid"`
	Runtime       RuntimeStats    `json:"runtime"`
	OS            OSStats         `json:"os"`
	Container     ContainerStats  `json:"container"`
	HTTP          HTTPStats       `json:"http"`
}

// CollectionStats describes one background collection pass.
type CollectionStats struct {
	DurationNS uint64   `json:"duration_ns"`
	Partial    bool     `json:"partial"`
	Errors     []string `json:"errors,omitempty"`
}

// ServiceStats describes the running service and its Go build metadata.
type ServiceStats struct {
	Name        string `json:"name,omitempty"`
	Version     string `json:"version,omitempty"`
	Environment string `json:"environment,omitempty"`
	GoVersion   string `json:"go_version"`
	Module      string `json:"module,omitempty"`
	Revision    string `json:"revision,omitempty"`
	VCSModified bool   `json:"vcs_modified,omitempty"`
}

// PIDStats describes the current process.
type PIDStats struct {
	CPUPercent float64 `json:"cpu_percent"`
	RSSBytes   uint64  `json:"rss_bytes"`
	PID        int32   `json:"pid"`
	Threads    int32   `json:"threads"`
	FDs        int32   `json:"fds"`
}

// RuntimeStats describes the Go runtime in the current process.
type RuntimeStats struct {
	Goroutines      int    `json:"goroutines"`
	GoroutinePeak   int    `json:"goroutine_peak"`
	HeapAllocBytes  uint64 `json:"heap_alloc_bytes"`
	HeapSysBytes    uint64 `json:"heap_sys_bytes"`
	HeapObjects     uint64 `json:"heap_objects"`
	NextGCBytes     uint64 `json:"next_gc_bytes"`
	Mallocs         uint64 `json:"mallocs"`
	Frees           uint64 `json:"frees"`
	NumGC           uint32 `json:"num_gc"`
	GCPauseLastNS   uint64 `json:"gc_pause_last_ns"`
	GCPauseTotalNS  uint64 `json:"gc_pause_total_ns"`
	GCPauseRecentNS uint64 `json:"gc_pause_recent_ns"`
	UptimeSeconds   uint64 `json:"uptime_seconds"`
}

// OSStats describes the host operating system.
type OSStats struct {
	CPUPercent        float64      `json:"cpu_percent"`
	MemoryUsedPercent float64      `json:"memory_used_percent"`
	MemoryTotalBytes  uint64       `json:"memory_total_bytes"`
	DiskUsedPercent   float64      `json:"disk_used_percent"`
	DiskTotalBytes    uint64       `json:"disk_total_bytes"`
	DiskUsedBytes     uint64       `json:"disk_used_bytes"`
	Disks             []DiskStats  `json:"disks"`
	Load1             float64      `json:"load1"`
	Network           NetworkStats `json:"network"`
}

// DiskStats describes usage for one configured filesystem path.
type DiskStats struct {
	Path        string  `json:"path"`
	Device      string  `json:"device,omitempty"`
	Fstype      string  `json:"fstype,omitempty"`
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkStats describes aggregate host network I/O, including loopback.
type NetworkStats struct {
	ReceivedBytes uint64  `json:"received_bytes"`
	SentBytes     uint64  `json:"sent_bytes"`
	ReceiveBPS    float64 `json:"receive_bps"`
	SendBPS       float64 `json:"send_bps"`
}

// ContainerStats describes Linux cgroup v2 limits and usage.
type ContainerStats struct {
	Detected          bool    `json:"detected"`
	MemoryUsageBytes  uint64  `json:"memory_usage_bytes,omitempty"`
	MemoryLimitBytes  uint64  `json:"memory_limit_bytes,omitempty"`
	MemoryUsedPercent float64 `json:"memory_used_percent,omitempty"`
	CPUQuotaCores     float64 `json:"cpu_quota_cores,omitempty"`
}

// HTTPStats describes requests seen by the middleware.
type HTTPStats struct {
	TotalRequests    uint64          `json:"total_requests"`
	InFlightRequests uint64          `json:"in_flight_requests"`
	RPS              float64         `json:"rps"`
	StatusCodes      StatusCodeStats `json:"status_codes"`
	Rates            HTTPRateStats   `json:"rates"`
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

// HTTPRateStats describes status rates for the most recent collection window.
type HTTPRateStats struct {
	Status4xxRate float64 `json:"4xx_rate"`
	Status5xxRate float64 `json:"5xx_rate"`
	ErrorRate     float64 `json:"error_rate"`
}

// LatencyStats describes business request duration in nanoseconds.
type LatencyStats struct {
	LastNS      uint64  `json:"last_ns"`
	RecentNS    float64 `json:"recent_ns"`
	P50NS       uint64  `json:"p50_ns"`
	P95NS       uint64  `json:"p95_ns"`
	P99NS       uint64  `json:"p99_ns"`
	RecentMaxNS uint64  `json:"recent_max_ns"`
	MaxNS       uint64  `json:"max_ns"`
}
