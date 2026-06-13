    const refreshMS = monitorConfig.refreshMS;
    const defaultLanguage = monitorConfig.defaultLanguage;
    const defaultTheme = monitorConfig.defaultTheme;
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
        pid: "PID",
        threads: "Threads",
        fds: "File handles",
        goroutines: "Goroutine / Peak",
        goroutinePeak: "Goroutine Peak",
        heap: "Heap",
        heapAlloc: "Heap Alloc",
        heapSys: "Heap Sys",
        heapObjects: "Heap Objects",
        nextGC: "Next GC",
        mallocs: "Mallocs",
        frees: "Frees",
        gcCount: "GC Count",
        gc: "GC",
        gcPauseLast: "GC pause last",
        gcPauseRecent: "GC pause window",
        gcPauseTotal: "GC pause total",
        heapDetailsTitle: "Heap Details",
        gcDetailsTitle: "GC Details",
        uptime: "Uptime",
        memory: "Memory",
        totalRam: "Total RAM",
        disk: "Disk",
        diskUsage: "Disk Usage",
        diskUsed: "Disk Used",
        diskTotal: "Disk Total",
        diskFree: "Free",
        diskDetails: "Details",
        diskDetailsTitle: "Disk Details",
        diskNoData: "No disk data",
        diskDevice: "Device",
        diskType: "Type",
        load1: "1m load",
        requests: "Requests",
        inFlight: "In-flight",
        recentLatency: "Recent latency",
        maxLatency: "Max latency",
        statusCodes: "Status codes",
        statusCodesTitle: "HTTP Status Codes",
        statusTotal: "Total",
        status2xx: "2xx",
        status3xx: "3xx",
        status4xx: "4xx",
        status5xx: "5xx",
        trends: "Trends",
        cpuTrend: "CPU",
        memoryTrend: "Memory",
        heapGCTrend: "Heap / Next GC",
        goroutineTrend: "Goroutines",
        requestTrend: "Requests / interval",
        latencyTrend: "HTTP latency",
        inFlightTrend: "In-flight",
        gcPauseTrend: "GC pause",
        recent: "Recent",
        max: "Max",
        active: "Active",
        window: "Window",
        last: "Last"
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
        pid: "PID",
        threads: "线程",
        fds: "文件句柄",
        goroutines: "Goroutine / 峰值",
        goroutinePeak: "Goroutine 峰值",
        heap: "堆",
        heapAlloc: "堆分配",
        heapSys: "堆系统",
        heapObjects: "堆对象",
        nextGC: "下次 GC",
        mallocs: "分配次数",
        frees: "释放次数",
        gcCount: "GC 次数",
        gc: "GC",
        gcPauseLast: "最近 GC 暂停",
        gcPauseRecent: "窗口 GC 暂停",
        gcPauseTotal: "累计 GC 暂停",
        heapDetailsTitle: "堆详情",
        gcDetailsTitle: "GC 详情",
        uptime: "运行时间",
        memory: "内存",
        totalRam: "总内存",
        disk: "磁盘",
        diskUsage: "磁盘使用率",
        diskUsed: "磁盘已用",
        diskTotal: "磁盘总量",
        diskFree: "空闲",
        diskDetails: "详情",
        diskDetailsTitle: "磁盘详情",
        diskNoData: "暂无磁盘数据",
        diskDevice: "设备",
        diskType: "类型",
        load1: "1分钟负载",
        requests: "请求数",
        inFlight: "处理中",
        recentLatency: "近期延迟",
        maxLatency: "最大延迟",
        statusCodes: "状态码",
        statusCodesTitle: "HTTP 状态码",
        statusTotal: "合计",
        status2xx: "2xx",
        status3xx: "3xx",
        status4xx: "4xx",
        status5xx: "5xx",
        trends: "趋势",
        cpuTrend: "CPU",
        memoryTrend: "内存",
        heapGCTrend: "堆 / 下次 GC",
        goroutineTrend: "Goroutine",
        requestTrend: "区间请求数",
        latencyTrend: "HTTP 延迟",
        inFlightTrend: "处理中请求",
        gcPauseTrend: "GC 暂停",
        recent: "近期",
        max: "最大",
        active: "活跃",
        window: "窗口",
        last: "最近"
      }
    };
    const languages = ["en", "zh-CN"];
    const history = {
      labels: [],
      pidCPU: [],
      osCPU: [],
      rssMiB: [],
      heapMiB: [],
      nextGCMiB: [],
      goroutines: [],
      requestsDelta: [],
      latencyRecentNS: [],
      latencyMaxNS: [],
      inFlight: [],
      gcPauseRecentNS: [],
      gcPauseLastNS: []
    };
    let previousSnapshot = null;
    let currentThemeMode = defaultTheme;
    let currentLang = defaultLanguage;
    let currentStatus = "live";
    let currentSampleWindow = defaultSampleWindow;
    let currentDisks = [];
    let currentRuntime = {};
    let currentHTTPStatusCodes = {};
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
      updateDiskUI();
      updateRuntimeDetailUI();
      updateHTTPStatusUI();
    }
    function resolveTheme(mode) {
      if (mode === "light" || mode === "dark") return mode;
      return defaultTheme === "light" ? "light" : "dark";
    }
    function applyTheme(mode, persist) {
      mode = resolveTheme(mode);
      const resolved = resolveTheme(mode);
      currentThemeMode = mode;
      document.documentElement.dataset.theme = resolved;
      if (persist !== false) storageSet("monitor.theme", mode);
      $("theme-toggle").dataset.active = resolved;
      renderCharts();
    }
    function nextTheme() {
      applyTheme(currentThemeMode === "dark" ? "light" : "dark");
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
    function durationAxisNS(v) {
      return durationNS(v).replace(" ", "");
    }
    function diskCountLabel(disks) {
      const count = disks.length;
      if (!count) return t("diskDetails");
      if (currentLang === "zh-CN") return nf.format(count) + " 个磁盘";
      return nf.format(count) + " " + (count === 1 ? "disk" : "disks");
    }
    function clampPercent(value) {
      return Math.min(100, Math.max(0, Number(value || 0)));
    }
    function appendDiskStat(parent, label, value) {
      const row = document.createElement("div");
      row.className = "row";
      const dt = document.createElement("dt");
      dt.textContent = label;
      const dd = document.createElement("dd");
      dd.textContent = value;
      row.append(dt, dd);
      parent.appendChild(row);
    }
    function appendDetailRow(parent, label, value) {
      const row = document.createElement("div");
      row.className = "row";
      const dt = document.createElement("dt");
      dt.textContent = label;
      const dd = document.createElement("dd");
      dd.textContent = value;
      row.append(dt, dd);
      parent.appendChild(row);
    }
    function renderDetailRows(listID, rows) {
      const list = $(listID);
      if (!list) return;
      list.replaceChildren();
      rows.forEach(function(row) {
        appendDetailRow(list, row[0], row[1]);
      });
    }
    function openModal(modalID, closeID) {
      const modal = $(modalID);
      if (!modal) return;
      modal.hidden = false;
      const close = $(closeID);
      if (close) close.focus();
    }
    function closeModal(modalID, returnID) {
      const modal = $(modalID);
      if (modal) modal.hidden = true;
      const button = $(returnID);
      if (button) button.focus();
    }
    function renderDiskList() {
      const list = $("disk-modal-list");
      if (!list) return;
      list.replaceChildren();
      if (!currentDisks.length) {
        const empty = document.createElement("div");
        empty.className = "disk-item";
        empty.textContent = t("diskNoData");
        list.appendChild(empty);
        return;
      }
      currentDisks.forEach(function(disk) {
        const item = document.createElement("article");
        item.className = "disk-item";

        const head = document.createElement("div");
        head.className = "disk-item__head";
        const path = document.createElement("span");
        path.className = "disk-item__path";
        path.textContent = disk.path || "-";
        const usage = document.createElement("span");
        usage.className = "disk-item__usage";
        usage.textContent = pct(disk.used_percent);
        head.append(path, usage);

        const meta = document.createElement("div");
        meta.className = "disk-item__meta";
        const metaParts = [];
        if (disk.device) metaParts.push(t("diskDevice") + ": " + disk.device);
        if (disk.fstype) metaParts.push(t("diskType") + ": " + disk.fstype);
        meta.textContent = metaParts.join(" / ");

        const meter = document.createElement("div");
        meter.className = "disk-meter";
        const bar = document.createElement("div");
        bar.className = "disk-meter__bar";
        bar.style.setProperty("--disk-used", clampPercent(disk.used_percent) + "%");
        meter.appendChild(bar);

        const dl = document.createElement("dl");
        appendDiskStat(dl, t("diskTotal"), bytes(disk.total_bytes));
        appendDiskStat(dl, t("diskUsed"), bytes(disk.used_bytes));
        appendDiskStat(dl, t("diskFree"), bytes(disk.free_bytes));

        item.append(head, meta, meter, dl);
        list.appendChild(item);
      });
    }
    function updateDiskUI() {
      const button = $("disk-details-button");
      if (button) button.textContent = diskCountLabel(currentDisks);
      renderDiskList();
    }
    function openDiskModal() {
      renderDiskList();
      openModal("disk-modal", "disk-modal-close");
    }
    function closeDiskModal() {
      closeModal("disk-modal", "disk-details-button");
    }
    function updateRuntimeDetailUI() {
      const runtime = currentRuntime || {};
      const heapButton = $("heap-details-button");
      const gcButton = $("gc-details-button");
      if (heapButton) heapButton.textContent = bytes(runtime.heap_alloc_bytes);
      if (gcButton) gcButton.textContent = durationNS(runtime.gc_pause_last_ns);
      renderDetailRows("heap-modal-list", [
        [t("heapAlloc"), bytes(runtime.heap_alloc_bytes)],
        [t("heapSys"), bytes(runtime.heap_sys_bytes)],
        [t("heapObjects"), nf.format(runtime.heap_objects || 0)],
        [t("nextGC"), bytes(runtime.next_gc_bytes)],
        [t("mallocs"), nf.format(runtime.mallocs || 0)],
        [t("frees"), nf.format(runtime.frees || 0)]
      ]);
      renderDetailRows("gc-modal-list", [
        [t("gcCount"), nf.format(runtime.num_gc || 0)],
        [t("gcPauseLast"), durationNS(runtime.gc_pause_last_ns)],
        [t("gcPauseRecent"), durationNS(runtime.gc_pause_recent_ns)],
        [t("gcPauseTotal"), durationNS(runtime.gc_pause_total_ns)]
      ]);
    }
    function openHeapModal() {
      updateRuntimeDetailUI();
      openModal("heap-modal", "heap-modal-close");
    }
    function closeHeapModal() {
      closeModal("heap-modal", "heap-details-button");
    }
    function openGCModal() {
      updateRuntimeDetailUI();
      openModal("gc-modal", "gc-modal-close");
    }
    function closeGCModal() {
      closeModal("gc-modal", "gc-details-button");
    }
    function statusCodesTotal(codes) {
      return (codes["1xx"] || 0) + (codes["2xx"] || 0) + (codes["3xx"] || 0) + (codes["4xx"] || 0) + (codes["5xx"] || 0);
    }
    function updateHTTPStatusUI() {
      const codes = currentHTTPStatusCodes || {};
      const button = $("http-status-button");
      if (button) {
        const errors = (codes["4xx"] || 0) + (codes["5xx"] || 0);
        button.textContent = errors ? "ERR " + nf.format(errors) : "2xx " + nf.format(codes["2xx"] || 0);
      }
      renderDetailRows("http-status-modal-list", [
        ["1xx", nf.format(codes["1xx"] || 0)],
        ["2xx", nf.format(codes["2xx"] || 0)],
        ["3xx", nf.format(codes["3xx"] || 0)],
        ["4xx", nf.format(codes["4xx"] || 0)],
        ["5xx", nf.format(codes["5xx"] || 0)],
        [t("statusTotal"), nf.format(statusCodesTotal(codes))]
      ]);
    }
    function openHTTPStatusModal() {
      updateHTTPStatusUI();
      openModal("http-status-modal", "http-status-modal-close");
    }
    function closeHTTPStatusModal() {
      closeModal("http-status-modal", "http-status-button");
    }
    function closeVisibleModals() {
      if (!$("disk-modal").hidden) closeDiskModal();
      if (!$("heap-modal").hidden) closeHeapModal();
      if (!$("gc-modal").hidden) closeGCModal();
      if (!$("http-status-modal").hidden) closeHTTPStatusModal();
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
      $("pid-threads").textContent = nf.format(data.pid.threads || 0);
      $("pid-id").textContent = nf.format(data.pid.pid || 0);
      $("pid-fds").textContent = nf.format(data.pid.fds || 0);
      currentRuntime = data.runtime || {};
      $("rt-goroutines").textContent = nf.format(currentRuntime.goroutines || 0) + " / " + nf.format(currentRuntime.goroutine_peak || currentRuntime.goroutines || 0);
      updateRuntimeDetailUI();
      $("rt-next-gc").textContent = bytes(currentRuntime.next_gc_bytes);
      $("rt-uptime").textContent = uptime(currentRuntime.uptime_seconds);
      $("os-cpu").textContent = pct(data.os.cpu_percent);
      $("os-memory").textContent = pct(data.os.memory_used_percent);
      $("os-total").textContent = bytes(data.os.memory_total_bytes);
      currentDisks = Array.isArray(data.os.disks) ? data.os.disks : [];
      updateDiskUI();
      $("os-load").textContent = Number(data.os.load1 || 0).toFixed(2);
      $("http-requests").textContent = nf.format(data.http.total_requests || 0);
      $("http-in-flight").textContent = nf.format(data.http.in_flight_requests || 0);
      $("http-latency-recent").textContent = durationNS(data.http.latency && data.http.latency.recent_ns);
      $("http-latency-max").textContent = durationNS(data.http.latency && data.http.latency.max_ns);
      currentHTTPStatusCodes = data.http.status_codes || {};
      updateHTTPStatusUI();
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
      history.nextGCMiB.push(bytesToMiB(snapshot.runtime.next_gc_bytes || 0));
      history.goroutines.push(snapshot.runtime.goroutines || 0);
      history.requestsDelta.push(delta);
      history.latencyRecentNS.push((snapshot.http.latency && snapshot.http.latency.recent_ns) || 0);
      history.latencyMaxNS.push((snapshot.http.latency && snapshot.http.latency.max_ns) || 0);
      history.inFlight.push(snapshot.http.in_flight_requests || 0);
      history.gcPauseRecentNS.push(snapshot.runtime.gc_pause_recent_ns || 0);
      history.gcPauseLastNS.push(snapshot.runtime.gc_pause_last_ns || 0);
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
        const label = options.formatValue ? options.formatValue(value) : formatAxis(value, options.unit);
        ctx.fillText(label, 4, y);
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
      const padding = { top: 12, right: 10, bottom: 18, left: (options && options.leftPadding) || 58 };
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
      drawLineChart($("heap-gc-chart"), [
        { data: visibleSamples(history.heapMiB), color: good },
        { data: visibleSamples(history.nextGCMiB), color: warn }
      ], { minValue: 0, unit: "MiB" });
      drawLineChart($("goroutine-chart"), [
        { data: visibleSamples(history.goroutines), color: accent }
      ], { minValue: 0, unit: "g" });
      drawLineChart($("request-chart"), [
        { data: visibleSamples(history.requestsDelta), color: accent }
      ], { minValue: 0, unit: "req" });
      drawLineChart($("latency-chart"), [
        { data: visibleSamples(history.latencyRecentNS), color: accent },
        { data: visibleSamples(history.latencyMaxNS), color: warn }
      ], { minValue: 0, formatValue: durationAxisNS, leftPadding: 70 });
      drawLineChart($("in-flight-chart"), [
        { data: visibleSamples(history.inFlight), color: accent }
      ], { minValue: 0, unit: "req" });
      drawLineChart($("gc-pause-chart"), [
        { data: visibleSamples(history.gcPauseRecentNS), color: accent },
        { data: visibleSamples(history.gcPauseLastNS), color: warn }
      ], { minValue: 0, formatValue: durationAxisNS, leftPadding: 70 });
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
    $("disk-details-button").addEventListener("click", openDiskModal);
    $("disk-modal-close").addEventListener("click", closeDiskModal);
    $("heap-details-button").addEventListener("click", openHeapModal);
    $("heap-modal-close").addEventListener("click", closeHeapModal);
    $("gc-details-button").addEventListener("click", openGCModal);
    $("gc-modal-close").addEventListener("click", closeGCModal);
    $("http-status-button").addEventListener("click", openHTTPStatusModal);
    $("http-status-modal-close").addEventListener("click", closeHTTPStatusModal);
    $("disk-modal").addEventListener("click", function(event) {
      if (event.target === $("disk-modal")) closeDiskModal();
    });
    $("heap-modal").addEventListener("click", function(event) {
      if (event.target === $("heap-modal")) closeHeapModal();
    });
    $("gc-modal").addEventListener("click", function(event) {
      if (event.target === $("gc-modal")) closeGCModal();
    });
    $("http-status-modal").addEventListener("click", function(event) {
      if (event.target === $("http-status-modal")) closeHTTPStatusModal();
    });
    initMetricPagination();
    document.querySelectorAll(".sample-option").forEach(function(button) {
      button.addEventListener("click", function() {
        applySampleWindow(button.dataset.samples);
      });
    });
    window.addEventListener("scroll", updateScrollOrb, { passive: true });
    window.addEventListener("keydown", function(event) {
      if (event.key === "Escape") closeVisibleModals();
    });
    window.addEventListener("resize", function() {
      renderCharts();
      updateScrollOrb();
    });
    setInterval(function() {
      if (!lastSuccessAt || currentStatus === "error") return;
      if (Date.now() - lastSuccessAt > refreshMS * 3) setStatus("stale");
    }, 1000);

    currentLang = detectLang();
    applyTheme(storageGet("monitor.theme") || defaultTheme, false);
    applySampleWindow(defaultSampleWindow);
    applyLang(currentLang);
    updateScrollOrb();
    refresh();
    setInterval(refresh, refreshMS);
