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
- LIVE / STALE / ERROR 状态
- 基于原生 Canvas 的短期浏览器内趋势图
- 通过 `Accept: application/json` 获取 JSON 快照

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

`monitor` 基于 `net/http`。Fiber 基于 `fasthttp`，但 Fiber 官方 adaptor 中间件可以包装 `net/http` 中间件：

```go
package main

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofurry/monitor"
)

func main() {
	app := fiber.New()

	app.Use(adaptor.HTTPMiddleware(func(next http.Handler) http.Handler {
		return monitor.New(next, monitor.Config{
			Path: "/monitor",
		})
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("hello")
	})

	_ = app.Listen(":8080")
}
```

打开 `http://localhost:8080/monitor`。

## 配置

```go
handler := monitor.New(mux, monitor.Config{
	Path:                "/monitor",
	Title:               "My App Monitor",
	Description:         "Live production service metrics.",
	Footer:              "Copyright 2026 Example Inc.",
	DefaultLanguage:     "en",
	DefaultTheme:        "dark",
	Background:          "solid",
	DefaultSampleWindow: 60,
	DiskPaths:           nil,
	Refresh:             2 * time.Second,
	APIOnly:             false,
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
| `DefaultLanguage` | `en` | 浏览器没有保存偏好时的初始 UI 语言。支持值：`en`、`zh-CN`。 |
| `DefaultTheme` | `dark` | 浏览器没有保存偏好时的初始 UI 主题。支持值：`light`、`dark`。 |
| `Background` | `solid` | HTML 页面背景。支持值：`solid`、`grid`。 |
| `DefaultSampleWindow` | `60` | 趋势图初始采样点数量。支持值：`30`、`60`、`90`。 |
| `DiskPaths` | `nil` | 要采样磁盘使用率的文件系统路径。为空时采样当前工作目录所在文件系统。 |
| `Refresh` | `2s` | 后台指标采集间隔。 |
| `APIOnly` | `false` | 让 `Path` 只返回 JSON，不提供 HTML。 |
| `IgnoreRequest` | `nil` | 从 `http.total_requests` 中排除指定请求。 |

访问 `Path` 的请求始终不会计入 `http.total_requests`；监控页面刷新和 JSON 轮询不会污染业务请求数。`IgnoreRequest` 用于排除其他非业务流量，例如负载均衡健康检查。被忽略的请求仍会继续交给你的 handler 处理。

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
- 统计业务请求总数
- 追踪处理中请求、HTTP 状态码分类和近期请求延迟
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
    "load1": 0.42
  },
  "http": {
    "total_requests": 1024,
    "in_flight_requests": 2,
    "status_codes": {
      "1xx": 0,
      "2xx": 1000,
      "3xx": 12,
      "4xx": 10,
      "5xx": 2
    },
    "latency": {
      "last_ns": 812000,
      "recent_ns": 924500,
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

`Monitor` 可以安全地并发使用。

## 性能基线

运行 benchmark：

```sh
go test -run=^$ -bench=Benchmark -benchmem .
```

benchmark 覆盖：

- 直接 `net/http` handler 开销
- monitor 包装后的业务请求
- 并行业务请求
- 被忽略请求
- JSON 快照响应
- HTML 状态页响应
- `Current()` 快照读取

## 说明

- 访问监控路径的请求不会计入业务请求。
- 监控路径只接受 `GET` 和 `HEAD`，响应会设置 `Cache-Control: no-store`、`Referrer-Policy: no-referrer` 和 `X-Content-Type-Options: nosniff`。
- 指标由后台 ticker 采集，并从最新的并发安全快照中返回。
- 部分指标采集失败时，相关值保持为零，不会导致监控端点失败。
- HTML 页面没有外部前端依赖；模板、CSS 和 JavaScript 都从 `internal/ui` 内嵌。

## 相关文档

- [贡献指南](CONTRIBUTING.md)
- [安全政策](SECURITY.md)
