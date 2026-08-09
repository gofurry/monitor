//go:build linux

package monitor

import (
	"os"
	"strings"
)

const defaultCgroupRoot = "/sys/fs/cgroup"

func collectContainer() (ContainerStats, []string) {
	return collectCgroupV2(defaultCgroupRoot, hasContainerHint())
}

func hasContainerHint() bool {
	for _, marker := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(marker); err == nil {
			return true
		}
	}

	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return false
	}
	cgroup := strings.ToLower(string(data))
	for _, marker := range []string{"docker", "kubepods", "containerd", "libpod", "podman"} {
		if strings.Contains(cgroup, marker) {
			return true
		}
	}
	return false
}
