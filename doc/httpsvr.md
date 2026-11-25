# HTTP 框架（aa/httpsvr）

`aa/httpsvr` 是 AaGo 的核心：一个基于 `net/http` 的、Iris 风格的路由器 + 请求解析 + DTO 响应层。

```go
type Handler func(*Context)
```

所有中间件与控制器都是 `Handler`，共享同一个 `*Context`。

## 1. 组件一览

| 类型 | 作用 |
|------|------|
| `Router` | 路由表、匹配前过滤器（`UseRouter`）、错误处理器、HTTP Server 生命周期 |
| `Party` | 带公共前缀与中间件的路由组 |
| `Context` | 单请求全部能力：路径参数、查询/请求体/Header 解析、值袋、写出 |
| `Settings` | 每个 App 的响应策略（Content-Type、错误处理、序列化） |
| `Body` | DTO 响应体 `{code, msg, data}` |
| `Params` / `Param` | 路径参数集合 |
| `RawValue` | 解析中间值（包装 `*atype.Atype`） |
| `CORS` | 跨域中间件工厂 |

## 2. 路由注册

```go
rt := httpsvr.NewRouter()          // New() 等价
api := rt.Party("/v1", authMW)     // 前缀 + begin 中间件
api.Get("/users/{uid:uint64}", getUser, audit)
api.Post("/users", createUser)
rt.Any("/health", health)
rt.Handle("PURGE", "/cache", purge)
```

| 方法 | 说明 |
|------|------|
| `Get/Post/Put/Patch/Head/Delete/Options/Connect/Trace(path, handlers...)` | 注册单方法路由 |
| `Any(path, handlers...)` | 注册 9 个标准方法 |
| `Handle(method, path, handlers...)` | 任意方法；`method == ""` 等价于 `Any` |
| `Party(prefix, handlers...)` / `PartyFunc(prefix, fn)` | 路由组；子组继承父组的 begin/done |
| `Use(handlers...)` | 追加 begin 中间件（匹配成功后执行） |
| `UseOnce(handlers...)` | 按函数指针去重/替换 |
| `Done(handlers...)` | 追加 end 中间件（链尾，前面的 handler 必须 `Next()`） |
| `Reset()` | 清空该 Party 的 begin/done |
| `UseRouter(handlers...)` | 匹配**之前**执行，404/405 也会执行 |
| `OnErrorCode(status, handlers...)` / `OnAnyErrorCode(...)` | 404/405 等状态的自定义处理 |
| `UseHooks(v any)` | 把值（通常是 `*Settings`）挂到每个请求的 `ctx.Hooks()` |
| `SetViews(dir)` | 解析目录下的模板，供 `ctx.View(name)` 使用 |
| `Listen(addr)` / `Run(addr)` / `Shutdown(ctx)` | 启动与优雅退出 |

### 路径模式

- `"/users/{uid}"`：字符串参数（非空）。
- `"/users/{uid:uint64}"`：带类型校验与转换。
- 支持类型：
  `string`、`bool`、`int`、`int8`、`int16`、`int32`、`int64`、`uint`、`uint8`、`uint16`、`uint32`、`uint64`、`alphabetical`、`uuid`、`email`、`mail`、`weekday`。
- 前导/尾随 `/` 会被规范化：`/users/` 与 `/users` 等价。
- 路由按注册顺序优先；**静态路由优先命中**（注册时建立精确索引）。
- 同一 method + 同一静态路径重复注册时，先注册者生效。

### 匹配与 404/405

1. 先查 `staticRoutes[method][规范化路径]`，命中即执行。
2. 否则构造一次 `pathView`（内联保存最多 16 段边界），按注册顺序比较带参数路由。
3. 仍未命中时，扫描其他 method：若存在路径匹配，返回 **405** 并设置 `Allow`，否则 **404**。
4. 404/405 会优先使用 `OnErrorCode` 注册的处理器，否则 `http.Error` 输出状态文本。

