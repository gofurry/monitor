# monitor v1.2 改进与实施文档

> 项目：`github.com/gofurry/monitor`  
> 目标版本：`v1.2.0`  
> 文档用途：版本设计、开发实施、测试与发布验收  
> 核心原则：保持 `monitor` 轻量、嵌入式、零外部服务依赖，不演进为完整监控平台。

---

## 1. 版本目标

`v1.2` 定位为一次 **Production Readiness / 实时可观测性增强版本**。

当前 `monitor` 已具备：

- `net/http` 中间件能力
- 嵌入式 HTML 状态页
- JSON Snapshot
- Process / Go Runtime / OS / HTTP 指标
- HTTP 请求计数、in-flight、状态码统计
- 最近请求延迟 EWMA
- 多磁盘采集
- Light / Dark Theme
- 中英文 UI
- Fiber / Gin / Echo 接入方式
- benchmark 与基础 CI

`v1.2` 不改变项目定位，而是在现有模型上重点解决以下问题：

1. JSON Snapshot 缺少版本与采集状态信息。
2. HTTP 指标缺少 RPS、错误率、P50/P95/P99 等生产环境常用指标。
3. 生命周期最大延迟 `max_ns` 长时间运行后参考价值不足。
4. Monitor 页面缺少服务版本、构建信息、运行环境等上下文。
5. 容器环境下 Host 指标可能不能真实反映进程资源限制。
6. Monitor 页面暴露生产运行信息，但目前缺少内建访问控制挂钩。
7. `Refresh` 可被错误配置为极小值，造成过度采集。
8. 手工 Request 生命周期 API 存在 in-flight 下溢风险。
9. CI 仍可补充 race、Windows 与多 Go 版本验证。
10. JSON / UI 对“指标值为 0”和“采集失败”缺少区分。

---

# 2. v1.2 设计原则

## 2.1 保持轻量

不引入：

- PostgreSQL
- Redis
- SQLite
- Prometheus Server
- 外部前端运行时
- 外部 JS 图表库
- 消息队列
- 后台 Agent
- 独立 Monitor Server

整个组件继续满足：

```text
import middleware
        ↓
启动一个轻量后台 collector
        ↓
访问 /monitor
        ↓
看到当前服务状态
```

---

## 2.2 不做历史监控平台

v1.2 明确不实现：

- 服务端长期历史数据
- 时序数据库
- 告警
- PromQL
- Dashboard 编排
- 多实例聚合
- 分布式 tracing
- 日志存储
- SLO / SLA 系统
- Redis / MySQL / PostgreSQL 专用 Collector
- Plugin / Collector 扩展机制

Collector 扩展机制统一推迟到未来 `v2` 再评估。

---

## 2.3 请求热路径必须保持简单

业务请求路径中允许：

- atomic counter
- 时间戳
- 固定桶 atomic increment
- 少量状态判断

避免：

- slice append
- mutex 竞争
- 动态 map 扩容
- 系统调用
- gopsutil
- JSON 编码
- 网络请求
- 每请求 goroutine
- 每请求对象大量分配

---

# 3. v1.2 功能范围

建议将 v1.2 分为以下 8 个工作包。

```text
1. Snapshot Metadata
2. HTTP Realtime Metrics
3. Service / Build Info
4. Container / cgroup Metrics
5. Access Control
6. Runtime Safety Fixes
7. UI Enhancement
8. Test / Benchmark / CI
```

---

# 4. Snapshot Metadata

## 4.1 目标

解决当前 JSON Snapshot 无法判断：

- Snapshot Schema 版本
- 数据采集时间
- 数据新鲜度
- Collector 是否部分失败
- 一次采集耗时多久

的问题。

---

## 4.2 建议数据结构

新增：

```go
type Stats struct {
    SchemaVersion int            `json:"schema_version"`
    CollectedAt   time.Time      `json:"collected_at"`
    Collection    CollectionStats `json:"collection"`

    Service ServiceStats `json:"service"`
    PID     PIDStats     `json:"pid"`
    Runtime RuntimeStats `json:"runtime"`
    OS      OSStats      `json:"os"`
    HTTP    HTTPStats    `json:"http"`
}
```

