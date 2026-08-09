package monitor

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

func collectServiceStats(cfg Config) ServiceStats {
	stats := ServiceStats{
		Name:        strings.TrimSpace(cfg.ServiceName),
		Version:     strings.TrimSpace(cfg.Version),
		Environment: strings.TrimSpace(cfg.Environment),
		GoVersion:   runtime.Version(),
	}

	if info, ok := debug.ReadBuildInfo(); ok && info != nil {
		stats.Module = info.Main.Path
		if stats.Version == "" {
			stats.Version = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				stats.Revision = setting.Value
			case "vcs.modified":
				stats.VCSModified = setting.Value == "true"
			}
		}
	}
	if stats.Version == "" {
		stats.Version = "(devel)"
	}
	if stats.Name == "" {
		stats.Name = executableName()
	}
	return stats
}

func executableName() string {
	if executable, err := os.Executable(); err == nil {
		if name := filepath.Base(executable); name != "." && name != string(filepath.Separator) {
			return name
		}
	}
	if len(os.Args) > 0 {
		return filepath.Base(os.Args[0])
	}
	return ""
}
