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
<html lang="en" data-theme="light">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{TITLE}}</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f7f8fb;
      --panel: #ffffff;
      --panel-soft: #f1f5f9;
      --text: #111827;
      --muted: #64748b;
      --border: #d8dee9;
      --accent: #0891b2;
      --accent-soft: rgba(8, 145, 178, 0.12);
      --good: #16a34a;
      --warn: #d97706;
      --bad: #dc2626;
      --shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
    }
    [data-theme="dark"] {
      color-scheme: dark;
      --bg: #0d1117;
      --panel: #151b23;
      --panel-soft: #10161d;
      --text: #f3f4f6;
      --muted: #9ca3af;
      --border: #30363d;
      --accent: #67e8f9;
      --accent-soft: rgba(103, 232, 249, 0.12);
      --good: #4ade80;
      --warn: #facc15;
      --bad: #fb7185;
      --shadow: none;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: var(--bg);
      color: var(--text);
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      line-height: 1.45;
    }
    main {
      width: min(1120px, calc(100% - 32px));
      margin: 0 auto;
      padding: 20px 0 22px;
    }
    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 18px;
      padding-bottom: 12px;
      border-bottom: 1px solid var(--border);
    }
    .header-main {
      min-width: 0;
    }
    h1 {
      margin: 0;
      overflow-wrap: anywhere;
      font-size: clamp(1.7rem, 3.2vw, 2.7rem);
      line-height: 1.05;
      letter-spacing: 0;
      font-weight: 780;
    }
    .meta {
      color: var(--muted);
      font-size: 0.95rem;
    }
    .header-side {
      display: grid;
      justify-items: end;
      gap: 12px;
    }
    button {
      border: 1px solid var(--border);
      background: var(--panel);
      color: var(--text);
      cursor: pointer;
      font: inherit;
      transition: border-color 140ms ease, background 140ms ease, box-shadow 140ms ease, transform 140ms ease;
    }
    button:hover {
      border-color: var(--accent);
      transform: translateY(-1px);
    }
    .status-box {
      display: grid;
      gap: 5px;
      color: var(--muted);
      text-align: right;
      font-size: 0.92rem;
      font-variant-numeric: tabular-nums;
    }
    .status-line {
      display: flex;
      align-items: center;
      justify-content: flex-end;
      gap: 7px;
      color: var(--text);
      font-weight: 800;
    }
    .status-controls {
      display: inline-flex;
      align-items: center;
      gap: 7px;
      margin-right: 5px;
    }
    .control-dot {
      width: 18px;
      height: 18px;
      padding: 0;
      border-radius: 50%;
      background: var(--panel-soft);
      box-shadow: inset 0 0 0 4px color-mix(in srgb, var(--muted) 18%, transparent);
    }
    .control-dot[data-active="light"] {
      background: #f8fafc;
      box-shadow: inset 0 0 0 5px #facc15;
    }
    .control-dot[data-active="dark"] {
      background: #111827;
      box-shadow: inset 0 0 0 5px #67e8f9;
    }
    .control-dot[data-active="en"] {
      background: #0891b2;
      box-shadow: inset 0 0 0 5px color-mix(in srgb, #ffffff 32%, transparent);
    }
    .control-dot[data-active="zh-CN"] {
      background: #16a34a;
      box-shadow: inset 0 0 0 5px color-mix(in srgb, #ffffff 32%, transparent);
    }
    .visually-hidden {
      position: absolute;
      width: 1px;
      height: 1px;
      padding: 0;
      margin: -1px;
      overflow: hidden;
      clip: rect(0, 0, 0, 0);
      white-space: nowrap;
      border: 0;
    }
    #live-dot {
      width: 9px;
      height: 9px;
      border-radius: 50%;
      background: var(--good);
      box-shadow: 0 0 0 4px color-mix(in srgb, var(--good) 18%, transparent);
    }
    #live-dot[data-status="stale"] {
      background: var(--warn);
      box-shadow: 0 0 0 4px color-mix(in srgb, var(--warn) 18%, transparent);
    }
    #live-dot[data-status="error"] {
      background: var(--bad);
      box-shadow: 0 0 0 4px color-mix(in srgb, var(--bad) 18%, transparent);
    }
    .status-meta {
      display: flex;
      justify-content: flex-end;
      gap: 14px;
      white-space: nowrap;
    }
    .cards, .chart-grid {
      display: grid;
      gap: 14px;
    }
    .cards {
      grid-template-columns: repeat(4, minmax(0, 1fr));
      margin-top: 20px;
    }
    .chart-section {
      margin-top: 22px;
    }
    .section-title {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: 14px;
      margin-bottom: 12px;
    }
    .chart-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    section, .chart-card {
      min-width: 0;
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 8px;
      box-shadow: var(--shadow);
    }
    section {
      padding: 15px;
    }
    .chart-card {
      padding: 14px;
    }
    h2 {
      margin: 0 0 14px;
      color: var(--accent);
      font-size: 0.82rem;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    .section-title h2 {
      margin: 0;
    }
    dl {
      display: grid;
      gap: 11px;
      margin: 0;
    }
    .row {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: 14px;
      min-height: 27px;
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
      font-weight: 800;
    }
    .chart-head {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 9px;
      font-size: 0.92rem;
      font-weight: 800;
    }
    .chart-legend {
      color: var(--muted);
      font-size: 0.78rem;
      font-weight: 700;
      text-align: right;
    }
    canvas {
      display: block;
      width: 100%;
      height: 150px;
      border-radius: 6px;
      background: var(--panel-soft);
    }
    @media (max-width: 980px) {
      .cards {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }
    }
    @media (max-width: 760px) {
      main {
        width: min(100% - 24px, 1120px);
        padding-top: 22px;
      }
      .header {
        align-items: stretch;
        flex-direction: column;
      }
      .header-side, .status-box {
        justify-items: start;
        text-align: left;
      }
      .status-line, .status-meta {
        justify-content: flex-start;
      }
      .chart-grid {
        grid-template-columns: 1fr;
      }
    }
    @media (max-width: 560px) {
      .cards {
        grid-template-columns: 1fr;
      }
      .section-title {
        display: block;
      }
      .status-meta {
        flex-wrap: wrap;
        gap: 8px 12px;
      }
    }
  </style>
</head>
<body>
  <main>
    <header class="header">
      <div class="header-main">
        <h1>{{TITLE}}</h1>
      </div>
      <div class="header-side">
        <div class="status-box">
          <div class="status-line">
            <span class="status-controls" aria-label="Display preferences">
              <button type="button" id="lang-toggle" class="control-dot" title="Language"><span class="visually-hidden">Language</span></button>
              <button type="button" id="theme-toggle" class="control-dot" title="Theme"><span class="visually-hidden">Theme</span></button>
            </span>
            <span id="live-dot" data-status="live"></span>
            <span id="live-text" data-i18n="live">LIVE</span>
          </div>
          <div class="status-meta"><span id="updated-at">-</span><span id="response-time">-</span></div>
        </div>
      </div>
    </header>

    <div class="cards">
      <section>
        <h2 data-i18n="process">Process</h2>
        <dl>
          <div class="row"><dt data-i18n="cpu">CPU</dt><dd id="pid-cpu">-</dd></div>
          <div class="row"><dt data-i18n="rss">RSS</dt><dd id="pid-rss">-</dd></div>
        </dl>
      </section>
      <section>
        <h2 data-i18n="runtime">Runtime</h2>
        <dl>
          <div class="row"><dt data-i18n="goroutines">Goroutines</dt><dd id="rt-goroutines">-</dd></div>
          <div class="row"><dt data-i18n="heapAlloc">Heap Alloc</dt><dd id="rt-heap-alloc">-</dd></div>
          <div class="row"><dt data-i18n="heapSys">Heap Sys</dt><dd id="rt-heap-sys">-</dd></div>
          <div class="row"><dt data-i18n="gcCount">GC Count</dt><dd id="rt-gc">-</dd></div>
          <div class="row"><dt data-i18n="uptime">Uptime</dt><dd id="rt-uptime">-</dd></div>
        </dl>
      </section>
      <section>
        <h2 data-i18n="system">System</h2>
        <dl>
          <div class="row"><dt data-i18n="cpu">CPU</dt><dd id="os-cpu">-</dd></div>
          <div class="row"><dt data-i18n="memory">Memory</dt><dd id="os-memory">-</dd></div>
          <div class="row"><dt data-i18n="totalRam">Total RAM</dt><dd id="os-total">-</dd></div>
          <div class="row"><dt data-i18n="load1">Load1</dt><dd id="os-load">-</dd></div>
        </dl>
      </section>
      <section>
        <h2 data-i18n="http">HTTP</h2>
        <dl>
          <div class="row"><dt data-i18n="requests">Requests</dt><dd id="http-requests">-</dd></div>
        </dl>
      </section>
    </div>

    <div class="chart-section">
      <div class="section-title">
        <h2 data-i18n="trends">Trends</h2>
        <div class="meta" data-i18n="chartWindow">Last 60 samples</div>
      </div>
      <div class="chart-grid">
        <article class="chart-card">
          <div class="chart-head">
            <span data-i18n="cpuTrend">CPU</span>
            <span class="chart-legend">PID / OS</span>
          </div>
          <canvas id="cpu-chart"></canvas>
        </article>
        <article class="chart-card">
          <div class="chart-head">
            <span data-i18n="memoryTrend">Memory</span>
            <span class="chart-legend">RSS / Heap</span>
          </div>
          <canvas id="memory-chart"></canvas>
        </article>
        <article class="chart-card">
          <div class="chart-head">
            <span data-i18n="goroutineTrend">Goroutines</span>
          </div>
          <canvas id="goroutine-chart"></canvas>
        </article>
        <article class="chart-card">
          <div class="chart-head">
            <span data-i18n="requestTrend">Requests / interval</span>
          </div>
          <canvas id="request-chart"></canvas>
        </article>
      </div>
    </div>
  </main>

  <script>
    const refreshMS = {{REFRESH_MS}};
    const maxPoints = 60;
    const $ = (id) => document.getElementById(id);
    const nf = new Intl.NumberFormat();
    const messages = {
      en: {
        live: "LIVE",
        stale: "STALE",
        error: "ERROR",
        process: "Process",
        runtime: "Runtime",
        system: "System",
        http: "HTTP",
        cpu: "CPU",
        rss: "RSS",
        goroutines: "Goroutines",
        heapAlloc: "Heap Alloc",
        heapSys: "Heap Sys",
        gcCount: "GC Count",
        uptime: "Uptime",
        memory: "Memory",
        totalRam: "Total RAM",
        load1: "Load1",
        requests: "Requests",
        trends: "Trends",
        chartWindow: "Last 60 samples",
        cpuTrend: "CPU",
        memoryTrend: "Memory",
        goroutineTrend: "Goroutines",
        requestTrend: "Requests / interval"
      },
      "zh-CN": {
        live: "运行中",
        stale: "已延迟",
        error: "错误",
        process: "进程",
        runtime: "运行时",
        system: "系统",
        http: "HTTP",
        cpu: "CPU",
        rss: "RSS",
        goroutines: "Goroutine",
        heapAlloc: "堆分配",
        heapSys: "堆系统",
        gcCount: "GC 次数",
        uptime: "运行时间",
        memory: "内存",
        totalRam: "总内存",
        load1: "Load1",
        requests: "请求数",
        trends: "趋势",
        chartWindow: "最近 60 个采样点",
        cpuTrend: "CPU",
        memoryTrend: "内存",
        goroutineTrend: "Goroutine",
        requestTrend: "区间请求数"
      }
    };
    const languages = ["en", "zh-CN"];
    const history = {
      labels: [],
      pidCPU: [],
      osCPU: [],
      rssMiB: [],
      heapMiB: [],
      goroutines: [],
      requestsDelta: []
    };
    let previousSnapshot = null;
    let currentThemeMode = "auto";
    let currentLang = "en";
    let currentStatus = "live";
    let lastSuccessAt = 0;

    function storageGet(key) {
      try {
        return localStorage.getItem(key);
      } catch (err) {
        return "";
      }
    }
    function storageSet(key, value) {
      try {
        localStorage.setItem(key, value);
      } catch (err) {}
    }
    function detectLang() {
      const saved = storageGet("monitor.lang");
      if (saved === "en" || saved === "zh-CN") return saved;
      return navigator.language && navigator.language.indexOf("zh") === 0 ? "zh-CN" : "en";
    }
    function t(key) {
      return (messages[currentLang] && messages[currentLang][key]) || messages.en[key] || key;
    }
    function applyLang(lang) {
      currentLang = lang === "zh-CN" ? "zh-CN" : "en";
      storageSet("monitor.lang", currentLang);
      document.documentElement.lang = currentLang;
      document.querySelectorAll("[data-i18n]").forEach(function(el) {
        el.textContent = t(el.dataset.i18n);
      });
      $("lang-toggle").dataset.active = currentLang;
      setStatus(currentStatus);
    }
    function resolveTheme(mode) {
      if (mode === "light" || mode === "dark") return mode;
      return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
    }
    function applyTheme(mode, persist) {
      if (mode !== "light" && mode !== "dark") mode = "auto";
      const resolved = resolveTheme(mode);
      currentThemeMode = mode;
      document.documentElement.dataset.theme = resolved;
      if (persist !== false) storageSet("monitor.theme", mode);
      $("theme-toggle").dataset.active = resolved;
      renderCharts();
    }
    function nextTheme() {
      applyTheme(resolveTheme(currentThemeMode) === "dark" ? "light" : "dark");
    }
    function nextLang() {
      const index = languages.indexOf(currentLang);
      const next = languages[(index + 1) % languages.length] || "en";
      applyLang(next);
    }
    function setStatus(status) {
      currentStatus = status;
      $("live-text").textContent = t(status);
      $("live-dot").dataset.status = status;
    }
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
    function bytesToMiB(v) {
      return Number(v || 0) / 1024 / 1024;
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
    function formatShort(v) {
      const n = Number(v || 0);
      if (Math.abs(n) >= 1000000) return (n / 1000000).toFixed(1) + "M";
      if (Math.abs(n) >= 1000) return (n / 1000).toFixed(1) + "k";
      if (Math.abs(n) >= 10) return n.toFixed(0);
      return n.toFixed(1);
    }
    function cssVar(name) {
      return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    }
    function renderSnapshot(data, elapsed) {
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
      $("updated-at").textContent = new Date().toLocaleString();
      $("response-time").textContent = elapsed.toFixed(1) + " ms";
    }
    function pushHistory(snapshot) {
      const requests = snapshot.http.total_requests || 0;
      const previousRequests = previousSnapshot ? previousSnapshot.http.total_requests || 0 : requests;
      const delta = Math.max(0, requests - previousRequests);

      history.labels.push(new Date().toLocaleTimeString());
      history.pidCPU.push(snapshot.pid.cpu_percent || 0);
      history.osCPU.push(snapshot.os.cpu_percent || 0);
      history.rssMiB.push(bytesToMiB(snapshot.pid.rss_bytes || 0));
      history.heapMiB.push(bytesToMiB(snapshot.runtime.heap_alloc_bytes || 0));
      history.goroutines.push(snapshot.runtime.goroutines || 0);
      history.requestsDelta.push(delta);
      Object.keys(history).forEach(function(key) {
        while (history[key].length > maxPoints) history[key].shift();
      });
      previousSnapshot = snapshot;
    }
    function drawGrid(ctx, width, height, padding, min, max) {
      const border = cssVar("--border");
      const muted = cssVar("--muted");
      const plotH = height - padding.top - padding.bottom;
      ctx.strokeStyle = border;
      ctx.lineWidth = 1;
      ctx.globalAlpha = 0.55;
      for (let i = 0; i <= 3; i++) {
        const y = padding.top + (plotH / 3) * i;
        ctx.beginPath();
        ctx.moveTo(padding.left, y);
        ctx.lineTo(width - padding.right, y);
        ctx.stroke();
      }
      ctx.globalAlpha = 1;
      ctx.fillStyle = muted;
      ctx.font = "11px system-ui, sans-serif";
      ctx.fillText(formatShort(max), 4, padding.top + 4);
      ctx.fillText(formatShort(min), 4, height - padding.bottom);
    }
    function drawSeries(ctx, data, cfg) {
      const width = cfg.width;
      const height = cfg.height;
      const padding = cfg.padding;
      const min = cfg.min;
      const max = cfg.max;
      const plotW = width - padding.left - padding.right;
      const plotH = height - padding.top - padding.bottom;
      if (!data.length) return;
      ctx.beginPath();
      ctx.lineWidth = 2;
      ctx.strokeStyle = cfg.color;
      data.forEach(function(value, i) {
        const x = padding.left + (data.length === 1 ? plotW : (i / (data.length - 1)) * plotW);
        const y = padding.top + (1 - (value - min) / (max - min || 1)) * plotH;
        if (i === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      });
      ctx.stroke();
    }
    function drawLineChart(canvas, seriesList, options) {
      if (!canvas) return;
      const ctx = canvas.getContext("2d");
      const dpr = window.devicePixelRatio || 1;
      const rect = canvas.getBoundingClientRect();
      const width = Math.max(rect.width, 1);
      const height = Math.max(rect.height, 1);
      const padding = { top: 12, right: 10, bottom: 18, left: 32 };
      canvas.width = Math.floor(width * dpr);
      canvas.height = Math.floor(height * dpr);
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.clearRect(0, 0, width, height);
      const values = [];
      seriesList.forEach(function(series) {
        series.data.forEach(function(value) {
          if (Number.isFinite(value)) values.push(value);
        });
      });
      const min = options && Number.isFinite(options.minValue) ? options.minValue : 0;
      const floorMax = options && Number.isFinite(options.minMax) ? options.minMax : 0;
      const max = Math.max(floorMax, values.length ? Math.max.apply(null, values) : 1, 1);
      drawGrid(ctx, width, height, padding, min, max);
      seriesList.forEach(function(series) {
        drawSeries(ctx, series.data, {
          width: width,
          height: height,
          padding: padding,
          min: min,
          max: max,
          color: series.color
        });
      });
    }
    function renderCharts() {
      const accent = cssVar("--accent");
      const good = cssVar("--good");
      const warn = cssVar("--warn");
      drawLineChart($("cpu-chart"), [
        { data: history.pidCPU, color: accent },
        { data: history.osCPU, color: warn }
      ], { minValue: 0, minMax: 100 });
      drawLineChart($("memory-chart"), [
        { data: history.rssMiB, color: accent },
        { data: history.heapMiB, color: good }
      ], { minValue: 0 });
      drawLineChart($("goroutine-chart"), [
        { data: history.goroutines, color: accent }
      ], { minValue: 0 });
      drawLineChart($("request-chart"), [
        { data: history.requestsDelta, color: accent }
      ], { minValue: 0 });
    }
    async function refresh() {
      const started = performance.now();
      try {
        const res = await fetch(location.href, {
          headers: { Accept: "application/json" },
          cache: "no-store"
        });
        if (!res.ok) throw new Error("bad status: " + res.status);
        const data = await res.json();
        renderSnapshot(data, performance.now() - started);
        pushHistory(data);
        renderCharts();
        lastSuccessAt = Date.now();
        setStatus("live");
      } catch (err) {
        setStatus("error");
      }
    }

    $("lang-toggle").addEventListener("click", nextLang);
    $("theme-toggle").addEventListener("click", nextTheme);
    if (window.matchMedia) {
      const themeQuery = window.matchMedia("(prefers-color-scheme: dark)");
      const onThemeChange = function() {
        if (currentThemeMode === "auto") applyTheme("auto", false);
      };
      if (themeQuery.addEventListener) themeQuery.addEventListener("change", onThemeChange);
      else if (themeQuery.addListener) themeQuery.addListener(onThemeChange);
    }
    window.addEventListener("resize", renderCharts);
    setInterval(function() {
      if (!lastSuccessAt || currentStatus === "error") return;
      if (Date.now() - lastSuccessAt > refreshMS * 3) setStatus("stale");
    }, 1000);

    currentLang = detectLang();
    applyTheme(storageGet("monitor.theme") || "auto", false);
    applyLang(currentLang);
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