新增：

```go
type CollectionStats struct {
    DurationNS uint64   `json:"duration_ns"`
    Partial    bool     `json:"partial"`
    Errors     []string `json:"errors,omitempty"`
}
```

初始：

```go
const StatsSchemaVersion = 1
```

---

## 4.3 JSON 示例

```json
{
  "schema_version": 1,
  "collected_at": "2026-08-09T10:00:00Z",
  "collection": {
    "duration_ns": 2829100,
    "partial": false
  }
}
```

若磁盘采集失败：

```json
{
  "collection": {
    "duration_ns": 3189000,
    "partial": true,
    "errors": [
      "os.disk"
    ]
  }
}
```

---

## 4.4 错误设计

v1.2 暂时不暴露底层完整错误字符串。

推荐：

```text
pid.cpu
pid.memory
pid.threads
pid.fds

os.cpu
os.memory
os.disk
os.load

runtime.memstats

container.cpu
container.memory
```

原因：

- 避免泄漏主机内部信息
- JSON 更稳定
- UI 更容易国际化
- 不受操作系统错误文本影响

---

## 4.5 验收标准

- 每个 Snapshot 必须带 `schema_version`
- 每个 Snapshot 必须带 `collected_at`
- 每次 collector 运行记录采集耗时
- 单个指标采集失败不能使 `/monitor` 返回 500
- 存在任意采集失败时 `partial=true`
- JSON 可明确区分“采集成功且值为 0”与“采集失败”

---

# 5. HTTP Realtime Metrics

这是 v1.2 最核心的功能增强。

---

## 5.1 新增指标

建议增加：

```text
RPS
Recent 4xx Rate
Recent 5xx Rate
Recent Error Rate
P50 Latency
P95 Latency
P99 Latency
Recent Max Latency
```

保留现有：

```text
total_requests
in_flight_requests
status_codes
last_ns
recent_ns
max_ns
```

为保持兼容，现有字段在 v1.2 不删除。

---

# 6. RPS

## 6.1 实现思路

不在请求路径直接计算 RPS。

继续使用：

```go
requests atomic.Uint64
```

在 collector 周期内：

```text
本次 total_requests - 上一次 total_requests
---------------------------------------------
        实际 elapsed seconds
```

计算：

```go
RPS float64
```

---

## 6.2 数据结构

```go
type HTTPStats struct {
    TotalRequests    uint64          `json:"total_requests"`
    InFlightRequests uint64          `json:"in_flight_requests"`
    RPS              float64         `json:"rps"`

    StatusCodes StatusCodeStats `json:"status_codes"`
    Rates       HTTPRateStats    `json:"rates"`
    Latency     LatencyStats     `json:"latency"`
}
```

---

## 6.3 注意

必须使用实际 elapsed，而不是直接假定：

```text
Refresh == 2s
```

否则 GC、系统调度、机器负载会产生误差。

---

# 7. Recent Error Rate

## 7.1 建议统计

```go
type HTTPRateStats struct {
    Status4xxRate float64 `json:"4xx_rate"`
    Status5xxRate float64 `json:"5xx_rate"`
    ErrorRate     float64 `json:"error_rate"`
}
```

定义：

```text
4xx_rate = interval_4xx / interval_requests

5xx_rate = interval_5xx / interval_requests

error_rate = interval_(4xx + 5xx) / interval_requests
```

单位建议：

```text
0.0 ~ 1.0
```

例如：

```json
{
  "5xx_rate": 0.0023
}
```

UI 显示：

```text
0.23%
```

---

## 7.2 不建议

不要直接用整个进程生命周期：

```text
total_5xx / total_requests
```

作为主 UI 指标。

原因是服务运行几天后，短时故障会被长期平均值稀释。

生命周期累计计数仍然保留在 `status_codes`。

---

# 8. Latency Histogram

## 8.1 目标

解决只有：

