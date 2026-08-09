<p align="center">
  <img src="https://img.shields.io/badge/License-MIT-6C757D?style=flat&color=3B82F6" alt="License">&nbsp&nbsp&nbsp
  <img src="https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat&logo=go&logoColor=white" alt="Go Version">&nbsp&nbsp&nbsp
  <a href="https://github.com/gofurry/monitor/actions/workflows/ci.yml"><img src="https://github.com/gofurry/monitor/actions/workflows/ci.yml/badge.svg" alt="CI"></a>&nbsp&nbsp&nbsp
  <a href="https://goreportcard.com/report/github.com/gofurry/monitor"><img src="https://goreportcard.com/badge/github.com/gofurry/monitor" alt="Go Report Card"></a>&nbsp&nbsp&nbsp
</p>

<p align="left">
  <a href="../../README.md">English</a> | 中文
</p>

一个用于实时查看 Go 服务基本状态的极轻量 `net/http` 中间件。

`monitor` 刻意保持很小：一个中间件、一个页面、一个 JSON 快照。

状态页完全内嵌，不需要前端构建步骤，也不依赖外部 JavaScript 库。

它包含：

- 亮色 / 暗色主题
- 纯色 / 网格背景
- 英文和简体中文界面
- LIVE / PARTIAL / STALE / ERROR 状态
- 基于原生 Canvas 的短期浏览器内趋势图
- 通过 `Accept: application/json` 获取 JSON 快照
- 请求速率、近期错误率和延迟百分位
- 聚合网络 I/O 与 Linux cgroup v2 限制

趋势图只保留很短的浏览器内历史。服务端不存储指标。进程重启后，内存计数器和图表历史都会清空。

<p align="center">
  <img src="../releases/preview.png" alt="monitor 状态页预览">
</p>

## 安装

```sh
go get github.com/gofurry/monitor
```

## 快速开始

```go
package main

import (
	"net/http"

	"github.com/gofurry/monitor"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	})

	handler := monitor.New(mux)
	_ = http.ListenAndServe(":8080", handler)
}
```

打开：

```text
http://localhost:8080/monitor
```

获取 JSON：

```sh
curl -H "Accept: application/json" http://localhost:8080/monitor
```

## Fiber

`monitor` 基于 `net/http`。Fiber 基于 `fasthttp`，因此推荐在服务启动阶段只创建一个 monitor 实例，通过 Fiber 官方 adaptor 只暴露监控端点，再用 Fiber 原生中间件记录业务请求。

> 重要：不要在 `adaptor.HTTPMiddleware` 内部调用 `monitor.New` 或 `monitor.NewMonitor`。
> Fiber 会在每个请求里执行这个中间件工厂函数，而每个 monitor 实例都会启动一个后台采集 goroutine。如果每个请求都创建 monitor 实例，就会泄漏采集 goroutine。

```go
package main

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofurry/monitor"
)

func main() {
	app := fiber.New()

	m := monitor.NewMonitor(http.NotFoundHandler(), monitor.Config{
		Path: "/monitor",
	})
	defer m.Stop()

	app.Use(func(c fiber.Ctx) error {
		if c.Path() == "/monitor" {
			return c.Next()
		}

		finish := m.BeginRequest()
		err := c.Next()

		status := c.Response().StatusCode()
		if err != nil {
			status = fiber.StatusInternalServerError
			if fiberErr, ok := err.(*fiber.Error); ok {
				status = fiberErr.Code
			}
		}
		finish(status)
		return err
	})

	app.All("/monitor", adaptor.HTTPHandler(m))

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("hello")
	})

	_ = app.Listen(":8080")
}
```

打开 `http://localhost:8080/monitor`。

这个 Fiber 示例可以安全地提供监控页面和 JSON 快照，同时通过原生 Fiber 中间件记录 `http.total_requests`、处理中请求、状态码分类和请求延迟。

## Gin

Gin 运行在 `net/http` 之上，但如果你希望 monitor 不直接包裹 Gin handler，也可以使用框架无关的请求生命周期方法。

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofurry/monitor"
)

