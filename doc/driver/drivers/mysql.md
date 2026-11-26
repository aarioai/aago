# MySQL Driver

Package: `github.com/aarioai/aago/driver` (file `mysql.go`)

## Configuration

```yaml
mysql:
  DEFAULT:
    host: 127.0.0.1:3306
    user: root
    password: secret
    tls: false               # true | false | skip-verify | preferred | <name>
    timeout: 5s,5s,5s        # connect, read, write
    pool_max_idle_conns: 10
    pool_max_open_conns: 100
    pool_conn_max_life_time: 1h
    pool_conn_max_idle_time: 30m
  test:                      # section name; schema defaults to the section name
    schema: test
```

You can provide a driver DSN directly. When `url` is set, its connection values
take precedence over the other connection keys; the DSN is parsed into
`MysqlOptions`. Without `url`, the driver builds a DSN from the configured
connection keys.

```yaml
mysql:
  reporting:
    url: "reporter:secret@tcp(db.example:3306)/reports?parseTime=true&tls=preferred"
```

| Key                       | Type     | Default      | Notes                                 |
|---------------------------|----------|--------------|---------------------------------------|
| `url`                     | string   | –            | Optional Go MySQL driver DSN; overrides connection keys |
| `host` / `addr`           | string   | –            | `host:port`; required unless `url` is set |
| `schema`                  | string   | section name | database name                         |
| `user` / `password`       | string   | –            | required unless `url` is set          |
| `tls`                     | string   | `""`         | forwarded to `mysql.Config.TLSConfig` |
| `timeout`                 | string   | –            | `conn,read,write` durations           |
| `pool_max_idle_conns`     | int      | `0`          | `sql.DB.SetMaxIdleConns`              |
| `pool_max_open_conns`     | int      | `0`          | `sql.DB.SetMaxOpenConns`              |
| `pool_conn_max_life_time` | duration | `0`          | `SetConnMaxLifetime`                  |
| `pool_conn_max_idle_time` | duration | `0`          | `SetConnMaxIdleTime`                  |

## API

```go
// pooled and shared per section (recommended)
schema, db, e := driver.NewMysqlPool(app, "test")

// unpooled, caller owns the *sql.DB
schema, db, e := driver.NewMysql(app, "test")
defer db.Close()

// DSN only
dsn, opts, e := driver.NewMysqlDSN(app, "test")

// parsed options only (no connection)
opts, err := driver.ParseMysqlConfig(app, "test")

driver.CloseMysqlPool()   // shutdown only
```

The DSN is built with `mysql.Config.FormatDSN()`, so user names, passwords and database names
containing special characters are escaped correctly and `tls`/timeouts are preserved with full
precision.

> 中文提示：DSN 由官方 `mysql.Config` 生成，密码含特殊字符也不会出错；`tls` 现在会真正生效。

## Error mapping

```go
schema, db, e := driver.NewMysqlPool(app, "test")
if e != nil && e.IsNotFound() { /* … */ }
```

`NewMysqlError` classifies driver errors:

| Condition | Result |
| --- | --- |
| `sql.ErrNoRows` | `ae.ErrorNotFound` |
| duplicate key (`Duplicate entry '…' for key '…'`) | `ae.NewConflict("sql key")` |
| `driver.ErrBadConn` / `ErrSkip` / `ErrRemoveArgument` / `sql.ErrConnDone` / `sql.ErrTxDone` | `ae.NewError` with a prefixed message |
| anything else | `ae.NewError(caller + "sql error: " + msg)` |

Every error accepts `WithDetail(...)` so the failing query and arguments can be attached.

> 中文提示：唯一键冲突直接映射为 409 类冲突错误，无需在业务层判断错误字符串。

## Health check

```go
check := driver.CheckMySQLHealth("test")
```

`CheckMySQLHealth` returns `func(app *aa.App, errChan chan<- error)`; it opens a dedicated
connection (not the pool), runs `Ping`, closes it and reports the result on `errChan`.
