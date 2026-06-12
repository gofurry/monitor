    const refreshMS = monitorConfig.refreshMS;
    const defaultLanguage = monitorConfig.defaultLanguage;
    const defaultSampleWindow = monitorConfig.defaultSampleWindow;
    const maxPoints = 90;
    const metricsPerPage = 5;
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
        inFlight: "In-flight",
        recentLatency: "Recent latency",
        maxLatency: "Max latency",
        status2xx: "2xx",
        status3xx: "3xx",
        status4xx: "4xx",
        status5xx: "5xx",
        trends: "Trends",
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
        inFlight: "处理中",
        recentLatency: "近期延迟",
        maxLatency: "最大延迟",
        status2xx: "2xx",
        status3xx: "3xx",
        status4xx: "4xx",
        status5xx: "5xx",
        trends: "趋势",
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
    let currentLang = defaultLanguage;
    let currentStatus = "live";
    let currentSampleWindow = defaultSampleWindow;
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
      return defaultLanguage;
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
    function applySampleWindow(value) {
      const next = Number(value);
      currentSampleWindow = [30, 60, 90].indexOf(next) >= 0 ? next : 60;
      document.querySelectorAll(".sample-option").forEach(function(button) {
        button.setAttribute("aria-pressed", button.dataset.samples === String(currentSampleWindow));
      });
      renderCharts();
    }
    function visibleSamples(data) {
      return data.slice(Math.max(0, data.length - currentSampleWindow));
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
    function formatAxis(v, unit) {
      const value = formatShort(v);
      return unit ? value + " " + unit : value;
    }
    function durationNS(v) {
      const n = Number(v || 0);
      if (n >= 1000000000) return (n / 1000000000).toFixed(2) + " s";
      if (n >= 1000000) return (n / 1000000).toFixed(2) + " ms";
      if (n >= 1000) return (n / 1000).toFixed(1) + " us";
      if (n > 0) return Math.max(1, Math.round(n)) + " ns";
      return "0 ns";
    }
    function initMetricPagination() {
      document.querySelectorAll(".metric-card").forEach(function(card) {
        const rows = Array.from(card.querySelectorAll("dl > .row"));
        const pager = card.querySelector(".metric-pager");
        if (!pager || rows.length <= metricsPerPage) return;

        const totalPages = Math.ceil(rows.length / metricsPerPage);
        const prev = pager.querySelector(".metric-pager__prev");
        const next = pager.querySelector(".metric-pager__next");
        const status = pager.querySelector(".metric-pager__status");
        let page = 0;

        function renderPage() {
          rows.forEach(function(row, index) {
            row.hidden = index < page * metricsPerPage || index >= (page + 1) * metricsPerPage;
          });
          status.textContent = (page + 1) + " / " + totalPages;
        }

        prev.addEventListener("click", function() {
          page = (page + totalPages - 1) % totalPages;
          renderPage();
        });
        next.addEventListener("click", function() {
          page = (page + 1) % totalPages;
          renderPage();
        });

        pager.hidden = false;
        renderPage();
      });
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
      $("http-in-flight").textContent = nf.format(data.http.in_flight_requests || 0);
      $("http-latency-recent").textContent = durationNS(data.http.latency && data.http.latency.recent_ns);
      $("http-latency-max").textContent = durationNS(data.http.latency && data.http.latency.max_ns);
      $("http-status-2xx").textContent = nf.format((data.http.status_codes && data.http.status_codes["2xx"]) || 0);
      $("http-status-3xx").textContent = nf.format((data.http.status_codes && data.http.status_codes["3xx"]) || 0);
      $("http-status-4xx").textContent = nf.format((data.http.status_codes && data.http.status_codes["4xx"]) || 0);
      $("http-status-5xx").textContent = nf.format((data.http.status_codes && data.http.status_codes["5xx"]) || 0);
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
    function drawGrid(ctx, width, height, padding, min, max, options) {
      const border = cssVar("--border");
      const muted = cssVar("--muted");
      const lines = 5;
      const plotH = height - padding.top - padding.bottom;
      ctx.strokeStyle = border;
      ctx.lineWidth = 1;
      ctx.globalAlpha = 0.55;
      for (let i = 0; i <= lines; i++) {
        const y = padding.top + (plotH / lines) * i;
        ctx.beginPath();
        ctx.moveTo(padding.left, y);
        ctx.lineTo(width - padding.right, y);
        ctx.stroke();
      }
      ctx.globalAlpha = 1;
      ctx.fillStyle = muted;
      ctx.font = "11px system-ui, sans-serif";
      ctx.textBaseline = "middle";
      for (let i = 0; i <= lines; i++) {
        const value = max - ((max - min) / lines) * i;
        const y = padding.top + (plotH / lines) * i;
        ctx.fillText(formatAxis(value, options.unit), 4, y);
      }
      ctx.textBaseline = "alphabetic";
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
      const padding = { top: 12, right: 10, bottom: 18, left: 58 };
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
      drawGrid(ctx, width, height, padding, min, max, options || {});
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
        { data: visibleSamples(history.pidCPU), color: accent },
        { data: visibleSamples(history.osCPU), color: warn }
      ], { minValue: 0, minMax: 100, unit: "%" });
      drawLineChart($("memory-chart"), [
        { data: visibleSamples(history.rssMiB), color: accent },
        { data: visibleSamples(history.heapMiB), color: good }
      ], { minValue: 0, unit: "MiB" });
      drawLineChart($("goroutine-chart"), [
        { data: visibleSamples(history.goroutines), color: accent }
      ], { minValue: 0, unit: "g" });
      drawLineChart($("request-chart"), [
        { data: visibleSamples(history.requestsDelta), color: accent }
      ], { minValue: 0, unit: "req" });
    }
    function updateScrollOrb() {
      const dock = $("page-scroll-dock");
      const value = $("page-scroll-dock-value");
      if (!dock || !value) return;
      const doc = document.documentElement;
      const maxScroll = Math.max(0, doc.scrollHeight - window.innerHeight);
      const top = window.scrollY || doc.scrollTop || document.body.scrollTop || 0;
      const progress = maxScroll ? Math.min(100, Math.max(0, (top / maxScroll) * 100)) : 0;
      const rounded = Math.round(progress);
      const shouldShow = window.innerWidth >= 768 && maxScroll > 320 && top > 72;
      dock.style.setProperty("--scroll-progress", rounded + "%");
      dock.setAttribute("aria-valuenow", String(rounded));
      dock.setAttribute("aria-label", "Scroll progress " + rounded + "%");
      dock.classList.toggle("page-scroll-dock--visible", shouldShow);
      value.textContent = rounded + "%";
    }
    function scrollUpQuarter() {
      const doc = document.documentElement;
      const maxScroll = Math.max(0, doc.scrollHeight - window.innerHeight);
      const top = window.scrollY || doc.scrollTop || document.body.scrollTop || 0;
      window.scrollTo({ top: Math.max(0, top - maxScroll * 0.25), behavior: "smooth" });
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
    $("page-scroll-dock").addEventListener("click", scrollUpQuarter);
    initMetricPagination();
    document.querySelectorAll(".sample-option").forEach(function(button) {
      button.addEventListener("click", function() {
        applySampleWindow(button.dataset.samples);
      });
    });
    if (window.matchMedia) {
      const themeQuery = window.matchMedia("(prefers-color-scheme: dark)");
      const onThemeChange = function() {
        if (currentThemeMode === "auto") applyTheme("auto", false);
      };
      if (themeQuery.addEventListener) themeQuery.addEventListener("change", onThemeChange);
      else if (themeQuery.addListener) themeQuery.addListener(onThemeChange);
    }
    window.addEventListener("scroll", updateScrollOrb, { passive: true });
    window.addEventListener("resize", function() {
      renderCharts();
      updateScrollOrb();
    });
    setInterval(function() {
      if (!lastSuccessAt || currentStatus === "error") return;
      if (Date.now() - lastSuccessAt > refreshMS * 3) setStatus("stale");
    }, 1000);

    currentLang = detectLang();
    applyTheme(storageGet("monitor.theme") || "auto", false);
    applySampleWindow(defaultSampleWindow);
    applyLang(currentLang);
    updateScrollOrb();
    refresh();
    setInterval(refresh, refreshMS);
