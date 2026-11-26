# InfluxDB Driver

Package: `github.com/aarioai/aago/driver` (file `influxdb.go`)

## Configuration

```yaml
influxdb:
  DEFAULT:
    url: http://127.0.0.1:8086
    auth: token-or-user:password
    org: my-org
    bucket: metrics
    log_level: error          # 0/error, 1/warn, 2/info, 3/debug
    batch_size: 5000
    flush_interval: 1s
    precision: 1ms            # time.Millisecond etc.
    gzip: true
    retry_interval: 5s
    max_retries: 3
    retry_buffer_limit: 20000
    max_retry_interval: 30s
    max_retry_time: 60s
    exponent_base: 2
httpc:
  influxdb_mydb:              # HTTP client section used by influxdb.mydb
    timeout: 30s
```

Standalone example: [influxdb_demo.yaml](../../../driver/influxdb_demo.yaml).

InfluxDB keys accept snake_case, camelCase, or PascalCase, and named sections inherit
missing values from `DEFAULT`.

The HTTP client section is `httpc_<section>`; `ParseHttpClientConfig` strips the `httpc_`
prefix, so both `httpc: {mydb: {...}}` and `httpc: {httpc_mydb: {...}}` work.

> 中文提示：InfluxDB 复用 `httpc` 的 HTTP 配置段，段名为 `httpc_<influxdb section>`。

## API

```go
data, e := driver.NewInfluxdbPool(app, "mydb", defaultTags) // shared
client, org, bucket, e := driver.NewInfluxdb(app, "mydb", defaultTags)
cfg, err := driver.ParseInfluxdbConfig(app, "mydb")

data.Client                // influxdb2.Client
data.DefaultQuery          // api.QueryAPI
data.DefaultWrite          // api.WriteAPI (async, batched)
data.DefaultWriteBlocking  // api.WriteAPIBlocking

driver.CloseInfluxdbPool()
```

## Tuning notes / 调优要点

- `precision` is a `time.Duration`; only `Nanosecond`, `Microsecond`, `Millisecond` and
  `Second` are meaningful for InfluxDB.
- `retry_buffer_limit` should be a multiple of `batch_size`.
- `max_retry_time` bounds the total retry duration and is now mapped correctly (earlier
  versions accidentally used `max_retry_interval`).

> 中文提示：`max_retry_time` 之前错误地使用了 `max_retry_interval` 的值，现已修正。

## Health check

```go
check := driver.CheckInfluxdbHealth("mydb") // client.Ping
```
