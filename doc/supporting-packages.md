# 支撑包

本文件覆盖 `ae`、`alog`、`acontext`、`helpers/debug` 与 `pkg/*`。

## 1. 错误（aa/ae）

### 1.1 错误码

`ae/codes.go` 中的常量与 HTTP 状态码对齐，并扩展了业务码：

| 区间 | 示例 | 说明 |
|------|------|------|
| 1xx–5xx | `OK=200`、`Created=201`、`BadRequest=400`、`NotFound=404`、`InternalServerError=500` | 标准状态码 |
| 3xx 扩展 | `FailedAndSeeOther=391` | 失败但需跳转 |
| 4xx 扩展 | `PageExpired=419`、`NoRowsAvailable=494`、`ConflictWith=499` | 令牌过期、空列表、冲突数据 |
| 5xx 扩展 | `Exception=590` | 未捕获异常 |

`ae.Text(code)` 返回标准文案。

### 1.2 Error

```go
type Error struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	Caller    string `json:"caller"`
	Detail    string `json:"details"`
	TraceInfo string `json:"trace_info"`
	locked    bool
}
```

| 方法 | 说明 |
|------|------|
| `New(code, msg...)` | 自动记录调用者；未给 msg 时用 `Text(code)` |
| `NewF(code, format, args...)` | 格式化 |
| `NewError(msg)` / `NewErrorf` | 500 级错误 |
| `NewErr(err, detail...)` | 包装标准 error |
| `NewBadParam(param, tips...)` | 400，消息包含参数名 |
| `NewUnsupportedMedia(wants...)` / `NewPreconditionFailed` / `NewNotAcceptable` / `NewConflict` / `NewConflictWith` / `NewParamTooShort` / `NewParamTooLong` | 常用语义化构造 |
| `WithMsg` / `WithMsgf` / `AppendMsg` | 修改/追加消息 |
| `WithDetail` / `WithCaller(skip)` / `WithTraceInfo(ctx)` | 补充上下文 |
| `Lock()` / `Unlock()` / `Clone()` | 常数错误冻结；修改已冻结错误会自动克隆 |
| `String()` / `Text()` / `Trace(ctx)` | 字符串输出（不含 caller / 含 caller / 含 trace） |
| `IsNotFound()` / `IsServerError()` / `IsFailedAndSeeOther()` / `ExceptNotFound()` | 判定 |

设计约束：

- `*ae.Error` **不实现** `error` 接口，避免与标准 error 混用；需要 error 时用 `NewErr` 方向的工具或 `errors.New(e.String())`。
- 包级常数错误在 init 时创建并 `Lock()`，可安全共享。
- `New` 会采样调用栈（`utils.Caller`），**不要在高频成功路径上创建错误**。

### 1.3 包级工具

`First` / `FirstError`（取第一个非空）、`IsNotFound(err error)`（兼容标准 error）、
`Panic` / `PanicF` / `PanicOn` / `PanicOnErrs` / `PanicWithCaller`、
`StopOnFirstError` / `StopOnFirstE`、`Wrap`、`ErrInvalid` / `ErrorInvalid` / `BadConfig`。

`ae.Separator`（`" - "`）用于拼接消息。

## 2. 日志（aa/alog）

### 2.1 级别

```go
LevelErrAll < LevelDebug < LevelInfo < LevelNotice < LevelWarn < LevelError < LevelFatal < LevelAlert < LevelEmerg
```

`NewDefaultLog(level)` 只输出 `level <= 配置级别` 的日志。

### 2.2 接口

```go
type LogInterface interface {
	New(prefix string, f func(context.Context, string, ...any), suffix ...string) func(context.Context, string, ...any)

	Debug/Info/Notice/Warn/Error/Fatal/Alert/Emerg(ctx, msg)
	Debugf/Infof/Noticef/Warnf/Errorf/Fatalf/Alertf/Emergf(ctx, format, args...)
	E(ctx context.Context, e *ae.Error, msg ...any)

	Print/Printf(ctx, ...)
	Trace(ctx)
}
```

- 所有方法都接收 `context.Context`，默认实现会从 context 中取 `TraceInfo` 拼到行尾。
- 默认实现基于标准库 `log`；`E` 使用 `*ae.Error` 自带的 `Caller`，避免重复采样。
- **禁用级别不会采样调用栈**（先判断级别，再计算 caller），生产环境把级别调到 `LevelInfo` 即可省掉这部分开销。

### 2.3 包级辅助

`Console`、`Print/Printf`、`Warn/Warnf`、`Error/Errorf`、`OnError(err)`、`LogOnE(e)`、`Start/Startf/Stop/Stopf`（带 `StartPrefix` 的阶段日志）。

### 2.4 自定义实现

```go
type zapLog struct{ /* ... */ }

app := aa.New(ctx, cancel, cfg).WithLog(zapLog{})
```

只要实现 `LogInterface` 即可；`App.Check` / `App.CheckErrors` 会用它输出服务端错误。

## 3. 请求上下文（aa/acontext）

### 3.1 Context 别名

`acontext.Context` 是 `context.Context` 的别名，并提供 `Background/TODO/WithCancel/WithTimeout/WithDeadline/WithValue/WithoutCancel/AfterFunc/Cause` 等包装，统一代码风格。

### 3.2 键

| 常量 | 值 |
|------|----|
| `CtxRequestID` | `aa.request.id` |
| `CtxTraceId` | `aa.trace.id` |
| `CtxClientIP` | `aa.client.ip` |
| `CtxRemoteAddr` | `aa.remote.addr` |
| `CtxRemoteUser` | `aa.remote.user` |
| `CtxContentType` | `aa.content.type` |

