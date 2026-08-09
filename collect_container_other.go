//go:build !linux

package monitor

func collectContainer() (ContainerStats, []string) {
	return ContainerStats{}, nil
}
