package monitor

import (
	"net/http"
	"time"
)

const (
	defaultPath         = "/monitor"
	defaultTitle        = "Monitor"
	defaultDescription  = "Live process, runtime, system, and HTTP metrics for this Go service."
	defaultFooter       = "Powered by github.com/gofurry/monitor - MIT License."
	defaultLanguage     = "en"
	defaultTheme        = "dark"
	defaultBackground   = "solid"
	defaultSampleWindow = 60
	defaultRefresh      = 2 * time.Second
)

// Config controls the monitor middleware.
//
// The zero value is valid and uses Path "/monitor", Title "Monitor", a short
// page description, an MIT License footer, English UI, dark theme, solid
// background, 60 trend samples, and a refresh interval of 2 seconds.
type Config struct {
	// Path is the endpoint used for the HTML page and JSON snapshot.
	Path string

	// Title is shown in the HTML page title and heading.
	Title string

	// Description is shown below the page header. It is intended for a short
	// service note or deployment context.
	Description string

	// Footer is shown at the bottom of the HTML page. It is intended for
	// copyright, ownership, or license text.
	Footer string

	// DefaultLanguage controls the initial HTML UI language when the browser has
	// no saved monitor language preference. Supported values are "en" and
	// "zh-CN". Empty or unsupported values use "en".
	DefaultLanguage string

	// DefaultTheme controls the initial HTML UI theme when the browser has no
	// saved monitor theme preference. Supported values are "light" and "dark".
	// Empty or unsupported values use "dark".
	DefaultTheme string

	// Background controls the HTML page background. Supported values are
	// "solid" and "grid". Empty or unsupported values use "solid".
	Background string

	// DefaultSampleWindow controls the initial number of trend samples shown in
	// charts. Supported values are 30, 60, and 90. Zero or unsupported values use
	// 60.
	DefaultSampleWindow int

	// DiskPaths controls which filesystems are sampled for disk usage. When
	// empty, the current working directory's filesystem is sampled.
	DiskPaths []string

	// Refresh controls how often metrics are collected in the background.
	Refresh time.Duration

	// APIOnly makes Path return JSON even when the request does not ask for it.
	APIOnly bool

	// IgnoreRequest reports whether r should be excluded from HTTP request
	// counting. It does not stop the request from being served by the next
	// handler.
	//
	// Requests to Path are always excluded before IgnoreRequest is called.
	IgnoreRequest func(r *http.Request) bool
}

// DefaultConfig returns the default monitor configuration.
func DefaultConfig() Config {
	return Config{
		Path:                defaultPath,
		Title:               defaultTitle,
		Description:         defaultDescription,
		Footer:              defaultFooter,
		DefaultLanguage:     defaultLanguage,
		DefaultTheme:        defaultTheme,
		Background:          defaultBackground,
		DefaultSampleWindow: defaultSampleWindow,
		Refresh:             defaultRefresh,
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
	if cfg.Description == "" {
		cfg.Description = defaultDescription
	}
	if cfg.Footer == "" {
		cfg.Footer = defaultFooter
	}
	if !isSupportedLanguage(cfg.DefaultLanguage) {
		cfg.DefaultLanguage = defaultLanguage
	}
	if !isSupportedTheme(cfg.DefaultTheme) {
		cfg.DefaultTheme = defaultTheme
	}
	if !isSupportedBackground(cfg.Background) {
		cfg.Background = defaultBackground
	}
	if !isSupportedSampleWindow(cfg.DefaultSampleWindow) {
		cfg.DefaultSampleWindow = defaultSampleWindow
	}
	if len(cfg.DiskPaths) > 0 {
		cfg.DiskPaths = append([]string(nil), cfg.DiskPaths...)
	}
	if cfg.Refresh <= 0 {
		cfg.Refresh = defaultRefresh
	}
	return cfg
}

func isSupportedLanguage(lang string) bool {
	return lang == "en" || lang == "zh-CN"
}

func isSupportedTheme(theme string) bool {
	return theme == "light" || theme == "dark"
}

func isSupportedBackground(background string) bool {
	return background == "solid" || background == "grid"
}

func isSupportedSampleWindow(samples int) bool {
	return samples == 30 || samples == 60 || samples == 90
}
