# Configuration / 配置

All drivers read their settings from an `*aa.App` (`app.Config`) using one shared lookup
helper, so the same rules apply to MySQL, PostgreSQL, Redis, MongoDB, RabbitMQ, InfluxDB and HTTP.

所有驱动都通过同一个查找函数读取 `*aa.App` 配置，因此规则完全一致。

## Section layout / 段落结构

The preferred YAML layout groups a `DEFAULT` section (shared values) and one section per
logical connection:

```yaml
mysql:
  DEFAULT:
    host: 127.0.0.1:3306
    user: root
    password: secret
    timeout: 5s,5s,5s
  test:              # inherits host/user/password/timeout from DEFAULT
    schema: test
  hello:
    schema: helloworld
```

PostgreSQL pools can be configured from either a connection URL or explicit fields:

```yaml
postgres:
  DEFAULT:
    host: 127.0.0.1
    port: 5432
    database: app
    user: app
    password: secret
    sslmode: verify-full
    timezone: Asia/Shanghai
    pool_max_conns: 30
    pool_min_conns: 2
    pool_max_conn_lifetime: 1h
    pool_max_conn_idle_time: 30m
  reporting:
    url: postgres://reporter:secret@db.example/app?sslmode=verify-full
```

PostgreSQL uses the `postgres` configuration base, supports the same `DEFAULT` inheritance
and section aliases, and accepts `pool_min_idle_conns`, `pool_max_conn_lifetime_jitter`, and
`pool_health_check_period` in addition to the settings shown above. The password stays in config;
the driver does not log the connection URL.

A section may use either `addr` (mysql-style `host:port`, or a unix socket directory/file with
`net: unix`) or the legacy `host`/`port` keys; `host` may include `:port`. A bare host without a
port falls back to `port`, then to `5432`, and a missing address defaults to `127.0.0.1:5432`.
`sslmode` (alias `tls`) defaults to `prefer`, and `timezone` sets the session parameter when
present. When `dsn` or its alias `url` is set, the remaining discrete keys (except pool tuning)
are ignored.

```go
driver.ParseMysqlConfig(app, "test")
driver.ParseMysqlConfig(app, "mysql_test") // same as "test"
driver.ParseMysqlConfig(app, "mysql")      // same as DEFAULT
driver.ParseMysqlConfig(app, "DEFAULT")    // DEFAULT
```

## Lookup order / 查找顺序

For `base = "mysql"`, `section = "test"`, `key = "host"`:

1. `mysql.test.host`
2. `mysql.DEFAULT.host`
3. Legacy flat keys: `test.host`, `mysql_test.host`, `mysql.host`

Normalisation rules / 归一化规则:

| Input `section` | Effective section | Note |
| --- | --- | --- |
| `""`, `"DEFAULT"`, `"mysql"` | `DEFAULT` | uses `mysql.DEFAULT.<key>` then `mysql.<key>` |
| `"mysql_test"` | `test` | `<base>_` prefix is stripped |
| `"test"` | `test` | section used as-is |

If no key is found and a default value was passed to the lookup, the default is returned;
otherwise the last lookup error is returned. Drivers treat optional keys (TLS, pool tuning,
retry settings) as optional and only require the connection-critical ones.

> 中文提示：`DEFAULT` 段用于共享公共配置，子段只需写差异项；旧的扁平写法（`test.host`）仍然兼容。

## Timeouts / 超时

`timeout` is a comma separated `conn,read,write` list. Every element is optional and accepts
any `time.ParseDuration` string:

```yaml
timeout: 10s,5s,5s     # connect, read, write
timeout: 3s            # connect only, read/write stay at their defaults
timeout: ,5s,5s        # read and write only
```

```go
conn, read, write := driver.ParseTimeouts("10s,1000ms")
```

Only positive, parseable values override the defaults.

> 中文提示：空项与非法值会被忽略，从而保留默认值，因此可以安全地只配置部分超时。

## Lists / 列表

Some options accept comma separated lists, e.g. `tls_cipher_suites`, `tls_next_protos`,
`tls_curve_preferences`. Spaces are removed before splitting.

TLS client certificates use `;` between pairs and `,` inside a pair:

```yaml
tls_certificate_file_pairs: /certs/a.crt,/certs/a.key;/certs/b.crt,/certs/b.key
```

A pair with an unexpected number of elements returns `driver.ErrInvalidHttpCertConfig`.

## Booleans and numbers / 布尔与数字

Values are converted with `types.ToBool`, `types.ToInt*`, `types.ParseDuration`; conversion
failures fall back to the zero value, so misspelled numbers do not panic at startup but should
be treated as configuration bugs.

## Lifecycle / 生命周期

```go
// main()
defer driver.CloseAllPools(context.Background())
```

`CloseAllPools` closes InfluxDB, MongoDB, MySQL, PostgreSQL, RabbitMQ and Redis pools concurrently.

> 中文提示：连接池按 `section` 缓存，同一个 section 多次调用 `NewXxxPool` 只会创建一次。
> 关闭连接只应在进程退出时统一进行。
