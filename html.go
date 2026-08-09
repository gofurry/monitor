package monitor

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"time"
)

const defaultMonitorFaviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
<rect width="64" height="64" rx="14" fill="#0f172a"/>
<path d="M8 34h13l6-17 11 32 7-20 4 5h7" fill="none" stroke="#67e8f9" stroke-width="6" stroke-linecap="round" stroke-linejoin="round"/>
</svg>`

var defaultMonitorFaviconURL = "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(defaultMonitorFaviconSVG))

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
	FaviconURL       any
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
	var faviconURL any = cfg.FaviconURL
	if cfg.FaviconURL == "" {
		faviconURL = template.URL(defaultMonitorFaviconURL) // #nosec G203 -- this is the package-owned embedded favicon.
	}

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
		FaviconURL:       faviconURL,
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
