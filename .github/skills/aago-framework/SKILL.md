---
name: aago-framework
description: Build, extend, review, or debug code that uses the AaGo Go framework (github.com/aarioai/aago). Use when writing routers, controllers, middleware, request parameter parsing, DTO responses, error codes, configuration, Consul registration, or when modifying the framework itself.
---

# AaGo Framework

AaGo 是一个基于标准库 `net/http` 的 Go 应用框架：Iris 风格路由 + PHP 风格 Controller + 统一 DTO/错误码 + YAML/Consul 配置。

## 何时使用

- 在当前仓库（`github.com/aarioai/aago`）新增或修改 `aa/*`、`pkg/*`。
- 用 AaGo 编写 HTTP 服务：路由、中间件、控制器、参数校验、响应。
- 排查 404/405、参数解析、响应 Content-Type、错误码、配置加载、Consul 注册问题。

## 五条核心规则

1. **Handler 是 `func(*httpsvr.Context)`**；要执行后续链路必须 `c.Next()`（推荐 `defer c.Next()`），中断用 `c.StopExecution()`。
2. **参数解析返回 `(value, *ae.Error)`**；错误用 `c.WriteE(e)` 或 `c.TryWrite(v, e)` 输出，不要直接 `http.Error`。
3. **必填是默认值**：`QueryUint64("uid")` 等于必填；可选要显式传 `false` 或用 `QueryFast`。
4. **业务错误默认放响应体** `{code,msg,data}`（HTTP 200）；只有开启 `ErrorAsStatus` 才映射为 HTTP 状态码。
5. **`*httpsvr.Context` 是池化对象**：不要跨 goroutine 使用；需要传递时用 `c.StdContext()` + 纯数据。

## 标准控制器模板

```go
func getUser(c *httpsvr.Context) {
	uid, e := c.QueryUint64("uid") // 路径参数 {uid:uint64} 也会被注入到 query
	if e != nil {
		_, _ = c.WriteE(e)
		return
	}
	user, e := userSvc.Get(c.StdContext(), uid)
	if e != nil {
		_, _ = c.WriteE(e)
		return
	}
	_, _ = c.WriteData(user)
}

func createUser(c *httpsvr.Context) {
	defer c.Next()
	name, e0 := c.BodyString("name", true, `^[\p{Han}A-Za-z ]{2,20}$`)
	age, e1 := c.BodyUint8("age", false)
	if e := c.FirstError(e0, e1); e != nil {
		return
	}
	_, _ = c.WriteData(map[string]any{"name": name, "age": age})
}
```

## 路由与应用装配模板

```go
cfg, err := aconfig.New("config/app.yaml", nil)
if err != nil {
	panic(err)
}
ctx, cancel := acontext.WithCancel(acontext.Background())
defer cancel()
app := aa.New(ctx, cancel, cfg)

rt := httpsvr.NewRouter()
rt.UseHooks(app.HTTP) // 注入 *httpsvr.Settings
rt.UseRouter(httpsvr.Recover, httpsvr.RequestID, httpsvr.AccessLog)

v1 := rt.Party("/v1", authMiddleware)
v1.Get("/users/{uid:uint64}", getUser)
v1.Post("/users", createUser)

rt.OnErrorCode(http.StatusNotFound, func(c *httpsvr.Context) {
	_, _ = c.WriteMsg(ae.NotFound, "resource not found")
})

if err := rt.Listen(":8080"); err != nil {
	app.Log.Error(ctx, err.Error())
}
```

## 参数速查

