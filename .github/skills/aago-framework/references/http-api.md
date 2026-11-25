# HTTP API 参考（aa/httpsvr）

## Router

```go
rt := httpsvr.NewRouter()   // 或 httpsvr.New()

// 中间件
rt.UseRouter(h...)                  // 匹配前（404/405 也执行）
rt.Use(h...) / rt.UseOnce(h...)     // begin（匹配成功后）
rt.Done(h...)                       // 链尾
rt.Reset()

// 路由
rt.Get/Post/Put/Patch/Head/Delete/Options/Connect/Trace(path, h...)
rt.Any(path, h...)
rt.Handle(method, path, h...)

// 分组
p := rt.Party("/v1", mw...)
p.Party("/admin", mw2...)           // 继承父组 begin/done
rt.PartyFunc("/v1", func(p *httpsvr.Party){ ... })

// 其它
rt.UseHooks(v any)                  // 每请求 ctx.Hooks()
rt.OnErrorCode(status, h...)
rt.OnAnyErrorCode(h...)
rt.SetViews(dir string) error
rt.Listen(addr string) error
rt.Run(addr string) error
rt.Shutdown(ctx context.Context) error
```

路径模式：`/a/{name}`、`/a/{id:uint64}`；支持类型：
`string bool int int8 int16 int32 int64 uint uint8 uint16 uint32 uint64 alphabetical uuid email mail weekday`。

匹配顺序：静态索引 → 动态路由（注册顺序）→ 405（带 `Allow`）→ 404。

## Context：请求信息

```
Method() string
Path() string
Host() string
FullRequestURI() string
Origin() string
RemoteAddr() string
UserAgent() string
ReqContentType() string
Request() *http.Request
ResetRequest(*http.Request)
ResponseWriter() http.ResponseWriter
StdContext() context.Context
```

## Context：参数

```
Params() Params
QueryBool/QueryBytes/QueryString/QueryStrings/QueryFast
QueryInt/QueryInt8/QueryInt16/QueryInt32/QueryInt64/QueryInts/QueryInt8s/...
QueryUint/QueryUint8/QueryUint16/QueryUint32/QueryUint64/QueryUints/QueryUint8s/...
QueryEnum/QueryEnum8/QueryEnum8i/QueryValid/QueryValid8/QueryId
Query(key, patterns...) (*RawValue, *ae.Error)
Queries() map[string]any
URLParam(key) string
BodyBool/BodyBytes/BodyString/BodyStringFast/BodyStrings/BodyInt.../BodyUint.../BodyEnum...
BodyAnyMap/BodyAnySlice/BodyStringMap/BodyStringsMap/BodyFloat64Map/BodyComplexMaps/...
Body(key, patterns...) (*RawValue, *ae.Error)
Files(path, required...) ([]*multipart.FileHeader, *ae.Error)
File(path, required...) (*multipart.FileHeader, *ae.Error)
HeaderParam/HeaderString/HeaderFast/HeaderValue/HeaderWild
HeaderWideString/HeaderWildFast
QueryWild/QueryWildString/QueryWildFast/QueryWildValue
Cookie/GetCookie/SetCookie/Cookies
FormValue
Values() *Values
ValueString(key) / SetValue(key, any)
Params().Get/GetString/GetInt/GetInt64/GetUint64/GetBool/GetEntry/Visit/Len
```

`patterns` 规则：`bool`=是否必填（默认 true）、`string`=正则、`*regexp.Regexp`=正则、`*atype.Atype`/`atype.Atype`=默认值。

路径参数匹配后会注入 query；优先级：`InjectQuery` > 路径参数 > URL query。

## Context：响应

```
WriteData(any) / WriteOK() / WriteCode(int) / WriteMsg(int, string) / WriteMsgf
WriteE(*ae.Error) / WriteErr(error)
TryWrite(any, *ae.Error) / TryWriteId/TryWriteUint64Id/TryWriteUintId/TryWriteAliasId...
WriteCreated([]byte) / WriteAccepted()
WriteRaw(contentType string, []byte) / WriteRawJSON/XML/HTML/OctetStream
WriteString(string) / Write([]byte)
JSON(any) / JSONP(any)
View(name string) / ViewData(name string, v any) / QueryToViewData(names...)
StatusCode(int) / Status() int / FlushStatus()
Header() http.Header / WithHeader(k, v) / WithHeaders(map) / OutHeader / DeleteOutHeader
WithContentType(ct) / SetContentType(ct) / ResolveContentType() string
WithServeContentTypes([]string)
WithErrorHandler(ErrorHandler) / WithSerialize(fn) / WithBeforeSerialize(fn) / WithBeforeFlush(fn)
ErrorAsStatus bool
FirstError(es ...*ae.Error) *ae.Error
IsOK(catchResponded bool, e *ae.Error) bool / OK(catchResponded bool, e *ae.Error)
```

## Settings

```go
type Settings struct {
	ServeContentTypes []string
	ErrorHandler      ErrorHandler
	BeforeSerialize   []BeforeSerialize
	Serialize         func(contentType string, d Body) ([]byte, string, error)
}

func NewSettings() *Settings
func SettingsFrom(ctx *Context) *Settings

// ErrorHandler: (ctx, contentType, dto) -> (n, err, next)
//   next=false 表示已自行写出响应
// BeforeSerialize: (ctx, contentType, dto) -> dto
```

Content-Type 优先级：`WithContentType` > 值袋 `aa.content.type` > `Accept` 协商 > 请求 `Content-Type` > 首个可服务类型（默认 `application/json`）。

## 内置中间件

```go
httpsvr.Recover      // panic → 500
httpsvr.RequestID    // X-Request-Id（缺失则生成），写入 ctx.Values 与响应头
httpsvr.AccessLog    // method path status duration
httpsvr.CORS(httpsvr.CORSOptions{
	AllowedOrigins:   []string{"*"},
	AllowedMethods:   []string{"GET", "POST"},
	AllowedHeaders:   []string{"*"},
	ExposedHeaders:   []string{"X-Request-Id"},
	MaxAge:           600,
	AllowCredentials: true,
})
```

## 中间件顺序

```
UseRouter → (匹配) → Use/Party begin → route handlers → Done
```

`StopExecution()` 之后 `Next()` 为空操作，后续（含 Done）全部跳过。

## 响应头 / 查询参数开关

| 名称 | 值 | 作用 |
|------|----|------|
| `X-Stringify` / `x-stringify` | `1`/`true` | int64/uint64 转字符串 |
| `X-Error-As-Status` / `x-error-as-status` | `1`/`true` | 错误码映射为 HTTP 状态码 |
| `x-field` | `a,b` 或 `[a,b]` | 只返回指定结构体/map 字段 |

## 相关源码

- [router.go](../../../../aa/httpsvr/router.go)
- [context.go](../../../../aa/httpsvr/context.go)
- [request_*.go](../../../../aa/httpsvr)
- [response_*.go](../../../../aa/httpsvr)