```text
last
EWMA recent
lifetime max
```

无法快速判断尾延迟的问题。

---

## 8.2 固定 Bucket

建议使用固定桶：

```text
<= 1 ms
<= 5 ms
<= 10 ms
<= 25 ms
<= 50 ms
<= 100 ms
<= 250 ms
<= 500 ms
<= 1 s
<= 2.5 s
<= 5 s
> 5 s
```

可以定义：

```go
var latencyBoundsNS = [...]uint64{
    1 * uint64(time.Millisecond),
    5 * uint64(time.Millisecond),
    10 * uint64(time.Millisecond),
    25 * uint64(time.Millisecond),
    50 * uint64(time.Millisecond),
    100 * uint64(time.Millisecond),
    250 * uint64(time.Millisecond),
    500 * uint64(time.Millisecond),
    1 * uint64(time.Second),
    2500 * uint64(time.Millisecond),
    5 * uint64(time.Second),
}
```

---

## 8.3 请求路径

在：

```go
recordBusinessRequest()
```

中：

```text
计算 duration
↓
找到 bucket
↓
atomic increment
```

不保存单次请求。

---

## 8.4 周期窗口

建议 v1.2 使用：

```text
collector interval window
```

即每次刷新计算过去一个采集周期的：

```text
P50
P95
P99
Recent Max
```

然后清空/切换 bucket。

---

## 8.5 推荐实现：双缓冲 bucket

避免 collector 清零 atomic 时与业务请求竞争，可以维护两组 histogram：

```text
active histogram
inactive histogram
```

collector 时：

```text
atomic swap active index
↓
新请求立即写新 histogram
↓
collector 读取旧 histogram
↓
计算 percentile
↓
清空旧 histogram
```

结构示意：

```go
type latencyHistogram struct {
    buckets [12]atomic.Uint64
    maxNS   atomic.Uint64
}

type Monitor struct {
    latencyHistograms [2]latencyHistogram
    activeHistogram   atomic.Uint32
}
```

---

## 8.6 Percentile

不要求精确 percentile。

根据累计 bucket 计算近似：

```text
P50
P95
P99
```

这是一个实时状态页，不是精确统计分析系统。

---

## 8.7 新 LatencyStats

建议：

```go
type LatencyStats struct {
    LastNS      uint64  `json:"last_ns"`
    RecentNS    float64 `json:"recent_ns"`

    P50NS       uint64  `json:"p50_ns"`
    P95NS       uint64  `json:"p95_ns"`
    P99NS       uint64  `json:"p99_ns"`

    RecentMaxNS uint64  `json:"recent_max_ns"`
    MaxNS       uint64  `json:"max_ns"`
}
```

其中：

```text
RecentNS     = EWMA
RecentMaxNS  = 最近采集窗口最大值
MaxNS        = 生命周期最大值
```

---

# 9. Service / Build Info

## 9.1 目标

让用户打开 `/monitor` 后第一眼知道：

```text
这是哪个服务
跑的是哪个版本
是什么环境
使用什么 Go 版本
编译版本是什么
```

---

## 9.2 配置

新增：

```go
type Config struct {
    // existing...

    ServiceName string
    Version     string
    Environment string
}
```

推荐默认：

```text
ServiceName = executable name
Version     = build info module version / "(devel)"
Environment = ""
```

---

## 9.3 自动采集

可以通过：

```go
runtime.Version()
debug.ReadBuildInfo()
os.Executable()
```

获得：

```text
Go Version
Module
Module Version
VCS Revision
VCS Time
VCS Modified
Executable
```

---

## 9.4 数据结构

```go
type ServiceStats struct {
    Name        string `json:"name,omitempty"`
    Version     string `json:"version,omitempty"`
    Environment string `json:"environment,omitempty"`

    GoVersion   string `json:"go_version"`
    Module      string `json:"module,omitempty"`
    Revision    string `json:"revision,omitempty"`
    VCSModified bool   `json:"vcs_modified,omitempty"`
}
```

---

## 9.5 隐私原则

