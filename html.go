package monitor

import (
	"html"
	"strconv"
	"strings"
	"time"
)

func renderHTML(cfg Config) string {
	title := html.EscapeString(cfg.Title)
	refreshMS := strconv.FormatInt(maxInt64(int64(cfg.Refresh/time.Millisecond), 250), 10)

	page := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{TITLE}}</title>
  <style>
    :root {
      color-scheme: light dark;
      --bg: #f7f7f4;
      --fg: #202124;
      --muted: #686b6f;
      --line: #d9d8d2;
      --panel: #ffffff;
      --accent: #176f7a;
      --good: #16844a;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #111315;
        --fg: #eff1f3;
        --muted: #a6abb1;
        --line: #2f3338;
        --panel: #181b1f;
        --accent: #61c3d0;
        --good: #60d394;
      }
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: var(--bg);
      color: var(--fg);
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      line-height: 1.45;
    }
    main {
      width: min(980px, calc(100% - 32px));
      margin: 0 auto;
      padding: 40px 0;
    }
    header {
      display: flex;
      align-items: flex-end;
      justify-content: space-between;
      gap: 20px;
      padding-bottom: 22px;
      border-bottom: 1px solid var(--line);
    }
    h1 {
      margin: 0 0 6px;
      font-size: clamp(2rem, 4vw, 3.25rem);
      line-height: 1;
      letter-spacing: 0;
    }
    .subtitle, .meta {
      color: var(--muted);
      font-size: 0.95rem;
    }
    .status {
      color: var(--good);
      font-weight: 700;
      white-space: nowrap;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 14px;
      margin-top: 22px;
    }
    section {
      min-width: 0;
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 16px;
    }
    h2 {
      margin: 0 0 14px;
      color: var(--accent);
      font-size: 0.82rem;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    dl {
      display: grid;
      gap: 12px;
      margin: 0;
    }
    .row {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: 14px;
      min-height: 28px;
    }
    dt {
      color: var(--muted);
      font-size: 0.9rem;
    }
    dd {
      margin: 0;
      overflow-wrap: anywhere;
      text-align: right;
      font-variant-numeric: tabular-nums;
      font-weight: 700;
    }
    @media (max-width: 820px) {
      header {
        display: block;
      }
      .meta {
        margin-top: 14px;
      }
      .grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }
    }
    @media (max-width: 560px) {
      main {
        width: min(100% - 24px, 980px);
        padding: 24px 0;
      }
      .grid {
        grid-template-columns: 1fr;
      }
    }
  </style>
</head>
<body>
  <main>
    <header>
      <div>
        <h1>{{TITLE}}</h1>
        <div class="subtitle">Real-time Go service status</div>
      </div>
      <div class="meta">
        <div><span class="status" id="status">LIVE</span></div>
        <div>Updated: <span id="updated">-</span></div>
        <div>Response: <span id="response">-</span></div>
      </div>
    </header>

    <div class="grid">
      <section>
        <h2>Process</h2>
        <dl>
          <div class="row"><dt>CPU</dt><dd id="pid-cpu">-</dd></div>
          <div class="row"><dt>RSS</dt><dd id="pid-rss">-</dd></div>
        </dl>
      </section>
      <section>
        <h2>Runtime</h2>
        <dl>
          <div class="row"><dt>Goroutines</dt><dd id="rt-goroutines">-</dd></div>
          <div class="row"><dt>Heap Alloc</dt><dd id="rt-heap-alloc">-</dd></div>
          <div class="row"><dt>Heap Sys</dt><dd id="rt-heap-sys">-</dd></div>
          <div class="row"><dt>GC Count</dt><dd id="rt-gc">-</dd></div>
          <div class="row"><dt>Uptime</dt><dd id="rt-uptime">-</dd></div>
        </dl>
      </section>
      <section>
        <h2>System</h2>
        <dl>
          <div class="row"><dt>CPU</dt><dd id="os-cpu">-</dd></div>
          <div class="row"><dt>Memory</dt><dd id="os-memory">-</dd></div>
          <div class="row"><dt>Total RAM</dt><dd id="os-total">-</dd></div>
          <div class="row"><dt>Load1</dt><dd id="os-load">-</dd></div>
        </dl>
      </section>
      <section>
        <h2>HTTP</h2>
        <dl>
          <div class="row"><dt>Requests</dt><dd id="http-requests">-</dd></div>
        </dl>
      </section>
    </div>
  </main>

  <script>
    const refreshMS = {{REFRESH_MS}};
    const $ = (id) => document.getElementById(id);
    const nf = new Intl.NumberFormat();

    function pct(v) {
      return Number(v || 0).toFixed(1) + "%";
    }
    function bytes(v) {
      const units = ["B", "KiB", "MiB", "GiB", "TiB"];
      let n = Number(v || 0);
      let i = 0;
      while (n >= 1024 && i < units.length - 1) {
        n /= 1024;
        i++;
      }
      return n.toFixed(i === 0 ? 0 : 1) + " " + units[i];
    }
    function uptime(seconds) {
      let s = Math.max(0, Number(seconds || 0));
      const d = Math.floor(s / 86400); s %= 86400;
      const h = Math.floor(s / 3600); s %= 3600;
      const m = Math.floor(s / 60);
      const parts = [];
      if (d) parts.push(d + "d");
      if (h) parts.push(h + "h");
      if (m) parts.push(m + "m");
      parts.push(Math.floor(s % 60) + "s");
      return parts.join(" ");
    }
    async function refresh() {
      const started = performance.now();
      try {
        const res = await fetch(location.pathname, { headers: { Accept: "application/json" } });
        const data = await res.json();
        $("pid-cpu").textContent = pct(data.pid.cpu_percent);
        $("pid-rss").textContent = bytes(data.pid.rss_bytes);
        $("rt-goroutines").textContent = nf.format(data.runtime.goroutines || 0);
        $("rt-heap-alloc").textContent = bytes(data.runtime.heap_alloc_bytes);
        $("rt-heap-sys").textContent = bytes(data.runtime.heap_sys_bytes);
        $("rt-gc").textContent = nf.format(data.runtime.num_gc || 0);
        $("rt-uptime").textContent = uptime(data.runtime.uptime_seconds);
        $("os-cpu").textContent = pct(data.os.cpu_percent);
        $("os-memory").textContent = pct(data.os.memory_used_percent);
        $("os-total").textContent = bytes(data.os.memory_total_bytes);
        $("os-load").textContent = Number(data.os.load1 || 0).toFixed(2);
        $("http-requests").textContent = nf.format(data.http.total_requests || 0);
        $("updated").textContent = new Date().toLocaleString();
        $("response").textContent = (performance.now() - started).toFixed(1) + " ms";
        $("status").textContent = "LIVE";
      } catch (err) {
        $("status").textContent = "OFFLINE";
      }
    }
    refresh();
    setInterval(refresh, refreshMS);
  </script>
</body>
</html>`

	page = strings.ReplaceAll(page, "{{TITLE}}", title)
	page = strings.ReplaceAll(page, "{{REFRESH_MS}}", refreshMS)
	return page
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
