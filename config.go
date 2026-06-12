package monitor

import "time"

const (
	defaultPath    = "/monitor"
	defaultTitle   = "Monitor"
	defaultRefresh = 2 * time.Second
)

// Config controls the monitor middleware.
//
// The zero value is valid and uses Path "/monitor", Title "Monitor", and a
// refresh interval of 2 seconds.
type Config struct {
	// Path is the endpoint used for the HTML page and JSON snapshot.
	Path string

	// Title is shown in the HTML page title and heading.
	Title string

	// Refresh controls how often metrics are collected in the background.
	Refresh time.Duration

	// APIOnly makes Path return JSON even when the request does not ask for it.
	APIOnly bool
}

// DefaultConfig returns the default monitor configuration.
func DefaultConfig() Config {
	return Config{
		Path:    defaultPath,
		Title:   defaultTitle,
		Refresh: defaultRefresh,
	}
}

func applyConfig(configs []Config) Config {
	cfg := DefaultConfig()
	if len(configs) > 0 {
		cfg = configs[0]
	}
	if cfg.Path == "" {
		cfg.Path = defaultPath
	}
	if cfg.Path[0] != '/' {
		cfg.Path = "/" + cfg.Path
	}
	if cfg.Title == "" {
		cfg.Title = defaultTitle
	}
	if cfg.Refresh <= 0 {
		cfg.Refresh = defaultRefresh
	}
	return cfg
}