> 超过 16 段的长路径自动回退到切片匹配（`pathSegments`），行为一致。

## 3. 中间件顺序

```
UseRouter（匹配前，含 404/405）
    → Use / Party(prefix, mw...)（begin，匹配成功后）
    → 路由 handlers（按声明顺序）
    → Done（链尾）
```

示例：

```go
rt.UseRouter(httpsvr.Recover, httpsvr.RequestID) // 1
rt.Use(httpsvr.CORS(opt))                        // 2
rt.Get("/x", h1, h2)                             // 3, 4
rt.Done(audit)                                   // 5
```

### `Next` 语义

```go
func (c *Context) Next() {
	if c.stopped {
		return
	}
	c.index++
	if c.index < len(c.handlers) {
		c.handlers[c.index](c)
	}
}
```

- 链式执行是**线性**的：`handler` 调用 `c.Next()` 才会进入下一个。
- 因此"前置逻辑"写 `c.Next()` 之前，"后置逻辑"写在之后。
- `c.StopExecution()` 后 `Next()` 变成空操作，剩下的 handler（含 `Done`）全部跳过。
- 常见写法：`defer c.Next()`（即使提前 return 也继续链路）。

```go
func (c *Context) AccessLog() {
	start := time.Now()
	c.Next()                       // 执行后续
	log.Println(time.Since(start)) // 之后
}
```

`httpsvr` 内置中间件：

| 中间件 | 行为 |
|--------|------|
| `Recover` | 捕获 panic → 500 + 日志 |
| `RequestID` | 读取 `X-Request-Id`，缺失则生成 32 位十六进制 ID，写入 Header 与值袋 `RequestIDKey` |
| `AccessLog` | 方法、路径、状态、耗时 |
| `CORS(opt)` | 跨域头 + OPTIONS 预检（204） |

## 4. Context 能力

### 4.1 请求信息

`Method()`、`Path()`、`Host()`、`FullRequestURI()`、`Origin()`、`RemoteAddr()`、`UserAgent()`、`ReqContentType()`、`Request()`、`ResetRequest(r)`、`ResponseWriter()`。

### 4.2 值袋与标准 context

```go
c.SetValue("trace.user", uid)      // 等价 c.Values().Set(...)
c.ValueString("trace.user")
stdCtx := c.StdContext()           // 绑定请求的 context.Context（懒创建）
```

`c.StdContext()` 由 `acontext.Bind(c)` 创建，携带请求 ID、trace、客户端 IP 等，可传给 service / repository。

### 4.3 路径参数

```go
uid, ok := c.Params().GetUint64("uid")
sid := c.Params().GetString("sid")
c.Params().Visit(func(k string, v any) { /* ... */ })
```

`Params` 提供 `Len/Get/GetEntry/GetString/GetInt/GetInt64/GetUint64/GetBool/Visit`。

### 4.4 路径参数与查询参数

匹配成功后，路径参数会被**注入**到查询参数集合（已存在的查询参数优先）。因此：

```go
rt.Get("/users/{uid:uint64}", func(c *Context) {
	uid, e := c.QueryUint64("uid") // 可读到路径参数 7
})
```

优先级：`c.InjectQuery(key, v)` > URL 路径参数 > URL query。

> URL query 每个请求只解析一次（缓存在 `Context` 上）；`ResetRequest` 会失效该缓存。

## 5. 参数解析

### 5.1 常用类型

| 来源 | 前缀方法 |
|------|----------|
| URL query / 路径参数 | `QueryBool/Bytes/String/Strings/Int/Int8/Int16/Int32/Int64/Ints/.../Uint64/Uint64s/Enum/Enum8/Enum8i/Valid/Valid8/Id` |
| 请求体（JSON / form / multipart） | `BodyBool/Bytes/String/Strings/Int.../Uint.../Enum.../AnyMap/AnySlice/StringMap/StringsMap/Float64Map/ComplexMaps/...` |
| Header | `HeaderParam/HeaderString/HeaderFast/HeaderValue/HeaderWild` |
| Cookie | `Cookie/GetCookie/SetCookie/Cookies` |

