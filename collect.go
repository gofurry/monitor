package monitor

import (
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gopsnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

func (m *Monitor) collectOnce() {
	started := time.Now()
	var elapsed time.Duration
	if !m.lastCollectedAt.IsZero() {
		elapsed = started.Sub(m.lastCollectedAt)
	}

	pidStats, pidErrors := collectPID(m.proc)
	osStats, osErrors := m.collectOS(elapsed)
	containerStats, containerErrors := collectContainer()
	errors := appendCollectionErrors(nil, pidErrors...)
	errors = appendCollectionErrors(errors, osErrors...)
	errors = appendCollectionErrors(errors, containerErrors...)

	stats := Stats{
		SchemaVersion: StatsSchemaVersion,
		Service:       m.service,
		PID:           pidStats,
		Runtime:       m.collectRuntime(),
		OS:            osStats,
		Container:     containerStats,
		HTTP:          m.collectHTTP(elapsed),
	}
	finished := time.Now()
	stats.CollectedAt = finished.UTC()
	duration := finished.Sub(started)
	if duration < 0 {
		duration = 0
	}
	stats.Collection = CollectionStats{
		DurationNS: uint64(duration.Nanoseconds()),
		Partial:    len(errors) > 0,
		Errors:     errors,
	}
	m.lastCollectedAt = started
	m.snapshot.Store(stats)
}

func appendCollectionErrors(existing []string, values ...string) []string {
	for _, value := range values {
		if value == "" {
			continue
		}
		duplicate := false
		for _, current := range existing {
			if current == value {
				duplicate = true
				break
			}
		}
		if !duplicate {
			existing = append(existing, value)
		}
	}
	return existing
}

func (m *Monitor) collectHTTP(elapsed time.Duration) HTTPStats {
	totalRequests := m.requests.Load()
	completedRequests := m.completedRequests.Load()
	statusCodes := StatusCodeStats{
		Status1xx: m.status1xx.Load(),
		Status2xx: m.status2xx.Load(),
		Status3xx: m.status3xx.Load(),
		Status4xx: m.status4xx.Load(),
		Status5xx: m.status5xx.Load(),
	}

	requestDelta := counterDelta(totalRequests, m.lastRequests)
	completedDelta := counterDelta(completedRequests, m.lastCompletedRequests)
	status4xxDelta := counterDelta(statusCodes.Status4xx, m.lastStatus4xx)
	status5xxDelta := counterDelta(statusCodes.Status5xx, m.lastStatus5xx)
	window := m.latencyHistogram.snapshotAndReset()

	m.lastRequests = totalRequests
	m.lastCompletedRequests = completedRequests
	m.lastStatus4xx = statusCodes.Status4xx
	m.lastStatus5xx = statusCodes.Status5xx

	var rates HTTPRateStats
	if completedDelta > 0 {
		rates.Status4xxRate = float64(status4xxDelta) / float64(completedDelta)
		rates.Status5xxRate = float64(status5xxDelta) / float64(completedDelta)
		rates.ErrorRate = float64(status4xxDelta+status5xxDelta) / float64(completedDelta)
	}

	return HTTPStats{
		TotalRequests:    totalRequests,
		InFlightRequests: m.inFlight.Load(),
		RPS:              ratePerSecond(requestDelta, elapsed),
		StatusCodes:      statusCodes,
		Rates:            rates,
		Latency: LatencyStats{
			LastNS:      m.latencyLastNS.Load(),
			RecentNS:    m.loadRecentLatencyNS(),
			P50NS:       window.percentile(50),
			P95NS:       window.percentile(95),
			P99NS:       window.percentile(99),
			RecentMaxNS: window.maxNS,
			MaxNS:       m.latencyMaxNS.Load(),
		},
	}
}

func counterDelta(current, previous uint64) uint64 {
	if current < previous {
		return 0
	}
	return current - previous
}

func ratePerSecond(delta uint64, elapsed time.Duration) float64 {
	if delta == 0 || elapsed <= 0 {
		return 0
	}
	return float64(delta) / elapsed.Seconds()
}

func collectPID(proc *process.Process) (PIDStats, []string) {
	var stats PIDStats
	if proc == nil {
		return stats, []string{"pid.cpu", "pid.memory", "pid.threads", "pid.fds"}
	}

	stats.PID = proc.Pid
	var errors []string
	if percent, err := proc.CPUPercent(); err == nil {
		stats.CPUPercent = percent
	} else {
		errors = append(errors, "pid.cpu")
	}
	if info, err := proc.MemoryInfo(); err == nil && info != nil {
		stats.RSSBytes = info.RSS
	} else {
		errors = append(errors, "pid.memory")
	}
	if threads, err := proc.NumThreads(); err == nil {
		stats.Threads = threads
	} else {
		errors = append(errors, "pid.threads")
	}
	if fds, err := proc.NumFDs(); err == nil {
		stats.FDs = fds
	} else {
		errors = append(errors, "pid.fds")
	}
	return stats, errors
}

func (m *Monitor) collectRuntime() RuntimeStats {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	uptime := time.Since(m.startedAt)
	if uptime < 0 {
		uptime = 0
	}
	pauseTotal := ms.PauseTotalNs
	pausePrevious := m.gcPauseTotal.Swap(pauseTotal)
	pauseRecent := uint64(0)
	if m.gcPauseSeen.Swap(true) && pauseTotal >= pausePrevious {
		pauseRecent = pauseTotal - pausePrevious
	}
	goroutines := runtime.NumGoroutine()
	updateMaxUint64(&m.goroutinePeak, uint64(goroutines))

	return RuntimeStats{
		Goroutines:      goroutines,
		GoroutinePeak:   int(m.goroutinePeak.Load()),
		HeapAllocBytes:  ms.HeapAlloc,
		HeapSysBytes:    ms.HeapSys,
		HeapObjects:     ms.HeapObjects,
		NextGCBytes:     ms.NextGC,
		Mallocs:         ms.Mallocs,
		Frees:           ms.Frees,
		NumGC:           ms.NumGC,
		GCPauseLastNS:   lastGCPauseNS(ms),
		GCPauseTotalNS:  pauseTotal,
		GCPauseRecentNS: pauseRecent,
		UptimeSeconds:   uint64(uptime.Seconds()),
	}
}

func lastGCPauseNS(ms runtime.MemStats) uint64 {
	if ms.NumGC == 0 {
		return 0
	}
	index := (ms.NumGC + uint32(len(ms.PauseNs)) - 1) % uint32(len(ms.PauseNs))
	return ms.PauseNs[index]
}

func (m *Monitor) collectOS(elapsed time.Duration) (OSStats, []string) {
	var stats OSStats
	var errors []string
	if values, err := cpu.Percent(0, false); err == nil && len(values) > 0 {
		stats.CPUPercent = values[0]
	} else {
		errors = append(errors, "os.cpu")
	}
	if vm, err := mem.VirtualMemory(); err == nil && vm != nil {
		stats.MemoryUsedPercent = vm.UsedPercent
		stats.MemoryTotalBytes = vm.Total
	} else {
		errors = append(errors, "os.memory")
	}
	var diskFailed bool
	stats.Disks, diskFailed = collectDisks(m.cfg.DiskPaths)
	if diskFailed {
		errors = append(errors, "os.disk")
	}
	if len(stats.Disks) > 0 {
		stats.DiskUsedPercent = stats.Disks[0].UsedPercent
		stats.DiskTotalBytes = stats.Disks[0].TotalBytes
		stats.DiskUsedBytes = stats.Disks[0].UsedBytes
	}
	if avg, err := load.Avg(); err == nil && avg != nil {
		stats.Load1 = avg.Load1
	} else {
		errors = append(errors, "os.load")
	}
	var networkFailed bool
	stats.Network, networkFailed = m.collectNetwork(elapsed)
	if networkFailed {
		errors = append(errors, "os.network")
	}
	return stats, errors
}

func (m *Monitor) collectNetwork(elapsed time.Duration) (NetworkStats, bool) {
	values, err := gopsnet.IOCounters(false)
	if err != nil || len(values) == 0 {
		return NetworkStats{}, true
	}
	current := values[0]
	stats := networkStatsFromCounters(current.BytesRecv, current.BytesSent, m.lastNetworkReceived, m.lastNetworkSent, elapsed)
	m.lastNetworkReceived = current.BytesRecv
	m.lastNetworkSent = current.BytesSent
	return stats, false
}

func networkStatsFromCounters(received, sent, previousReceived, previousSent uint64, elapsed time.Duration) NetworkStats {
	return NetworkStats{
		ReceivedBytes: received,
		SentBytes:     sent,
		ReceiveBPS:    ratePerSecond(counterDelta(received, previousReceived), elapsed),
		SendBPS:       ratePerSecond(counterDelta(sent, previousSent), elapsed),
	}
}

func collectDisks(paths []string) ([]DiskStats, bool) {
	targets := paths
	if len(targets) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return nil, true
		}
		targets = []string{wd}
	}

	partitions := partitionLookup()
	disks := make([]DiskStats, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	failed := false
	for _, path := range targets {
		if path == "" {
			failed = true
			continue
		}
		usage, err := disk.Usage(path)
		if err != nil || usage == nil {
			failed = true
			continue
		}
		key := usage.Path
		if key == "" {
			key = path
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		stat := DiskStats{
			Path:        key,
			Fstype:      usage.Fstype,
			TotalBytes:  usage.Total,
			UsedBytes:   usage.Used,
			FreeBytes:   usage.Free,
			UsedPercent: usage.UsedPercent,
		}
		if partition, ok := partitions[key]; ok {
			stat.Device = partition.Device
			if stat.Fstype == "" {
				stat.Fstype = partition.Fstype
			}
		}
		disks = append(disks, stat)
	}
	return disks, failed || len(disks) == 0
}

func partitionLookup() map[string]disk.PartitionStat {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil
	}
	lookup := make(map[string]disk.PartitionStat, len(partitions))
	for _, partition := range partitions {
		lookup[partition.Mountpoint] = partition
	}
	return lookup
}

func currentProcess() *process.Process {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil
	}
	return proc
}
