# aago driver Documentation

 It turns `aago` configuration into ready-to-use clients for MySQL, Redis, MongoDB, RabbitMQ,
InfluxDB and HTTP, and adds small query/O-R-M helpers on top of the official drivers.

`aago driver` 是 aago 体系的数据访问层：把 `aago` 配置转换成 MySQL、Redis、MongoDB、RabbitMQ、
InfluxDB、HTTP 客户端，并在官方驱动之上提供查询 / O-R-M 辅助工具。

## Architecture / 结构

```
aago/
├── driver/                 # connection pools + configuration parsing + error mapping
│   ├── base.go             # section lookup, timeout parsing, CloseAllPools
│   ├── mysql.go            # MySQL pool (database/sql)
│   ├── postgres.go         # PostgreSQL pool (pgxpool)
│   ├── redis.go            # Redis pool (go-redis v9)
│   ├── mongodb.go          # MongoDB pool (mongo-driver v2)
│   ├── rabbitmq.go         # RabbitMQ connection (go-rabbitmq)
│   ├── influxdb.go         # InfluxDB client (influxdb-client-go v2)
│   ├── httpclient.go       # HTTP client with dial/TLS/pool tuning
│   ├── selftest.go         # health checks for every driver
│   ├── index/              # index/entity metadata shared by SQL & Mongo
│   ├── mysqlx/             # MySQL 8.0+ helpers: DB, Tx, ORMS, Cond, unions
│   ├── postgresx/          # PostgreSQL helpers using native pgx/v5
│   ├── mongodb/            # Mongo O-R-M: ORMS chain, bson2, bson3
│   ├── redisx/             # Redis helpers: hash, TTL, keys, locks, counters
│   ├── rabbitmqx/          # consumer/publisher declarations
│   └── emqxi/              # EMQX REST client
└── doc/                    # this documentation
```

## Quick start / 快速开始

```go
c, err := aconfig.New("./config.yaml", nil)
if err != nil {
    panic(err)
}
ctx, cancel := acontext.WithCancel(acontext.Background())
app := aa.New(ctx, cancel, c)

// Pooled clients are cached per section and shared process-wide.
schema, db, e := driver.NewMysqlPool(app, "test")
if e != nil {
    panic(e)
}
defer driver.CloseAllPools(context.Background()) // usually in main()
```

```yaml
# config.yaml
mysql:
  DEFAULT:
    host: 127.0.0.1:3306
    user: root
    password: secret
    timeout: 5s,5s,5s
  test:
    schema: test
```

## Guides / 文档目录

| Document                                         | Description                                              | 说明                   |
|--------------------------------------------------|----------------------------------------------------------|------------------------|
| [configuration.md](./configuration.md)           | Section lookup rules, `DEFAULT` inheritance, legacy keys | 配置查找规则与兼容键   |
| [drivers/mysql.md](./drivers/mysql.md)           | MySQL pool, DSN building, error mapping                  | MySQL 连接池与错误映射 |
| [drivers/redis.md](./drivers/redis.md)           | Redis options mapping, TTL helpers                       | Redis 配置与辅助函数   |
| [drivers/mongodb.md](./drivers/mongodb.md)       | Mongo client options, credentials, TLS                   | MongoDB 连接与认证     |
| [drivers/rabbitmq.md](./drivers/rabbitmq.md)     | RabbitMQ connections, consumers/publishers               | RabbitMQ 连接与声明    |
| [drivers/influxdb.md](./drivers/influxdb.md)     | InfluxDB batching/retry tuning                           | InfluxDB 批量与重试    |
| [drivers/httpclient.md](./drivers/httpclient.md) | HTTP transport, dialer and TLS tuning                    | HTTP 客户端调优        |
| [mysqlx.md](./mysqlx.md)                         | `mysqlx.DB`, `Tx`, `ORMS`, `Cond`, unions                 | MySQL 8.0+ 查询构建    |
| [postgresx.md](./postgresx.md)                     | `postgresx.DB`, `Tx`, `ORMS`, `Cond`, unions               | PostgreSQL 查询构建     |
| [asql.md](./asql.md)                             | Shared `Dialect`, `Cond`, struct/insert/union builders    | mysqlx/postgresx 公共层  |
| [mongodb-orm.md](./mongodb-orm.md)               | Mongo `ORMS` chain, `bson2`/`bson3` builders             | Mongo 条件构建         |
| [redisx.md](./redisx.md)                         | Hashes, counters, locks, time-slot keys                  | Redis 常用封装         |
| [index-model.md](./index-model.md)               | `index.Entity`, primary/unique/text keys                 | 实体索引元数据         |
| [emqx.md](./emqx.md)                             | EMQX REST client                                         | EMQX REST 客户端       |
| [errors.md](./errors.md)                         | `*ae.Error` mapping rules                                | 错误映射规则           |

## Design notes / 设计要点

- **Pooling** — every driver keeps a `sync.Map` keyed by config section. `NewXPool` is
  idempotent and never returns a client that the caller may close; use `CloseXPool` or
  `CloseAllPools` at shutdown.
- **Errors** — every driver converts library errors into `*ae.Error` so HTTP layers can map
  them to status codes without string matching.
- **Configuration** — keys are read through a single helper, so YAML `DEFAULT` sections and
  legacy flat keys keep working.

## 中文提示

- 生产环境只需在 `main` 结束时调用一次 `driver.CloseAllPools(ctx)`。
- 所有 `NewXxxPool` 的返回值归连接池管理，**不能**调用 `Close()` 释放，否则其它持有者会 panic。
- 错误统一是 `*ae.Error`（不是 `error`），请使用 `Code` / `Msg` 字段或 `IsNotFound()` 判断。