func main() {
	r := gin.New()
	r.Use(gin.Recovery())

	m := monitor.NewMonitor(http.NotFoundHandler(), monitor.Config{
		Path: "/monitor",
	})
	defer m.Stop()

	r.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/monitor" {
			c.Next()
			return
		}

		finish := m.BeginRequest()
		c.Next()

		status := c.Writer.Status()
		if status == 0 {
			status = http.StatusOK
		}
		finish(status)
	})

	r.GET("/monitor", gin.WrapH(m))
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})

	_ = r.Run(":8080")
}
```

## Echo

Echo 也可以用同一组 monitor 生命周期方法记录请求。

```go
package main

import (
	"net/http"

	"github.com/gofurry/monitor"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	m := monitor.NewMonitor(http.NotFoundHandler(), monitor.Config{
		Path: "/monitor",
	})
	defer m.Stop()

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().URL.Path == "/monitor" {
				return next(c)
			}

			finish := m.BeginRequest()
			err := next(c)

			status := c.Response().Status
			if err != nil {
				status = http.StatusInternalServerError
				if echoErr, ok := err.(*echo.HTTPError); ok {
					status = echoErr.Code
				}
			}
			if status == 0 {
				status = http.StatusOK
			}
			finish(status)
			return err
		}
	})

	e.GET("/monitor", echo.WrapHandler(m))
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello")
	})

	_ = e.Start(":8080")
}
```

## 配置

```go
handler := monitor.New(mux, monitor.Config{
	Path:                "/monitor",
	Title:               "My App Monitor",
	Description:         "Live production service metrics.",
	Footer:              "Copyright 2026 Example Inc.",
	FaviconURL:          "/assets/favicon.svg",
	ServiceName:         "payments-api",
	Version:             "v1.2.0",
	Environment:         "production",
	DefaultLanguage:     "en",
	DefaultTheme:        "dark",
	Background:          "solid",
	DefaultSampleWindow: 60,
	DiskPaths:           nil,
	Refresh:             2 * time.Second,
	APIOnly:             false,
	Authorize: func(r *http.Request) bool {
		return r.Header.Get("Authorization") == "Bearer change-me"
	},
	IgnoreRequest: func(r *http.Request) bool {
		return r.URL.Path == "/healthz" || r.URL.Path == "/readyz"
	},
})
```

默认值：

| 字段 | 默认值 | 说明 |
|---|---:|---|
| `Path` | `/monitor` | HTML 页面和 JSON 快照端点。 |
| `Title` | `Monitor` | HTML 页面标题和主标题。 |
| `Description` | `Live process, runtime, system, and HTTP metrics for this Go service.` | 页头下方的简短说明。 |
| `Footer` | `Powered by github.com/gofurry/monitor - MIT License.` | 页脚文本，可用于版权、归属或许可证说明。 |
| `FaviconURL` | 内嵌 favicon | 用站内相对 URL 或 HTTP(S) 绝对 URL 覆盖仪表盘图标。为空或无效时使用内嵌图标。 |
| `ServiceName` | 可执行文件名 | UI 和快照中显示的服务名。 |
| `Version` | Go module 构建版本 | 当前服务版本；显式配置优先。 |
| `Environment` | 空 | 部署环境，例如 `production`。 |
| `DefaultLanguage` | `en` | 浏览器没有保存偏好时的初始 UI 语言。支持值：`en`、`zh-CN`。 |
| `DefaultTheme` | `dark` | 浏览器没有保存偏好时的初始 UI 主题。支持值：`light`、`dark`。 |
| `Background` | `solid` | HTML 页面背景。支持值：`solid`、`grid`。 |
| `DefaultSampleWindow` | `60` | 趋势图初始采样点数量。支持值：`30`、`60`、`90`。 |
| `DiskPaths` | `nil` | 要采样磁盘使用率的文件系统路径。为空时采样当前工作目录所在文件系统。 |
| `Refresh` | `2s` | 后台指标采集间隔；小于 `250ms` 的值会被限制为 `250ms`。 |
| `APIOnly` | `false` | 让 `Path` 只返回 JSON，不提供 HTML。 |
| `Authorize` | `nil` | 决定是否允许访问 `Path`；拒绝时返回 `401 Unauthorized`。 |
| `IgnoreRequest` | `nil` | 从 `http.total_requests` 中排除指定请求。 |

访问 `Path` 的请求始终不会计入 `http.total_requests`；监控页面刷新和 JSON 轮询不会污染业务请求数。`IgnoreRequest` 用于排除其他非业务流量，例如负载均衡健康检查。被忽略的请求仍会继续交给你的 handler 处理。

### 仪表盘 favicon

仪表盘默认包含内嵌 favicon。通过 `FaviconURL` 可以使用应用自身提供的图标或远程 HTTP(S) URL。不能直接填写 `./favicon.ico` 这类文件系统路径，应先由应用提供该文件，再配置对应 URL。建议使用同源 URL，避免浏览器访问额外的远程主机。

### 访问控制

仪表盘包含运行信息，生产环境不应默认公开。`Authorize` 只作用于配置的监控路径，并且先于方法分派执行，因此未授权的 HTML、JSON、`HEAD` 和不支持的方法请求都会得到 `401 Unauthorized`。

```go
handler := monitor.New(mux, monitor.Config{
	Authorize: func(r *http.Request) bool {
		return subtle.ConstantTimeCompare(
			[]byte(r.Header.Get("Authorization")),
			[]byte("Bearer "+os.Getenv("MONITOR_TOKEN")),
		) == 1
	},
})
```

TLS 终止和网络层访问限制应在本包之外完成，监控 token 不要提交到源码仓库。

### 采集行为

每个快照都会记录 UTC 采集时间、实际采集耗时和 partial 状态。单个 collector 失败不会让 `/monitor` 失败；UI 会将受影响的值显示为 `N/A`，`collection.errors` 只包含 `pid.cpu`、`os.disk`、`os.network`、`container.memory` 等稳定标识，不会暴露底层系统错误文本。

网络总量会汇总主机报告的全部接口，明确包含 loopback。首次采集和计数器回退后的速率为零。在 Linux 上，容器指标读取 cgroup v2 的 `memory.current`、`memory.max` 和 `cpu.max`，并支持 unlimited。仅在检测到容器标记、已知 cgroup 路径或有限资源限制组合时显示容器卡片；其他操作系统返回未检测到容器的快照。

## 最佳实践

`monitor` 不持久化指标、日志、链路追踪或图表历史。它只展示当前进程、当前主机、Go runtime，以及当前中间件实例处理到的请求情况。

它最适合单机单体 Go 服务：当你只需要一个非常轻量的内置状态页，用来观察本机本服务的基本健康和运行时状态时，`monitor` 会很合适。

如果你需要下面这些能力，请使用 Prometheus、Grafana、链路追踪和集中日志等专门的可观测性方案：

- 长期历史
- 告警
- 多实例聚合
- 分布式追踪
- 业务指标
- 集群级仪表盘

## 范围

`monitor` 会：

- 提供轻量状态页
- 提供 JSON 快照
- 展示当前进程指标
- 展示 Go runtime 指标，包括 GC 暂停时间
- 展示基础系统指标
- 展示包含 loopback 的聚合网络总量与速率
- 展示检测到的 Linux cgroup v2 内存和 CPU 限制
- 统计业务请求总数
- 追踪 RPS、处理中请求、状态码分类、近期错误率和延迟百分位
- 不依赖外部图表库，直接渲染短期浏览器内趋势图
- 支持亮色 / 暗色主题
- 支持英文和简体中文 UI

`monitor` 不会：

- 存储历史指标
- 发送告警
- 替代 Prometheus 或 Grafana
- 提供链路追踪
- 聚合多个实例
- 采集应用业务指标
- 在服务端存储图表历史
- 依赖外部图表库
- 提供可配置告警阈值

## JSON 快照

```json
{
  "schema_version": 1,
  "collected_at": "2026-08-09T10:30:00Z",
  "collection": {
    "duration_ns": 1850000,
    "partial": false
  },
  "service": {
    "name": "payments-api",
    "version": "v1.2.0",
    "environment": "production",
    "go_version": "go1.24.6",
    "module": "example.com/payments",
    "revision": "abc123",
    "vcs_modified": false
  },
  "pid": {
    "cpu_percent": 2.4,
    "rss_bytes": 48140288,
    "pid": 12345,
    "threads": 12,
    "fds": 32
  },
  "runtime": {
    "goroutines": 18,
    "goroutine_peak": 42,
    "heap_alloc_bytes": 7327744,
    "heap_sys_bytes": 12582912,
    "heap_objects": 42011,
    "next_gc_bytes": 14655488,
    "mallocs": 260112,
    "frees": 218101,
    "num_gc": 12,
    "gc_pause_last_ns": 128000,
    "gc_pause_total_ns": 3200000,
    "gc_pause_recent_ns": 128000,
    "uptime_seconds": 3600
  },
  "os": {
    "cpu_percent": 12.8,
    "memory_used_percent": 61.5,
    "memory_total_bytes": 8589934592,
    "disk_used_percent": 47.2,
    "disk_total_bytes": 512110190592,
    "disk_used_bytes": 241737318400,
    "disks": [
      {
        "path": "C:\\",
        "device": "C:",
        "fstype": "NTFS",
        "total_bytes": 512110190592,
        "used_bytes": 241737318400,
        "free_bytes": 270372872192,
        "used_percent": 47.2
      },
      {
        "path": "D:\\",
        "device": "D:",
        "fstype": "NTFS",
        "total_bytes": 1024209543168,
        "used_bytes": 388547952640,
        "free_bytes": 635661590528,
        "used_percent": 37.9
      }
    ],
    "load1": 0.42,
    "network": {
      "received_bytes": 104857600,
      "sent_bytes": 52428800,
      "receive_bps": 8192,
      "send_bps": 4096
    }
  },
  "container": {
    "detected": true,
    "memory_usage_bytes": 134217728,
    "memory_limit_bytes": 536870912,
    "memory_used_percent": 25,
    "cpu_quota_cores": 2
  },
  "http": {
    "total_requests": 1024,
    "in_flight_requests": 2,
    "rps": 18.5,
    "status_codes": {
      "1xx": 0,
      "2xx": 1000,
      "3xx": 12,
      "4xx": 10,
      "5xx": 2
    },
    "rates": {
      "4xx_rate": 0.01,
      "5xx_rate": 0.002,
      "error_rate": 0.012
    },
    "latency": {
      "last_ns": 812000,
      "recent_ns": 924500,
      "p50_ns": 1000000,
      "p95_ns": 5000000,
      "p99_ns": 10000000,
      "recent_max_ns": 11800000,
      "max_ns": 12000000
    }
  }
}
```

## 生命周期

`New` 返回一个 `http.Handler`，适合最简单的接入方式。如果你想读取当前快照或显式停止后台采集器，可以使用 `NewMonitor`：

```go
m := monitor.NewMonitor(mux)
defer m.Stop()

