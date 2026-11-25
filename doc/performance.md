# 性能特征与优化记录

## 1. 基准环境与方法

- 机器：Intel Core i5-10400（12 线程），Windows，Go 1.27。
- 命令：

```bash
go test ./aa/httpsvr -run '^$' -bench . -benchmem -count=5
```

- `-benchmem` 报告每次操作的字节数与分配次数。
- 机器为共享环境，绝对数值有 ±10% 波动；关注**同一轮次内的相对变化**与分配次数的下降。
- 基准文件：[router_bench_test.go](../aa/httpsvr/router_bench_test.go)、[response_bench_test.go](../aa/httpsvr/response_bench_test.go)、[stress_test.go](../aa/httpsvr/stress_test.go)。

## 2. 优化前后对比

| 基准 | 优化前 (ns/op) | 优化后 (ns/op) | 分配次数 | 内存 |
|------|----------------|----------------|----------|------|
| `HTTPXJSONEcho` | ~39 200 | ~3 850 | 71 → **32** | 6922 B → **2508 B** |
| `RouterMultiParam` | ~2 460 | ~1 384 | 18 → **14** | 1377 B → **1193 B** |
| `RouterTypedUint64` | ~1 627 | ~1 434 | 20 → **16** | 1321 B → **1185 B** |
| `RouterParallel` | ~951 | ~702 | 18 → **14** | 1298 B → **1162 B** |
| `RouterManyRoutes`（200 条） | ~2 956 | ~3 119 | 10 → **8** | 472 B → **376 B** |
| `RouterStatic` | ~1 308 | ~1 206 | 17 → **13** | 1249 B → **1129 B** |
| `HTTPXPing` | — | ~1 078 | 13 | 1129 B |
| `HTTPXFormLogin` | — | ~8 489 | 53 | 8774 B |

> `RouterManyRoutes` 的耗时在噪声范围内持平，但分配次数从 10 降到 8；其余场景耗时与分配同时下降。

`HTTPXJSONEcho` 从 39µs 降到 3.8µs，主要不是"微优化"，而是修掉了一个**每请求都会创建并丢弃一个 400 错误**的缺陷（见 3.1），`ae.New` 的调用栈采样代价极高。

## 3. 已实施的优化

### 3.1 修复路径参数注入时机（正确性 + 性能）

**问题**：`acquireContext` 在路由匹配前调用 `resolveErrorAsStatus` → `QueryBool` → `ensurePathParams`，此时 `c.params` 为空，却把 `paramsInjected` 置为 `true`。匹配成功后再也不会注入路径参数，导致：

- `QueryUint64("uid")` 读不到 `/users/{uid}` 的路径参数；
- 每次请求都生成一个 `400 Bad Parameter`（`ae.New` 内部 `utils.Caller` 会遍历调用栈），这是 39µs 的主因。

**修复**：在 `dispatchContext` 设置 `c.params` 后重置 `paramsInjected`，让下一次查询重新注入。

### 3.2 静态路由精确索引

注册无参数路由时写入 `staticRoutes[method][规范化路径]`，匹配时 O(1)。同一路径重复注册时先注册者生效，与线性扫描顺序一致。

### 3.3 单次路径分段视图（pathView）

每个请求只构造一次 `pathView`：内联 `[16]int` 保存每段的起止下标，路由匹配直接切片比较，不再：

- 为每个请求切分 `[]string`（原实现）；
- 为每个候选路由重新扫描字符串（中间版本的回归）。

超过 16 段时自动回退到 `pathSegments`。

### 3.4 预构建 UseRouter 链

`UseRouter` 注册时构建 `routerFilters + serveMatched` 的完整链，请求期不再分配切片。

### 3.5 URL query 只解析一次

`url.Values` 缓存在 `Context`（`urlValues` / `urlValuesParsed`），`Query/QueryString/Queries/URLParam` 共用；`ResetRequest` 会使缓存失效。

### 3.6 避免成功路径上的错误分配

`atype.Bool("")` 直接返回 `false, nil`，不再让 `strconv.ParseBool("")` 生成 `*NumError`。`QueryBool` 在参数缺失时不再分配错误。

### 3.7 日志按级别短路

`alog` 之前即使级别被禁用也会先 `utils.Caller(1)` 采样调用栈。现在先判断级别，再采样（`xlog.log`）。

### 3.8 整数解析快路径修正