默认不要暴露：

- 完整 executable filesystem path
- GOPATH
- 工作目录
- Build Host
- 用户名
- 环境变量
- Git Remote URL

UI/JSON 仅保留必要构建标识。

---

# 10. Container / cgroup Metrics

## 10.1 目标

当 Go 服务运行在 Linux Container 中时，展示：

```text
Host Resources
Container Limits
Container Usage
```

避免只展示宿主机 32GB 内存，而容器实际 limit 只有 2GB。

---

## 10.2 v1.2 范围

只支持：

```text
Linux cgroup v2
```

可选兼容 cgroup v1，但不作为 v1.2 必须目标。

Windows Container 暂不实现。

---

## 10.3 建议指标

```go
type ContainerStats struct {
    Detected bool `json:"detected"`

    MemoryUsageBytes uint64 `json:"memory_usage_bytes,omitempty"`
    MemoryLimitBytes uint64 `json:"memory_limit_bytes,omitempty"`
    MemoryUsedPercent float64 `json:"memory_used_percent,omitempty"`

    CPUQuotaCores float64 `json:"cpu_quota_cores,omitempty"`
}
```

可以进一步增加：

```text
CPU usage percent
```

但如果实现复杂，可将 CPU usage 放到后续 v1.2.x。

---

## 10.4 cgroup v2 文件

主要读取：

```text
/sys/fs/cgroup/memory.current
/sys/fs/cgroup/memory.max
/sys/fs/cgroup/cpu.max
```

---

## 10.5 memory.max

需要处理：

```text
max
```

代表无限制。

此时：

```text
memory_limit_bytes = 0
```

UI 显示：

```text
Unlimited
```

而不是：

```text
0 B
```

---

## 10.6 降级策略

Container 采集失败不能影响主 Snapshot。

例如：

```text
cgroup 不存在
↓
Detected = false
```

不是 Error。

只有：

```text
明确检测到 cgroup
但关键文件读取失败
```

才可加入：

```text
collection.errors
```

---

# 11. Access Control

## 11.1 目标

Monitor 不实现完整用户系统。

仅提供最小访问控制 Hook。

---

## 11.2 推荐 API

建议：

```go
type Config struct {
    // existing...

    Authorize func(*http.Request) bool
}
```

逻辑：

```go
if request is monitor path {
    if cfg.Authorize != nil && !cfg.Authorize(r) {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    serveMonitor(...)
}
```

---

## 11.3 用户可以自行实现

### Bearer Token

```go
Authorize: func(r *http.Request) bool {
    return r.Header.Get("Authorization") == "Bearer secret"
},
```

### Basic Auth

```go
Authorize: func(r *http.Request) bool {
    user, pass, ok := r.BasicAuth()
    return ok && user == "admin" && pass == "secret"
},
```

### IP 白名单

```go
Authorize: func(r *http.Request) bool {
    // application-specific logic
    return true
},
```

---

## 11.4 不要实现

v1.2 不加入：

```text
Username
Password
JWT Secret
Session
Login Page
Cookie Auth
User Database
OAuth
OIDC
```

这些会使 monitor 偏离 middleware 定位。

---

## 11.5 状态码

默认建议：

```text
401 Unauthorized
```

若用户希望隐藏 monitor 存在，可以自行在 Authorize 外层路由处理。

---

# 12. Refresh 配置保护

## 12.1 当前问题

如果：

```go
Refresh: time.Millisecond
```

后台 collector 会高频执行 gopsutil。

---

## 12.2 方案

增加：

```go
const minRefresh = 250 * time.Millisecond
```

applyConfig：

```go
if cfg.Refresh <= 0 {
    cfg.Refresh = defaultRefresh
}

if cfg.Refresh < minRefresh {
    cfg.Refresh = minRefresh
}
```

---

## 12.3 UI

浏览器 JSON polling 和后台 collector 使用相同有效 `Refresh`。

避免：

```text
collector 1ms
browser 250ms
```

这种不一致。

---

# 13. Request Lifecycle 安全修复

