package monitor

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func collectCgroupV2(root string, containerHint bool) (ContainerStats, []string) {
	memoryMaxRaw, memoryMaxErr := readCgroupValue(root, "memory.max")
	cpuMaxRaw, cpuMaxErr := readCgroupValue(root, "cpu.max")

	memoryLimit, memoryUnlimited, memoryLimitValid := parseMemoryMax(memoryMaxRaw)
	cpuQuota, cpuUnlimited, cpuQuotaValid := parseCPUMax(cpuMaxRaw)
	finiteLimit := (memoryLimitValid && !memoryUnlimited) || (cpuQuotaValid && !cpuUnlimited)
	if !containerHint && !finiteLimit {
		return ContainerStats{}, nil
	}

	stats := ContainerStats{Detected: true}
	var errors []string
	memoryCurrentRaw, memoryCurrentErr := readCgroupValue(root, "memory.current")
	memoryUsage, memoryUsageErr := strconv.ParseUint(memoryCurrentRaw, 10, 64)
	if memoryCurrentErr != nil || memoryUsageErr != nil || memoryMaxErr != nil || !memoryLimitValid {
		errors = append(errors, "container.memory")
	} else {
		stats.MemoryUsageBytes = memoryUsage
		if !memoryUnlimited {
			stats.MemoryLimitBytes = memoryLimit
			if memoryLimit > 0 {
				stats.MemoryUsedPercent = float64(memoryUsage) / float64(memoryLimit) * 100
			}
		}
	}

	if cpuMaxErr != nil || !cpuQuotaValid {
		errors = append(errors, "container.cpu")
	} else if !cpuUnlimited {
		stats.CPUQuotaCores = cpuQuota
	}
	return stats, errors
}

func readCgroupValue(root, name string) (string, error) {
	value, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(value)), nil
}

func parseMemoryMax(value string) (limit uint64, unlimited, valid bool) {
	if value == "max" {
		return 0, true, true
	}
	limit, err := strconv.ParseUint(value, 10, 64)
	return limit, false, err == nil
}

func parseCPUMax(value string) (cores float64, unlimited, valid bool) {
	fields := strings.Fields(value)
	if len(fields) != 2 {
		return 0, false, false
	}
	period, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil || period == 0 {
		return 0, false, false
	}
	if fields[0] == "max" {
		return 0, true, true
	}
	quota, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0, false, false
	}
	return float64(quota) / float64(period), false, true
}
