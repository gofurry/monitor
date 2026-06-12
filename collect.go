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
		OS:      m.collectOS(),
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

func (m *Monitor) collectOS() OSStats {
	var stats OSStats
	if values, err := cpu.Percent(0, false); err == nil && len(values) > 0 {
		stats.CPUPercent = values[0]
	}
	if vm, err := mem.VirtualMemory(); err == nil && vm != nil {
		stats.MemoryUsedPercent = vm.UsedPercent
		stats.MemoryTotalBytes = vm.Total
	}
	stats.Disks = collectDisks(m.cfg.DiskPaths)
	if len(stats.Disks) > 0 {
		stats.DiskUsedPercent = stats.Disks[0].UsedPercent
		stats.DiskTotalBytes = stats.Disks[0].TotalBytes
		stats.DiskUsedBytes = stats.Disks[0].UsedBytes
	}
	if avg, err := load.Avg(); err == nil && avg != nil {
		stats.Load1 = avg.Load1
	}
	return stats
}

func collectDisks(paths []string) []DiskStats {
	targets := paths
	if len(targets) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return nil
		}
		targets = []string{wd}
	}

	partitions := partitionLookup()
	disks := make([]DiskStats, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, path := range targets {
		if path == "" {
			continue
		}
		usage, err := disk.Usage(path)
		if err != nil || usage == nil {
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
	return disks
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