## 13.1 当前风险

错误调用：

```go
m.RequestFinished(...)
```

而没有先：

```go
m.RequestStarted()
```

可能造成：

```text
uint64 underflow
```

最终：

```text
in_flight_requests = 18446744073709551615
```

---

## 13.2 最低要求

`RequestFinished()` 必须保证：

```text
inFlight == 0
```

时不会继续递减。

可以使用 CAS：

```go
func decrementIfPositive(v *atomic.Uint64) {
    for {
        current := v.Load()
        if current == 0 {
            return
        }
        if v.CompareAndSwap(current, current-1) {
            return
        }
    }
}
```

---

## 13.3 新便捷 API

建议增加：

```go
func (m *Monitor) BeginRequest() func(status int)
```

用法：

```go
finish := m.BeginRequest()

// handle request

finish(http.StatusOK)
```

内部：

```text
RequestStarted
记录 started
sync.Once 防重复 finish
RequestFinished
```

---

## 13.4 更推荐 API

如果希望避免 callback 风格，也可增加：

```go
type RequestObservation struct {
    ...
}

func (m *Monitor) StartRequest() *RequestObservation
func (o *RequestObservation) Finish(status int)
```

但 v1.2 为避免 API 膨胀，推荐只选择一种。

建议使用：

```go
BeginRequest() func(status int)
```

---

# 14. Network IO

建议作为 v1.2 的次要功能。

新增：

```text
Network RX Bytes
Network TX Bytes
RX Rate
TX Rate
```

---

## 14.1 数据结构

```go
type NetworkStats struct {
    ReceivedBytes uint64  `json:"received_bytes"`
    SentBytes     uint64  `json:"sent_bytes"`

    ReceiveBPS    float64 `json:"receive_bps"`
    SendBPS       float64 `json:"send_bps"`
}
```

放在：

```go
OSStats
```

中：

```go
Network NetworkStats `json:"network"`
```

---

## 14.2 注意

默认展示所有网络接口汇总。

暂不做：

```text
per NIC
eth0
ens3
lo
docker0
```

避免 UI 和 JSON 复杂化。

Loopback 是否包含应明确记录在文档中。

如果 gopsutil 跨平台行为存在明显差异，v1.2 可先实现 aggregate total。

---

# 15. UI 改进

## 15.1 首页信息层级

建议顶部：

```text
Monitor

sagaflow-api
production · v0.8.1 · Go 1.xx

LIVE
Uptime 3d 12h
```

---

## 15.2 HTTP 卡片

新增：

```text
RPS
Error Rate
P50
P95
P99
In Flight
```

建议主信息：

```text
128.4 req/s
42 ms P95
0.2% 5xx
```

---

## 15.3 Collection 状态

当前：

```text
LIVE / STALE / ERROR
```

继续保留。

新增：

```text
PARTIAL
```

语义：

```text
LIVE     Snapshot 正常且新鲜
STALE    Snapshot 过旧
PARTIAL  Snapshot 新鲜，但部分 Collector 失败
ERROR    JSON 请求或解析失败
```

---

## 15.4 Container 卡片

仅：

```text
container.detected == true
```

时显示。

示例：

```text
Container

Memory
1.82 / 2.00 GB
91%

CPU Limit
2 cores
```

非容器环境完全隐藏该 section。

---

## 15.5 N/A

采集失败时：

```text
N/A
```

不要显示：

```text
0%
0 B
0 ms
```

---

# 16. API Compatibility

v1.2 应保持向后兼容。

---

## 16.1 保留

必须保留：

```go
monitor.New(...)
monitor.NewMonitor(...)
Monitor.Current()
Monitor.Stop()
Monitor.RequestStarted()
Monitor.RequestFinished()
Monitor.ObserveRequest()
```

---

## 16.2 JSON

现有字段不删除：

```text
pid
runtime
os
http
```

以及：

```text
http.latency.last_ns
http.latency.recent_ns
http.latency.max_ns
```

只允许新增字段。

---

## 16.3 Config

现有 Config 字段行为不破坏：