`types.ParseSigned/ParseUnsigned` 的小整数查表之前忽略 `base` 与 `bitSize`：

- `ParseInt8("200")` 会静默返回 `-56`；
- `ParseSigned[int]("10", 2, 64)` 返回 10（应为 2）。

现在只在 `base == 10` 且位宽足够时走查表，其余走 `strconv`。

同时 [`parseUint64`](../aa/httpsvr/request_helper.go) 会把 `bitSize` 传给 `types.ParseUint64`，`QueryUint8("300")` 返回 400 而不是静默截断为 44。

### 3.9 `UnmarshalUint8s` 重写

原实现逐位拼接字符串、忽略解析错误、且对不完整 JSON（如 `[1`）会越界 panic。现改为按区间扫描 + `strconv.ParseUint`，并校验外层括号、支持 `null`/空数组。

### 3.10 httpc 连接复用

`pkg/httpc` 之前每个请求新建 `http.Transport` 并设置 `req.Close = true`，无法复用连接。现在共享一个 `http.Transport`（keep-alive、空闲连接池），并修正了 context 丢失、DELETE 误用 POST、JSON 解码目标错误、单 header map 越界等问题。

## 4. 剩余热点

对 `HTTPXJSONEcho` 做 `alloc_objects` 采样（已排除 httptest 自身）：

| 热点 | 占比 | 说明 |
|------|------|------|
| `newRawValue` | ~10% | 每次参数读取创建 `RawValue` + `Atype`；`Atype` 已池化，`RawValue` 未池化 |
| `encoding/json/v2` | ~12% | DTO 序列化的反射开销 |
| `RequestID` 中间件 | ~5% | 生成 32 字符 ID + Header.Set |
| `url.ParseQuery` | ~2% | 每请求一次 |
| `mime.ParseMediaType` | ~2% | `ReqContentType` |

可继续优化的方向（按收益排序）：

1. 给 `RawValue` 增加对象池（需保证所有调用路径 `Close`）。
2. 对固定响应结构预生成序列化器，避免重复反射。
3. `RequestID` 改为 16 字节十六进制（已有实现即 32 字符，可按需缩短）。
4. 复用 `Accept` / `Content-Type` 的解析结果。

## 5. 业务侧调优建议

| 场景 | 建议 |
|------|------|
| 纯文本/健康检查 | 用 `c.WriteString` 或 `WriteRaw`，避免 DTO 序列化 |
| 可选参数 | 用 `QueryFast` / `BodyStringFast`，避免构造错误 |
| 必填参数 | 用 `QueryString(p, true)` / `QueryUint64(p)` |
| 正则校验 | 预编译 `*regexp.Regexp` 传入，避免每次 `regexp.Compile` |
| 大响应 | 开启 `X-Stringify` 会走反射遍历，按需开启 |
| 日志 | 生产环境 `NewDefaultLog(alog.LevelInfo)`，禁用级别不会采样调用栈 |
| 错误 | 不要在高频成功路径创建 `ae.Error`（会采样调用栈） |
| 出站请求 | 复用 `httpc` 的共享 Transport；不要每次 `http.Get` |

## 6. 已知限制与后续工作

1. **路由注册不是并发安全的**：必须在 `Listen` 之前完成注册；运行期加路由会产生数据竞争。
2. **超长路径回退**：超过 16 段的路由会使用切片匹配（每次请求一次分配），属罕见场景。
3. **`x-field` 过滤仅支持 struct / map**：对基础类型返回 400。
4. **`WriteData` 的装饰开销**：`x-stringify` / `x-field` 关闭时只做一次参数查询；开启后会做全量反射遍历。
5. **`atype.ToIP` 使用文本形式**：`OK()` 以 4/16 字节存储宽度判断，若调用方传入文本 IP，`Bytes()` 会返回 nil；需要二进制语义时请直接 `Scan([]byte(ip.To4()))`。

## 7. 回归防线

每次改动 HTTP 热路径后至少执行：

```bash
gofmt -l .
go build ./...
go test ./...
go vet ./...
go test ./aa/httpsvr -run '^$' -bench 'BenchmarkHTTPX|BenchmarkRouter' -benchmem -count=3
```

重点关注：`HTTPXJSONEcho`、`RouterStatic`、`RouterManyRoutes` 的 `ns/op` 与 `allocs/op` 不应回退。
