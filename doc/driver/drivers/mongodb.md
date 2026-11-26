# MongoDB Driver

Package: `github.com/aarioai/aago/driver` (file `mongodb.go`) and
`github.com/aarioai/aago/driver/mongodb` (O-R-M helpers).

## Configuration

```yaml
mongodb:
  DEFAULT:
    protocol: mongodb          # default "mongodb"
    hosts: 127.0.0.1:27017,127.0.0.1:27018
    db: app
    username: root
    password: secret
    auth_source: admin
    auth_mechanism: SCRAM-SHA-256
  stats:
    connect_timeout: 10s
    server_selection_timeout: 30s
    timeout: 60s               # operation (timeoutMS) timeout
    max_pool_size: 100
    min_pool_size: 4
    replica_set: rs0
    tls: true
    tls_insecure_skip_verify: false
    write_concern: majority    # or a number such as "2"
    write_concern_journal: true
```

Standalone example: [mongodb_demo.yaml](../../../driver/mongodb_demo.yaml).

MongoDB keys accept snake_case, camelCase, or PascalCase, and named sections inherit
missing values from `DEFAULT`.

| Key                                                                                            | Struct field              | Notes                             |
|------------------------------------------------------------------------------------------------|---------------------------|-----------------------------------|
| `protocol`                                                                                     | `MongodbOptions.Protocol` | defaults to `mongodb`             |
| `hosts`                                                                                        | `MongodbOptions.Hosts`    | spaces are stripped               |
| `db`                                                                                           | `MongodbOptions.DB`       | database name                     |
| `username` / `password` / `auth_source` / `auth_mechanism`                                     | `MongodbCredential`       | credentials applied via `SetAuth` |
| `SERVICE_NAME`, `CANONICALIZE_HOST_NAME`, `SERVICE_REALM`, `SERVICE_HOST`, `AWS_SESSION_TOKEN` | `AuthMechanismProperties` | only read when a username is set  |
| `connect_timeout`                                                                              | `ConnectTimeout`          | dial timeout                      |
| `timeout`                                                                                      | `Timeout`                 | operation timeout (`SetTimeout`)  |
| `server_selection_timeout`                                                                     | `ServerSelectionTimeout`  | default 30s                       |
| `max_pool_size` / `min_pool_size` / `max_idle_time`                                            | pool tuning               |                                   |
| `replica_set`, `direct`, `heartbeat_frequency`                                                 | topology                  |                                   |
| `tls`, `tls_insecure_skip_verify`                                                              | `TLSConfig`               | enables TLS when `tls: true`      |
| `write_concern`, `write_concern_journal`                                                       | `WriteConcern`            | numeric strings become `w=2`      |
| `writer_concern`, `writer_concern_journal`                                                     | same fields               | legacy spelling, still accepted   |

### Security note / 安全说明

Credentials are **never embedded in the connection URI**. The URI is `mongodb://<hosts>` and
the credential is applied with `options.SetAuth`, so passwords do not leak through logs that
print the URI.

> 中文提示：连接 URI 中不再包含用户名与密码（历史版本曾硬编码了一个 root 账号），
> 密码通过 `SetAuth` 传入，避免被日志或错误信息泄露。

## API

```go
client, db, e := driver.NewMongodbPool(app, "stats") // shared, do not Disconnect
client, db, e := driver.NewMongodb(app, "stats")     // unpooled, caller Disconnect
opts, err := driver.ParseMongodbConfig(app, "stats")

driver.CloseMongodbPool(ctx)
```

## Error mapping

| Condition | Result |
| --- | --- |
| `mongo.ErrNoDocuments`, `ErrNilDocument`, `ErrFileNotFound` | `ae.ErrorNotFound` |
| `mongo.ErrClientDisconnected` | `ae.NewError(caller + "sql bad conn: " + msg)` |
| anything else | `ae.NewErr(err)` with details |

> 中文提示：查询不到文档统一返回 `ae.ErrorNotFound`，可直接用 `e.IsNotFound()` 判断。

## O-R-M helpers

See [mongodb-orm.md](../mongodb-orm.md).

## Health check

```go
check := driver.CheckMongodbHealth("stats") // Ping with a dedicated client
```
