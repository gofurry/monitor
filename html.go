package monitor

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"
	"time"
)

//go:embed internal/ui/page.html
var monitorPageHTML string

//go:embed internal/ui/style.css
var monitorStyleCSS string

//go:embed internal/ui/app.js
var monitorAppJS string

var monitorPageTemplate = template.Must(template.New("monitor").Parse(monitorPageHTML))

type monitorPageData struct {
	Title            string
	Description      string
	Footer           string
	CSS              template.CSS
	JS               template.JS
	ConfigJSON       template.JS
	DefaultTheme     string
	Background       string
	Samples30Pressed bool
	Samples60Pressed bool
	Samples90Pressed bool
}

type monitorClientConfig struct {
	RefreshMS           int64  `json:"refreshMS"`
	DefaultLanguage     string `json:"defaultLanguage"`
	DefaultTheme        string `json:"defaultTheme"`
	DefaultSampleWindow int    `json:"defaultSampleWindow"`
}

func renderHTML(cfg Config) string {
	refreshMS := maxInt64(int64(cfg.Refresh/time.Millisecond), 250)
	configJSON, _ := json.Marshal(monitorClientConfig{
		RefreshMS:           refreshMS,
		DefaultLanguage:     cfg.DefaultLanguage,
		DefaultTheme:        cfg.DefaultTheme,
		DefaultSampleWindow: cfg.DefaultSampleWindow,
	})

	data := monitorPageData{
		Title:            cfg.Title,
		Description:      cfg.Description,
		Footer:           cfg.Footer,
		CSS:              template.CSS(monitorStyleCSS),
		JS:               template.JS(monitorAppJS),
		ConfigJSON:       template.JS(configJSON),
		DefaultTheme:     cfg.DefaultTheme,
		Background:       cfg.Background,
		Samples30Pressed: cfg.DefaultSampleWindow == 30,
		Samples60Pressed: cfg.DefaultSampleWindow == 60,
		Samples90Pressed: cfg.DefaultSampleWindow == 90,
	}

	var buf bytes.Buffer
	if err := monitorPageTemplate.Execute(&buf, data); err != nil {
		return ""
	}
	return buf.String()
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
