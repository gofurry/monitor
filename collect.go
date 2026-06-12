package monitor

import (
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

func (m *Monitor) collectOnce() {
	stats := Stats{
		PID:     collectPID(m.proc),
		Runtime: m.collectRuntime(),
		OS:      collectOS(),
		HTTP:    m.collectHTTP(),
	}
	m.snapshot.Store(stats)
}

func (m *Monitor) collectHTTP() HTTPStats {
	statusCodes := StatusCodeStats{
		Status1xx: m.status1xx.Load(),
		Status2xx: m.status2xx.Load(),
		Status3xx: m.status3xx.Load(),
		Status4xx: m.status4xx.Load(),
		Status5xx: m.status5xx.Load(),
	}

	return HTTPStats{
		TotalRequests:    m.requests.Load(),
		InFlightRequests: m.inFlight.Load(),
		StatusCodes:      statusCodes,
		Latency: LatencyStats{
			LastNS:   m.latencyLastNS.Load(),
			RecentNS: m.loadRecentLatencyNS(),
			MaxNS:    m.latencyMaxNS.Load(),
		},
	}
}

func collectPID(proc *process.Process) PIDStats {
	var stats PIDStats
	if proc == nil {
		return stats
	}

	if percent, err := proc.CPUPercent(); err == nil {
		stats.CPUPercent = percent
	}
	if info, err := proc.MemoryInfo(); err == nil && info != nil {
		stats.RSSBytes = info.RSS
	}
	return stats
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

func collectOS() OSStats {
	var stats OSStats
	if values, err := cpu.Percent(0, false); err == nil && len(values) > 0 {
		stats.CPUPercent = values[0]
	}
	if vm, err := mem.VirtualMemory(); err == nil && vm != nil {
		stats.MemoryUsedPercent = vm.UsedPercent
		stats.MemoryTotalBytes = vm.Total
	}
	if wd, err := os.Getwd(); err == nil {
		if usage, err := disk.Usage(wd); err == nil && usage != nil {
			stats.DiskUsedPercent = usage.UsedPercent
			stats.DiskTotalBytes = usage.Total
			stats.DiskUsedBytes = usage.Used
		}
	}
	if avg, err := load.Avg(); err == nil && avg != nil {
		stats.Load1 = avg.Load1
	}
	return stats
}

func currentProcess() *process.Process {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil
	}
	return proc
}