补充：

- 数值参数按目标类型位宽校验：`QueryUint8("300")`、`BodyInt8("200")` 返回 `400`，不会静默截断。
- `QueryFast(p)` / `BodyStringFast(p)`：忽略错误、`required=false`，用于可选值。
- `QueryWild(key, patterns...)`：依次查 query、Header、Cookie；`QueryWildFast` 是字符串快路径。
- `URLParam(key)`：只查原始 URL query（不查注入值）。

### 5.2 校验规则（patterns）

`QueryString(p, patterns...)`、`BodyString(p, patterns...)` 等接受可变规则：

| 规则 | 含义 |
|------|------|
| `bool` | 是否必填；缺省为 `true` |
| `string` | 正则表达式；取第一个非空字符串 |
| `*regexp.Regexp` | 编译好的正则，避免重复编译 |
| `*atype.Atype` / `atype.Atype` | 默认值：字段为空时填充 |

```go
name, e := c.QueryString("name", true, `^[\p{Han}A-Za-z ]{2,20}$`)
page, e := c.QueryUint64("page", false)                    // 可选，缺省 0
sex, e  := c.QueryString("sex", false, atype.New("male"))  // 默认 "male"
```

规则：

- 必填但为空 → `400 Bad Parameter`。
- 正则不匹配 → `400 Bad Parameter`。
- 参数名会写入错误消息，便于定位。

### 5.3 请求体

- 解析是**惰性**的：第一次调用任意 `Body*` 时读取并缓存。
- `Content-Type` 分派：
  - 空 / `application/json` / `application/octet-stream` → 读入后按 JSON 解析；
  - `application/x-www-form-urlencoded` → `url.ParseQuery`；
  - `multipart/form-data` → `ReadForm`，文件放 `injectedFiles`；
  - 其他类型不解析（返回空）。
- 默认限制：表单/JSON `1 MiB`（`SetMaxFormSize`），multipart `5 MiB`（`SetMaxMultipartSize`）。
- 超限返回 `413`；Body 为空时返回零值（必填才报错）。

```go
type CreateUser struct {
	Name string `json:"name"`
	Age  uint8  `json:"age"`
}
user, e := c.BodyStringMap("user", true)   // 简单字段
obj, e  := c.BodyAnyMap("profile", false)  // 嵌套结构
```

## 6. 响应

### 6.1 统一 DTO

```go
type Body struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}
```

| 方法 | 结果 |
|------|------|
| `WriteData(v)` | `{code:200, msg:"OK", data:v}` |
| `WriteOK()` | `{code:200, msg:"OK", data:null}` |
| `WriteCode(code)` | `{code, msg: ae.Text(code), data:null}` |
| `WriteMsg(code, msg)` / `WriteMsgf` | 自定义消息 |
| `WriteE(e)` / `WriteErr(err)` | 错误 DTO |
| `TryWrite(v, e)` / `TryWriteId(...)` | 有错写错误，无错写数据 |
| `WriteCreated(bytes)` / `WriteAccepted()` | 201 / 202 |
| `WriteRaw(ct, b)` / `WriteRawJSON/XML/HTML/OctetStream` | 原始响应，不包装 DTO |
| `JSON(v)` / `JSONP(v)` | 直接序列化任意值（不装饰） |
| `View(name)` | 渲染模板（需 `Router.SetViews`） |

`Body` 与 `Msg` 的语义由业务约定；框架只保证 envelope 结构。

### 6.2 Content-Type 协商

`ResolveContentType()` 的优先级：

1. `c.WithContentType(...)` / `c.WithHeader("Content-Type", ...)`
2. 值袋 `acontext.CtxContentType`
3. `Settings.ServeContentTypes`（缺省 `["application/json"]`）：
   - 按 `Accept` 的 q 值协商（支持 `*/*`、`type/*`）；
   - 再尝试请求的 `Content-Type`；
   - 都失败则取第一个可服务类型。