```text
Path
Title
Description
Footer
FaviconURL
DefaultLanguage
DefaultTheme
Background
DefaultSampleWindow
DiskPaths
Refresh
APIOnly
IgnoreRequest
```

新增：

```text
ServiceName
Version
Environment
Authorize
```

---

# 17. 建议内部代码结构调整

目前代码量已经开始增长，v1.2 建议轻微整理，但不要过度工程化。

推荐：

```text
monitor.go
config.go
stats.go

collect.go
collect_process.go
collect_runtime.go
collect_os.go
collect_container_linux.go
collect_container_other.go

http_metrics.go
histogram.go

buildinfo.go

html.go

internal/ui/
```

---

## 17.1 Build Tags

Container：

```text
collect_container_linux.go
collect_container_other.go
```

例如：

```go
//go:build linux
```

避免 Windows 编译碰到 Linux 专用代码。

---

# 18. Benchmark

v1.2 的 HTTP histogram 不应显著影响 hot path。

---

## 18.1 必测 Benchmark

继续保留：

```text
BenchmarkDirectHandler
BenchmarkMonitorBusinessRequest
BenchmarkMonitorBusinessRequestWrite
BenchmarkMonitorBusinessRequestParallel
BenchmarkMonitorIgnoredRequest
BenchmarkMonitorObserveRequest
BenchmarkMonitorManualRequestLifecycle
BenchmarkMonitorJSONSnapshot
BenchmarkMonitorHTMLPage
BenchmarkMonitorCurrent
```

新增：

```text
BenchmarkMonitorBusinessRequestHistogram
BenchmarkMonitorBeginRequest
```

---

## 18.2 性能目标

建议不规定绝对 ns/op，因为 GitHub Runner 和机器差异较大。

关注：

```text
allocs/op
相对于 v1.1.x 的回归比例
```

建议目标：

```text
Business Request:
0 或极低额外 allocation

v1.2 相比 v1.1.x：
hot path 延迟增长尽量 < 20%
```

如果 histogram 造成明显性能下降，应优先优化实现而不是增加指标。

---

# 19. Tests

## 19.1 Snapshot

测试：

```text
schema_version
collected_at
collection.duration
partial
errors
```

---

## 19.2 RPS

使用可控时间或 helper 测：

```text
0 req/s
1 req/s
burst
collector interval jitter
```

不要依赖真实 sleep 编写脆弱测试。

建议抽象：

```go
now func() time.Time
```

为内部测试提供 fake clock。

若认为侵入过大，也可拆分 RPS 纯函数测试。

---

## 19.3 Histogram

必须测试边界：

```text
0
1ms
5ms
10ms
...
5s
>5s
```

测试：

```text
P50
P95
P99
empty histogram
single sample
high concurrency
window reset
```

---

## 19.4 Lifecycle

必须测试：

```text
RequestFinished without Started
double Finished
BeginRequest double finish
parallel Start / Finish
```

---

## 19.5 Authorization

测试：

```text
Authorize == nil
Authorize allow
Authorize deny
HTML
JSON Accept
HEAD
POST
```

---

## 19.6 Container

Linux：

```text
normal cgroup v2
memory.max == max
invalid file
missing file
malformed value
```

建议将文件读取逻辑拆成可注入 path/root：

```go
collectCgroup(root string)
```

生产传：

```text
/sys/fs/cgroup
```

测试传 temp dir。

---

# 20. Race Test

CI 增加：

```bash
go test -race ./...
```

重点验证：

```text
snapshot atomic.Value
histogram swap
request counters
BeginRequest
collector Stop
```

---

# 21. CI Matrix

当前项目属于跨平台系统信息采集库，建议 v1.2 CI：

```text
Ubuntu
Windows
```

Go：

```text
最低支持版本
当前稳定版本
```

如果继续保持：

```text
Go 1.24+
```

建议：

```yaml
matrix:
  os:
    - ubuntu-latest
    - windows-latest
  go:
    - "1.24.x"
    - "stable"
```

Race：