stats := m.Current()
_ = stats
```

框架适配器可以使用 `BeginRequest` 安全地配对开始和结束，并自动忽略重复结束调用：

```go
finish := m.BeginRequest()
defer finish(http.StatusOK)
```

底层的 `RequestStarted`、`RequestFinished` 和 `ObserveRequest` 仍然保留。原生 `net/http` 包装继续走直接生命周期路径，以保持较低的热路径分配。

`Monitor` 可以安全地并发使用。

## 性能基线

运行 benchmark：

```sh
go test -run=^$ -bench=Benchmark -benchmem .
```

benchmark 覆盖：

- 直接 `net/http` handler 开销
- monitor 包装后的业务请求
- 固定桶延迟直方图写入
- `BeginRequest` 生命周期调用
- 并行业务请求
- 被忽略请求
- JSON 快照响应
- HTML 状态页响应
- `Current()` 快照读取

## 说明

- 访问监控路径的请求不会计入业务请求。
- 监控路径只接受 `GET` 和 `HEAD`，响应会设置 `Cache-Control: no-store`、`Referrer-Policy: no-referrer` 和 `X-Content-Type-Options: nosniff`。
- 指标由后台 ticker 采集，并从最新的并发安全快照中返回。
- 部分指标采集失败不会导致监控端点失败；JSON 返回稳定错误标识，UI 将受影响值显示为 `N/A`。
- HTML 页面没有外部前端依赖；模板、CSS 和 JavaScript 都从 `internal/ui` 内嵌。

## 相关文档

- [贡献指南](CONTRIBUTING.md)
- [安全政策](SECURITY.md)