### 3.3 Bind 与取值

```go
stdCtx := c.StdContext()          // 内部调用 acontext.Bind(c)
id := acontext.TraceID(stdCtx)
ip := acontext.RemoteAddrValue(stdCtx)
user := acontext.RemoteUserValue(stdCtx)
```

`Bind` 把 `trace id / 客户端 IP / 用户` 打包成**一个** context value，避免多次 `WithValue` 分配；它同时把新 context 写回 `http.Request`（`ResetRequest`）。

`ClientIP` 的取值顺序：`X-Forwarded-For` 第一个非空项 → `X-Real-IP` → `RemoteAddr`，并缓存在值袋 `CtxClientIP`。

`WithTrace(parent)` 用调用者位置生成 trace id（`file:line.func`），适合非 HTTP 场景。

`WithRemoteUser(ctx, user)` 泛型写入用户标识。

### 3.4 健康状态

```go
acontext.SetServStarting()
acontext.SetServReady()
acontext.SetServStopping()
acontext.SetServUnhealthy()
acontext.ServFallbackReady()   // 仍为 starting 时才置 ready

acontext.ServHealth().IsReady()
```

基于 `atomic.Value`；适合 `/healthz` 与优雅退出。

### 3.5 RequestCarrier

`httpsvr.Context` 通过实现 `RequestCarrier`（`GetHeader/RemoteAddr/Request/ResetRequest/ValueString/SetValue`）与 `acontext` 解耦，避免循环依赖。

## 4. 调试（aa/helpers/debug）

- `NewLogWriter(dir, perm, bufSize, symlinks...)`：把标准库 `log` 输出重定向到文件（按大小切分、可选软链），`RedirectLog` 是其快捷版本。
- `Profile`：轻量阶段计时。

```go
prof := debug.NewProfile("boot")
defer prof.Stop("done")
prof.WithLabel("db").Start("connect")
```

`Profile` 会输出每段耗时（阈值 1000ms 内按毫秒格式化），`IsDisabled()` 可用于跳过昂贵统计。

## 5. 工具包（pkg/）

### 5.1 `pkg/types`

| 能力 | 代表 API |
|------|----------|
| 整数格式化（小整数查表） | `FormatInt`、`FormatUint`、`FormatUint8`、`Itoa` |
| 整数解析 | `ParseInt8/16/32/64`、`ParseUint8/.../64`、`ParseSigned[T]`、`ParseUnsigned[T]` |
| 自定义 64 进制 | `Base64Digits`、`FormatBase64Uint`、`ParseBase64Uint`、`ConvertBase` |
| 浮点 | `FormatFloat`、`ToFloat32/64` |
| 空值判断 | `IsEmpty`、`IsNil`、`NotEmpty` |
| 类型工具 | `PType`、`PrimitiveType`、`Ref`、`RefNz`、`Deref` |
| 时长 | `ParseDuration`、`StringifyDuration` |

> 小整数（0–255）解析/格式化走查表快路径；解析时**同时校验 base 与 bitSize**，越界仍返回 `strconv` 错误。

### 5.2 `pkg/afmt`

字符串补齐/裁剪（`PadLeft/PadRight/PadBoth/Trim`）、控制台颜色（`PrintRed/PrintGreen/WithStyle/PrintBorder`）、时间差（`TimeDiff`、`DurationInChinese`、`DurationString`）、路径/URL（`FilePath`、`DirPath`、`Url`）、参数处理（`First`、`FirstNotEmpty`、`SprintfArgs`、`DefaultIfZero`）。

### 5.3 `pkg/arrmap`

`Keys`、`SortedKeys`、`KeysFunc`、`Merge`、`Compact`、`HasDuplicates`、`JoinKeys`、`BuildSliceMap`，以及进制校验 `IsBase` / `PanicIfNotBase`。

### 5.4 `pkg/httpc`

轻量 HTTP 客户端：`Get/Post/Put/Patch/Delete`、`*Json` 变体、`Request`、`RequestJson`、`RequestJsonBody`、`SetQueries`、`SetHeaders`、`AddHeaders`、`SetNxHeader`。

- 所有请求都通过 `http.NewRequestWithContext` 绑定调用方 context（`nil` 视为 `context.Background()`）。
- 复用一个包级 `http.Transport`（`MaxIdleConns=100`、`MaxIdleConnsPerHost=10`、`IdleConnTimeout=90s`），默认开启连接复用。
- `DefaultRequestTimeout`（10s）每次请求读取；`DefaultDialTimeout`（5s）在首次使用时读取。
- 多个 header map 会按顺序合并（`SetHeaders` 覆盖、`AddHeaders` 追加）。
- `*Json` 变体把响应解码到调用方传入的 `target`（需为指针）；`JsonUnmarshal` 可替换为自定义解码器。

### 5.5 其它

- `pkg/ios`：`MkdirAll`、`PrepareFile`。
- `pkg/nets`：`LanIP`、`LocalPublicIP`（带缓存与 UDP 探测）。
- `pkg/utils`：`Caller(skip)`（跳过 aago 帧）、`Close`（批量关闭）、`GitVersion`、`Dup2`（平台实现）。
- `pkg/basic`：`Ter(cond, a, b)` 三元表达式。

## 6. 依赖方向速查

```
pkg/*  ←  aa/*  ←  业务代码
```

`pkg/*` 不允许 import `aa/*`；`aa/ae`、`aa/alog`、`aa/acontext`、`aa/atype` 不应 import `aa/httpsvr`。