```text
ubuntu-latest
stable
```

单独 job 即可。

---

# 22. README 更新

v1.2 README 应新增：

```text
Service Info
RPS
Error Rate
Latency Percentiles
Container Metrics
Access Control
Partial Collection
```

---

## 22.1 Scope

继续明确：

`monitor` does:

```text
embedded real-time status
current process/runtime/host/container snapshot
HTTP realtime metrics
JSON API
short browser charts
```

`monitor` does not:

```text
long-term history
alerting
multi-node aggregation
tracing
logs
business metrics platform
plugin system
```

---

# 23. Security 文档

SECURITY.md 建议补充：

Monitor 页面可能暴露：

```text
process state
runtime usage
resource usage
disk mount information
request traffic
service version
revision
environment name
```

推荐用户：

```text
不要直接公开暴露到 Internet
```

或使用：

```text
Authorize
Reverse Proxy Auth
Cloudflare Access
Internal Network
VPN
IP Allowlist
```

---

# 24. 实施顺序

建议严格按下面顺序开发。

---

## Phase 1：基础数据模型

任务：

- [ ] 增加 `StatsSchemaVersion`
- [ ] 增加 `CollectedAt`
- [ ] 增加 `CollectionStats`
- [ ] 增加 partial/error 机制
- [ ] 更新 JSON tests
- [ ] UI 支持 PARTIAL / N/A

完成后：

```text
Snapshot 模型稳定
```

再继续其他功能。

---

## Phase 2：HTTP Realtime Metrics

任务：

- [ ] RPS
- [ ] interval status delta
- [ ] 4xx rate
- [ ] 5xx rate
- [ ] error rate
- [ ] latency histogram
- [ ] P50
- [ ] P95
- [ ] P99
- [ ] recent max
- [ ] histogram benchmark
- [ ] concurrency tests

这是 v1.2 最大工作包。

---

## Phase 3：Service Info

任务：

- [ ] Config.ServiceName
- [ ] Config.Version
- [ ] Config.Environment
- [ ] `runtime.Version`
- [ ] `debug.ReadBuildInfo`
- [ ] VCS revision
- [ ] UI header/service section
- [ ] README example

---

## Phase 4：安全与运行时修复

任务：

- [ ] Config.Authorize
- [ ] 401 处理
- [ ] Refresh 最小值
- [ ] RequestFinished underflow fix
- [ ] BeginRequest API
- [ ] Tests

---

## Phase 5：Container

任务：

- [ ] cgroup v2 detection
- [ ] memory.current
- [ ] memory.max
- [ ] cpu.max
- [ ] linux build tags
- [ ] other OS stub
- [ ] UI Container section
- [ ] tests

---

## Phase 6：Network

任务：

- [ ] network total RX
- [ ] network total TX
- [ ] RX BPS
- [ ] TX BPS
- [ ] UI
- [ ] tests

若 v1.2 开发周期需要进一步压缩：

```text
Network IO 可以作为最后一个可裁剪功能。
```

其他工作包优先级更高。

---

## Phase 7：CI / Release

任务：

- [ ] Windows CI
- [ ] stable Go CI
- [ ] race
- [ ] benchmark baseline
- [ ] README
- [ ] 中文 README
- [ ] SECURITY
- [ ] CHANGELOG / Release Notes
- [ ] tag v1.2.0

---

# 25. v1.2 Definition of Done

只有以下条件全部满足，才建议发布 `v1.2.0`。

## Functional

- [ ] `/monitor` HTML 正常
- [ ] JSON Snapshot 向后兼容
- [ ] Schema Version 正常
- [ ] Collection 状态可判断
- [ ] RPS 正常
- [ ] Error Rate 正常
- [ ] P50/P95/P99 正常
- [ ] Recent Max 正常
- [ ] Service Info 正常
- [ ] Access Control 正常
- [ ] Refresh clamp 正常
- [ ] in-flight 不会 underflow
- [ ] cgroup v2 Linux 正常
- [ ] 非 Linux 编译正常
- [ ] Network IO 正常或明确裁剪