### 6.3 Settings（按 App 隔离）

```go
settings := httpsvr.NewSettings().
	SetErrorHandler(myErrorHandler)
settings.SetServeContentTypes([]string{"application/json", "text/html"})
settings.AddBeforeSerialize(func(ctx *httpsvr.Context, ct string, d httpsvr.Body) httpsvr.Body {
	d.Msg = "OK"
	return d
})
settings.SetSerialize(func(ct string, d httpsvr.Body) ([]byte, string, error) {
	return httpsvr.EncodeJson(d)
})

rt.UseHooks(settings)
```

`ErrorHandler` 签名：`func(ctx, contentType string, d Body) (n int, err error, next bool)`；`next=false` 表示已自行写出。

`BeforeSerialize` 在序列化前依次执行，可改写 DTO。

### 6.4 错误以 HTTP 状态码输出

三种开启方式（任一即可）：

1. 请求头 `X-Error-As-Status: true`
2. URL 参数 `x-error-as-status=true`
3. 服务端设置 `ctx.ErrorAsStatus = true`

开启后，`code >= 400` 的 DTO 会同时作为 HTTP 状态码；`1xx / 204 / 205 / 304` 不发送响应体。

### 6.5 bigint 字符串化与字段过滤

- `X-Stringify: true` 或 `x-stringify=true`：把响应中所有 `int64/uint64`（含指针、切片、map、结构体字段）转换为字符串，避免 JS 精度丢失。
- `x-field=a,b`：只返回结构体/ map 中 `a`、`b` 字段；`x-field=[a,b]`：对数组元素逐个裁剪。
- 只作用于 `WriteData` 路径（`decorateData`）；`JSON` 不装饰。

### 6.6 响应头

`WithHeader(k, v)`、`WithHeaders(map)`、`OutHeader(k)`、`DeleteOutHeader(k)`、`Header()`。

`WithHeader` 会忽略空的 `Last-Modified`（1970）与 `Content-Length`（由 net/http 计算）。

## 7. 错误与 NotFound

```go
rt.OnErrorCode(http.StatusNotFound, func(c *httpsvr.Context) {
	_, _ = c.WriteMsg(ae.NotFound, "resource not found")
})
```

- `Router.writeError` 先设置状态码；有注册处理器则执行，否则 `http.Error`。
- 路由层只产出 404/405；控制器里的错误请用 `ae` + `WriteE`。

## 8. 模板

```go
if err := rt.SetViews("./views"); err != nil { /* ... */ }

rt.Get("/page", func(c *httpsvr.Context) {
	c.ViewData("Title", "Home")
	_ = c.View("index.html")
})
```

`View` 会自动设置 `text/html; charset=utf-8`。

## 9. 常见陷阱

| 陷阱 | 后果 | 做法 |
|------|------|------|
| 忘记 `c.Next()` | 后续中间件/控制器不执行 | 用 `defer c.Next()` |
| 在 goroutine 里使用 `*Context` | 对象已被复用，数据错乱 | 只把 `StdContext()` 与纯数据传出去 |
| 运行期注册路由 | 数据竞争 | 启动前完成注册 |
| 用 `QueryFast` 处理必填参数 | 静默返回空串 | 必填参数用 `QueryString(p, true)` |
| 服务端直接抛 HTTP 状态码 | 网关/监控误判 | 默认用 body `code`，必要时才 `ErrorAsStatus` |
| `Done` 里做关键业务 | 前面 handler 没有 `Next()` 就不会执行 | 关键逻辑放 handler 或 begin 中间件 |

## 10. 测试与基准

```bash
go test ./aa/httpsvr -run TestRouter -v
go test ./aa/httpsvr -run '^$' -bench . -benchmem
```

已有基准：`router_bench_test.go`、`response_bench_test.go`。数据见 [performance.md](./performance.md)。
