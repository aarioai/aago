---
name: aago-driver-config
description: >-
  Configure and wire aago driver connections (MySQL, Redis, MongoDB, RabbitMQ, InfluxDB, HTTP)
  from an aago App: choose config section names, rely on DEFAULT inheritance, use pooled
  clients, add health checks, and shut the pools down correctly. Use when adding a new driver
  section, debugging configuration lookups, or wiring clients into a service.
---

# Working with aago driver configuration

## When to use this skill

- A configuration value is not picked up (`host`, `timeout`, `db`, …) or a section resolves to the
  wrong values.
- Wiring clients into `main`, adding health checks, or fixing shutdown order.

## Golden rules

1. **Never call `Close` on a pooled client.** `driver.NewXxxPool` returns a process-wide shared
   client keyed by config section; closing it breaks other users. Only `driver.CloseXxxPool()` /
   `driver.CloseAllPools(ctx)` at shutdown may close them.
2. **Pooled first.** Use `NewXxxPool` in services; use the non-pooled `NewXxx` only for one-shot
   tools, health checks or tests, and close that client yourself.
3. **One section per logical target.** `ParseXxxConfig(app, section)` resolves
   `<base>.<section>.<key>` → `<base>.DEFAULT.<key>` → legacy `section.key` / `<base>_<section>.key`
   / `<base>.key`. A section named `""`, `"DEFAULT"` or equal to the base name resolves to
   `DEFAULT`; a `<base>_` prefix is stripped (`mysql_test` → `test`).
4. **Timeouts are `conn,read,write`.** Any element may be omitted (`5s`, `,5s,5s`); invalid values
   keep the default instead of failing.
5. **Secrets stay in config.** Never hardcode credentials in code, URIs or tests (MongoDB
   credentials are applied via `SetAuth`, not embedded in the URI).

## Typical wiring

```go
func main() {
    c, err := aconfig.New("./config.yaml", nil)
    if err != nil {
        panic(err)
    }
    ctx, cancel := acontext.WithCancel(acontext.Background())
    app := aa.New(ctx, cancel, c)
    defer driver.CloseAllPools(context.Background())

    schema, db, e := driver.NewMysqlPool(app, "test")
    if e != nil {
        panic(e)
    }
    _ = mysqlx.NewDriver(schema, db, nil)
}
```

```yaml
mysql:
  DEFAULT:
    host: 127.0.0.1:3306
    user: root
    password: secret
    timeout: 5s,5s,5s
  test:
    schema: test
```

## Bases and required keys

| Driver | Base | Required keys |
| --- | --- | --- |
| MySQL | `mysql` | `host`, `user`, `password` (`schema` defaults to the section) |
| PostgreSQL | `postgres` | `host`, `user` (`database` defaults to the section; password may be empty) |
| Redis | `redis` | `addr` |
| MongoDB | `mongodb` | `hosts` |
| RabbitMQ | `rabbitmq` | `host`, `user`, `password` |
| InfluxDB | `influxdb` | `url`, `auth`, `org`, `bucket` |
| HTTP | `httpc` | – (defaults applied) |

InfluxDB reads its HTTP client from the `httpc` section named `httpc_<section>`.

## Health checks

Each driver exposes `CheckXxxHealth(section)` returning `func(app *aa.App, errChan chan<- error)`:

```go
checks := []func(*aa.App, chan<- error){
    driver.CheckMySQLHealth("test"),
    driver.CheckPostgreHealth("analytics"),
    driver.CheckRedisHealth("cache"),
    driver.CheckMongodbHealth("stats"),
    driver.CheckRabbitmqHealth("mq", tlsConfig, nil, nil),
    driver.CheckInfluxdbHealth("metrics"),
}
```