| 需求 | API |
|------|-----|
| query 必填/可选 | `QueryString(p, true/false)`、`QueryUint64(p, false)` |
| query 快路径（无错误） | `QueryFast(p)` |
| 正则 / 默认值 | `QueryString(p, true, "\\w+")`、`QueryString(p, false, atype.New("x"))` |
| 枚举 | `QueryEnum(p, required, []string{...})`、`QueryAtype.Sex(p)` |
| 请求体（JSON/form/multipart） | `BodyString`、`BodyAnyMap`、`BodyStrings`、`Files`、`File` |
| Header / Cookie | `HeaderString`、`HeaderFast`、`GetHeader`、`GetCookie` |
| 路径参数 | `c.Params().GetUint64("uid")`，或直接 `QueryUint64("uid")` |
| 分页 | `c.QueryAtype.Paging()` |
| 业务类型 | `c.BodyAtype.ImagePaths(...)`、`c.QueryAtype.Datetime(...)` |

## 响应速查

| 需求 | API |
|------|-----|
| 成功 DTO | `WriteData(v)` / `WriteOK()` |
| 错误 DTO | `WriteE(e)` / `WriteErr(err)` / `WriteMsg(code, msg)` |
| 有错写错、无错写数据 | `TryWrite(v, e)` / `TryWriteId(...)` |
| 纯文本 / 原始 JSON | `WriteString(s)` / `WriteRawJSON(b)` |
| 任意值 JSON | `JSON(v)` / `JSONP(v)` |
| 模板 | `Router.SetViews(dir)` + `c.ViewData(k, v)` + `c.View(name)` |
| 自定义序列化 | `settings.SetSerialize(...)`、`c.WithSerialize(...)` |
| bigint 转字符串 | 请求头 `X-Stringify: true` 或 query `x-stringify=true` |
| 字段裁剪 | query `x-field=a,b`（数组用 `x-field=[a,b]`） |
| 错误即状态码 | 请求头 `X-Error-As-Status: true` / query `x-error-as-status=true` / `c.ErrorAsStatus = true` |

## 中间件模板

```go
func authMiddleware(c *httpsvr.Context) {
	token := c.HeaderFast("X-Token")
	if token == "" {
		c.StatusCode(ae.Unauthorized)
		_, _ = c.WriteMsg(ae.Unauthorized, "missing token")
		c.StopExecution()
		return
	}
	// c.StdContext() 携带 trace / client ip / remote user
	c.SetValue(acontext.CtxRemoteUser, "user-1")
	c.Next()
}
```

- `UseRouter`：匹配前执行（404/405 也执行），适合 Recover / RequestID / CORS / 访问日志。
- `Use` / `Party(prefix, mw...)`：匹配成功后执行。
- `Done`：链尾；前面的 handler 必须 `Next()` 才会到达。

## 常见陷阱

| 错误做法 | 后果 | 正确做法 |
|----------|------|----------|
| 忘记 `c.Next()` | 后续中间件/控制器不执行 | `defer c.Next()` |
| `go func(){ use c }()` | Context 已被复用，数据错乱 | 只传 `c.StdContext()` 与数据 |
| 运行期注册路由 | 数据竞争 | 启动前注册 |
| 成功路径创建 `ae.New...` | 采样调用栈，明显变慢 | 只在错误分支创建 |
| 用 `*ae.Error` 当 `error` | 不实现 error 接口 | 用 `ae.NewErr` 或 `.String()` |
| 业务错误直接写 HTTP 4xx | 网关/监控误判 | 默认 body code，必要时 `ErrorAsStatus` |
| 直接改 `Settings` 共享对象做单接口定制 | 影响同 App 其他请求 | 用 `c.WithSerialize` / `c.WithErrorHandler` |

## 验证命令

```bash
gofmt -l .
go build ./...
go test ./...
go vet ./...
```

修改 HTTP 热路径时：

```bash
go test ./aa/httpsvr -run '^$' -bench 'BenchmarkHTTPX|BenchmarkRouter' -benchmem -count=3
```

## 深入阅读

- 完整 HTTP API：[references/http-api.md](./references/http-api.md)
- 配置 / Consul / resolver：[references/config-services.md](./references/config-services.md)
- 类型 / 错误 / 日志：[references/types-errors.md](./references/types-errors.md)
- 仓库架构：[../../../doc/architecture.md](../../../doc/architecture.md)
- 性能与限制：[../../../doc/performance.md](../../../doc/performance.md)
