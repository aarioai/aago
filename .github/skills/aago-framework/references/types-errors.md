# 类型 / 错误 / 日志参考

## 1. 错误：aa/ae

```go
type Error struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	Caller    string `json:"caller"`
	Detail    string `json:"details"`
	TraceInfo string `json:"trace_info"`
}
```

常用构造：

```go
ae.New(code, msg...)
ae.NewF(code, format, args...)
ae.NewError(msg) / ae.NewErrorf(format, args...)
ae.NewErr(err, detail...)
ae.NewBadParam(param, tips...)
ae.NewUnsupportedMedia(wants...)
ae.NewPreconditionFailed(msg...)
ae.NewConflict(name) / ae.NewConflictWith(msg)
ae.NewParamTooShort/NewParamTooLong(param, tips...)
ae.NewNotAcceptable/NewLoopDetected/NewVariantAlsoNegotiates/NewProxyAuthRequired
```

链式：

```go
e.WithMsg("...") / WithMsgf(...) / AppendMsg(...)
e.WithDetail("...") / WithCaller(skip) / WithTraceInfo(ctx)
e.Lock() / Unlock() / Clone()
e.String() / Text() / Trace(ctx)
e.IsNotFound() / IsServerError() / IsFailedAndSeeOther() / ExceptNotFound()
```

包级：

```go
ae.First(es...)
ae.FirstError(errs...)
ae.IsNotFound(err error)
ae.Panic / PanicF / PanicOn / PanicOnErrs / PanicWithCaller
ae.StopOnFirstError(func() error...) / StopOnFirstE(func() *ae.Error...)
ae.Wrap(head string, err error, tails ...string)
ae.Text(code int) string
```

约束：

- `*ae.Error` 不实现 `error` 接口。
- `New` 会采样调用栈，**成功路径禁止创建错误**。
- 常数错误 `Lock()` 后可安全共享。
- 错误码：标准 1xx–5xx + `FailedAndSeeOther=391`、`PageExpired=419`、`NoRowsAvailable=494`、`ConflictWith=499`、`Exception=590`。

## 2. 日志：aa/alog

```go
log := alog.NewDefaultLog(alog.LevelInfo)

log.Debugf(ctx, "cache hit %s", key)
log.Infof(ctx, "started on %s", addr)
log.E(ctx, e)                 // 使用 e.Caller，不重复采样
log.Errorf(ctx, "db: %v", err)

// 阶段日志
log2 := log.New("service: ", func(ctx context.Context, msg string, args ...any) {
	log.Infof(ctx, msg, args...)
}, " done")
```

级别：`LevelErrAll < Debug < Info < Notice < Warn < Error < Fatal < Alert < Emerg`。
禁用级别不会采样调用栈，生产建议 `LevelInfo`。

包级辅助：`Console`、`Print/Printf`、`Warn/Warnf`、`Error/Errorf`、`OnError`、`LogOnE`、`Start/Stop`。

## 3. 值包装：atype.Atype

```go
v := atype.New(raw)
defer v.Close()

v.String() / v.ReleaseString()
v.Bytes() / v.ReleaseBytes()
v.Bool() / v.DefaultBool(false) / v.ReleaseDefaultBool(false)
v.Int() / v.Int8() / v.Int16() / v.Int32() / v.Int64()
v.Uint() / v.Uint8() / v.Uint16() / v.Uint32() / v.Uint64()
v.Float32() / v.Float64()
v.NullString() / v.SqlFloat64() / ...
v.Raw() / v.Reload(x) / v.Get(keys...)
```

## 4. 业务类型速查

| 需求 | 类型 / 函数 |
|------|-------------|
| 金额 | `atype.Money`、`atype.Decimal`、`atype.VMoney`（`MoneyScale=4`） |
| 日期时间 | `atype.Date`、`Datetime`、`Timestamp`、`Weekday`、`Year`、`YearMonth` |
| 分页 | `atype.Paging`、`NewPaging(pageStart, pageEnd, pageSize...)` |
| 路径 | `atype.Path`、`Filename`、`Ext`、`PathPattern`、`AudioPath`、`VideoPath`、`ImagePath` |
| 校验字符串 | `UUID`、`Email`、`Word`、`Alphabetical`、`AlphaDigits`、`Uppers` |
| 可空列 | `NullString`、`NullJson`、`NullUint8s`、`NullInt64s` 等 |
| 区划 | `Distri`、`Province`、`Dist` |
| 版本 | `Version`、`VersionStruct`、`VersionTag`、`ToVersion`、`PackVersion` |
| IP / 坐标 | `IP`、`Coordinate`、`Location`、`Point`、`Position` |

枚举（`aa/atype/aenum`）：`Sex`、`Status`、`Country`、`Continent`。

校验函数：`IsEmail`、`IsUUID`、`IsPath`、`IsUnicodePath`、`IsFilename`、`IsWord`、`IsAlphabetical`、`IsNumberString`、`IsBin`、`IsLower(s)`、`IsUppers(s)`。

## 5. HTTP 类型命名空间

```go
page := c.QueryAtype.Paging()
page = c.QueryAtype.PagingWithSize(50)
page = c.QueryAtype.PagingWithFirstPageEnd(1000)

sex, e := c.QueryAtype.Sex("sex")
from, e := c.QueryAtype.Datetime("from", cfg.TimeLocation)
money, e := c.QueryAtype.Money("price", atype.Money(1), atype.Money(100000))

images, e := c.BodyAtype.ImagePaths("images", true)
```

`httpsvr/atype.Query{Src QuerySource}` / `Body{Src BodySource}` 只是适配层，可在测试里传入自定义 Source。

## 6. 请求上下文：aa/acontext

```go
stdCtx := c.StdContext()          // Bind：trace id + client ip + remote user
id := acontext.TraceID(stdCtx)
ip := acontext.RemoteAddrValue(stdCtx)
user := acontext.RemoteUserValue(stdCtx)

acontext.WithRemoteUser(c, "u1")   // 写入值袋
acontext.WithTrace(parent)         // 非 HTTP 场景生成 trace id

acontext.SetServStarting/Ready/Stopping/Unhealthy()
acontext.ServFallbackReady()
acontext.ServHealth().IsReady()
```

键：`CtxRequestID`、`CtxTraceId`、`CtxClientIP`、`CtxRemoteAddr`、`CtxRemoteUser`、`CtxContentType`。

## 7. pkg 工具

| 包 | 用途 |
|----|------|
| `types` | 整数格式化/解析（小整数查表，校验 base/bitSize）、`IsEmpty`、`Ref/RefNz/Deref`、64 进制、`ParseDuration` |
| `afmt` | 字符串补齐/裁剪、控制台颜色、时间差、路径/URL、`First`/`FirstNotEmpty` |
| `arrmap` | `Keys`/`SortedKeys`/`Merge`/`Compact`/`JoinKeys` |
| `httpc` | HTTP 客户端（共享 Transport、context、`*Json` 变体） |
| `ios` | `MkdirAll`、`PrepareFile` |
| `nets` | `LanIP`、`LocalPublicIP` |
| `utils` | `Caller`、`Close`、`GitVersion` |
| `basic` | `Ter(cond, a, b)` |

## 8. 相关源码

- [ae](../../../../aa/ae)
- [alog](../../../../aa/alog)
- [atype](../../../../aa/atype)
- [acontext](../../../../aa/acontext)
- [pkg](../../../../pkg)
