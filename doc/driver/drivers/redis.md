# Redis Driver

Package: `github.com/aarioai/aago/driver` (file `redis.go`)

## Configuration

```yaml
redis:
  DEFAULT:
    addr: 127.0.0.1:6379
    password: secret
    timeout: 5s,3s,3s          # dial, read, write
    pool_size: 64
    min_idle_conns: 8
    pool_timeout: 4s
  cache:
    db: 1
    max_retries: 3
```

Standalone example: [redis_demo.yaml](../../../driver/redis_demo.yaml).

Redis keys accept snake_case, camelCase, or PascalCase, and named sections inherit
missing values from `DEFAULT`.

Mapped fields (all optional except `addr`):

| Key                                                                 | Maps to                                      |
|---------------------------------------------------------------------|----------------------------------------------|
| `addr`, `network`, `client_name`                                    | `redis.Options`                              |
| `protocol`, `username`, `password`, `db`                            | `redis.Options`                              |
| `max_retries`, `min_retry_backoff`, `max_retry_backoff`             | retry policy                                 |
| `timeout`                                                           | `DialTimeout`, `ReadTimeout`, `WriteTimeout` |
| `context_timeout_enabled`, `pool_fifo`, `pool_size`, `pool_timeout` | pool behaviour                               |
| `min_idle_conns`, `max_idle_conns`, `max_active_conns`              | pool sizing                                  |
| `conn_max_idle_time`, `conn_max_lifetime`                           | connection lifetime                          |
| `disable_identity`, `identity_suffix`, `unstable_resp3`             | client identity / RESP3                      |

## API

```go
client, e := driver.NewRedisPool(app, "cache") // shared, do not Close
client, e := driver.NewRedis(app, "cache")     // unpooled, caller closes
opts, err := driver.ParseRedisConfig(app, "cache")

driver.CloseRedisPool()
```

> 中文提示：go-redis 自带线程安全连接池；`NewRedisPool` 只是保证同一 section 全局复用同一个客户端。

## Error mapping

| Condition     | Result                                                  |
|---------------|---------------------------------------------------------|
| `redis.Nil`   | `ae.New(ae.NotFound, "redis key not found")`            |
| anything else | `ae.New(ae.InternalServerError, caller+" redis: "+msg)` |

## Helpers (package `../../../aago/driver/redisx`)

See [redisx.md](../redisx.md) for hashes, counters, locks and time-slot keys.

```go
err := redisx.HSet(ctx, client, time.Hour, "k", map[string]any{"a": 1})
n, e := redisx.HIncr(ctx, client, time.Hour, "k", "a")
vals, ok, err := redisx.TryHMGetString(ctx, client, "k", "a", "b")
```

## Health check

```go
check := driver.CheckRedisHealth("cache") // func(app *aa.App, errChan chan<- error)
```
