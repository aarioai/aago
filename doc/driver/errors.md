# Error handling / 错误处理

Every driver returns `*ae.Error` (from `github.com/aarioai/aago/aa/ae`) rather than `error`.
`*ae.Error` is a struct with `Code`, `Msg`, `Caller`, `Detail` and `TraceInfo`; it is **not** an
`error` implementation, so pass `err.Msg` when an `error` is required.

所有驱动返回 `*ae.Error`，它带有 `Code` / `Msg` / `Caller` / `Detail` / `TraceInfo` 字段；
它不是 `error` 接口，需要转换为 `error` 时使用 `err.Msg`。

## Constructors used by the drivers

| Helper                                         | Usage                                                |
|------------------------------------------------|------------------------------------------------------|
| `ae.NewError(msg)`                             | generic internal error                               |
| `ae.NewErrorf(format, args…)`                  | formatted internal error                             |
| `ae.NewErr(err, details...)`                   | wrap a library error                                 |
| `ae.NewConflict(name)`                         | conflict (duplicate key, lock already held)          |
| `ae.NewF(code, format, args…)`                 | explicit code, e.g. `newConfigError`                 |
| `ae.ErrorNotFound`                             | shared not-found value (locked, cannot take details) |
| `ae.ErrorNoRowsAvailable`                      | query returned no rows                               |
| `ae.ErrorEmptyInput` / `ae.ErrorInputTooShort` | invalid argument                                     |
| `ae.FirstError(errs...)`                       | first non-nil `error`                                |

## Mapping table

| Driver  | Library condition                                                                       | `*ae.Error`                                               |
|---------|-----------------------------------------------------------------------------------------|-----------------------------------------------------------|
| MySQL   | `sql.ErrNoRows`                                                                         | `ae.ErrorNotFound`                                        |
| MySQL   | duplicate key message                                                                   | `ae.NewConflict("sql key")`                               |
| MySQL   | `driver.ErrBadConn`, `ErrSkip`, `ErrRemoveArgument`, `sql.ErrConnDone`, `sql.ErrTxDone` | `ae.NewError(caller + prefix + msg)`                      |
| MySQL   | other                                                                                   | `ae.NewError(caller + "sql error: " + msg)`               |
| Redis   | `redis.Nil`                                                                             | `ae.New(ae.NotFound, "redis key not found")`              |
| Redis   | other                                                                                   | `ae.New(ae.InternalServerError, …)`                       |
| Redis   | `ApplyLock` not acquired                                                                | `ae.NewConflict(key)`                                     |
| MongoDB | `ErrNoDocuments`, `ErrNilDocument`, `ErrFileNotFound`                                   | `ae.ErrorNotFound`                                        |
| MongoDB | `ErrClientDisconnected`                                                                 | `ae.NewError(caller + "sql bad conn: " + msg)`            |
| MongoDB | other                                                                                   | `ae.NewErr(err)`                                          |
| Config  | missing/invalid section                                                                 | `newConfigError` → `ae.NewF(ae.VariantAlsoNegotiates, …)` |
| EMQX    | `{"code":…,"reason":…}`                                                                 | `ae.NewErrorf("code: %s, reason: %s", …)`                 |

## Checking errors

```go
_, db, e := driver.NewMysqlPool(app, "test")
switch {
case e == nil:
case e.IsNotFound():
    // HTTP 404
default:
    log.Println(e.Code, e.Msg, e.Detail)
}
```

`IsNotFound()` and the `Code` constants (`ae.NotFound`, `ae.Conflict`, `ae.InternalServerError`,
`ae.VariantAlsoNegotiates`, …) come from the `ae` package.

> 中文提示：判断是否“没查到”请用 `e.IsNotFound()`，不要比较字符串；
> `ae.ErrorNotFound` 是全局共享的锁值，不能 `WithDetail`。
