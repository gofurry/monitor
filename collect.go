package monitor

import (
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

func (m *Monitor) collectOnce() {
	stats := Stats{
		PID:     collectPID(m.proc),
		Runtime: collectRuntime(m.startedAt),
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

func collectRuntime(startedAt time.Time) RuntimeStats {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	uptime := time.Since(startedAt)
	if uptime < 0 {
		uptime = 0
	}

	return RuntimeStats{
		Goroutines:     runtime.NumGoroutine(),
		HeapAllocBytes: ms.HeapAlloc,
		HeapSysBytes:   ms.HeapSys,
		NumGC:          ms.NumGC,
		UptimeSeconds:  uint64(uptime.Seconds()),
	}
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