## Quality

- [ ] `gofmt`
- [ ] `go vet ./...`
- [ ] `go test ./...`
- [ ] `go test -race ./...`
- [ ] Linux CI
- [ ] Windows CI
- [ ] Minimum Go CI
- [ ] Stable Go CI
- [ ] benchmark 无明显回退

## Documentation

- [ ] English README
- [ ] Chinese README
- [ ] Config table
- [ ] JSON example
- [ ] Access Control example
- [ ] Container behavior
- [ ] Security note
- [ ] Release Notes

---

# 26. 建议 v1.2 Release Notes

```markdown
## monitor v1.2.0

v1.2 improves production visibility while keeping monitor lightweight and dependency-free.

### Added

- Snapshot schema version and collection metadata.
- Partial collection status for unavailable metrics.
- HTTP requests-per-second metrics.
- Recent HTTP 4xx / 5xx / error rates.
- Approximate P50 / P95 / P99 request latency.
- Recent maximum request latency.
- Service and Go build information.
- Optional request authorization hook for the monitor endpoint.
- Linux cgroup v2 container memory and CPU limit metrics.
- Network receive / transmit metrics.
- Safer framework-neutral request lifecycle helpers.

### Improved

- Monitor UI now distinguishes unavailable metrics from real zero values.
- Added production-oriented service and HTTP summary information.
- Refresh intervals are clamped to a safe minimum.
- Improved CI coverage with race and cross-platform testing.

### Fixed

- Prevented in-flight request counter underflow when request lifecycle methods are misused.

### Scope

monitor remains a lightweight embedded status page.

It does not provide long-term storage, alerting, tracing, multi-instance aggregation, or a plugin system.
```

---

# 27. v1.2 完成后的项目定位

v1.2 完成后，`monitor` 的定位建议统一描述为：

> A lightweight embedded live status page for Go services.

或者更完整：

> `monitor` is a lightweight Go middleware that embeds a production-oriented live status page directly into your service, with process, runtime, system, container, and HTTP metrics and no external monitoring stack required.

核心竞争点：

```text
轻量
嵌入式
零外部服务
零前端构建
开箱即用
生产环境实时状态
Go 原生
```

而不是：

```text
Prometheus replacement
Grafana replacement
Observability platform
```

---

# 28. 后续版本边界

完成 v1.2 后建议暂时进入：

```text
v1.2.x
```

维护阶段。

适合 v1.2.x 的内容：

```text
Bug Fix
跨平台兼容
UI 小改进
指标准确性修复
性能优化
文档完善
```

不要因为小功能快速推进：

```text
v1.3
v1.4
v1.5
```

只有出现新的明确能力阶段时再提升 minor version。

---

# 29. v2 暂存议题

以下内容不进入 v1.2：

```text
Custom Collector
Plugin / Extension API
Custom Section
Redis / SQL / Kafka metrics
Exporter
Multi-instance
Persistent History
Alerting
```

未来如果进入 v2，应优先考虑：

```text
稳定扩展接口
```

而不是直接把各种依赖集成进核心。

推荐方向：

```text
monitor core
    +
optional collectors
```

核心包继续保持最小依赖与高稳定性。

---

# 30. 最终实施重点

整个 v1.2 的开发过程中，需要始终优先保证：

```text
正确性 > 指标数量
实时价值 > 历史复杂度
低开销 > 功能堆叠
通用接口 > 框架绑定
可维护性 > 过度抽象
```

v1.2 最重要的成果不是“增加多少个指标”，而是让用户打开 `/monitor` 后，可以在几秒钟内回答：

```text
这个服务是谁？
运行的是哪个版本？
服务活着多久了？
当前请求压力多大？
错误率有没有异常？
P95 / P99 有没有升高？
CPU / 内存是否异常？
是否接近容器限制？
指标是否采集完整？
```

做到这些以后，`monitor` 就已经具备非常明确且实用的生产环境价值，同时仍然保持它作为一个轻量 Go middleware 的核心特色。
