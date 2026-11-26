# RabbitMQ Driver

Package: `github.com/aarioai/aago/driver` (files `rabbitmq.go`, `rabbitmq_logger.go`)
and `github.com/aarioai/aago/driver/rabbitmqx`.

## Configuration

```yaml
rabbitmq:
  DEFAULT:
    host: 127.0.0.1:5672
    user: guest
    password: guest
    vhost: /
    channel_max: 2047        # 0 means 2^16-1
    frame_size: 0            # 0 means unlimited
    heartbeat: 10s           # < 1s uses the server value
    reconnect_interval: 5s
```

Standalone example: [rabbitmq_demo.yaml](../../../driver/rabbitmq_demo.yaml).

RabbitMQ keys accept snake_case, camelCase, or PascalCase, and named sections inherit
missing values from `DEFAULT`.

The URL is assembled with `net/url`, so user names, passwords and virtual hosts containing
reserved characters (`/`, `@`, `:`, `%`, …) are escaped correctly.

> 中文提示：URL 使用 `net/url` 拼接，密码或 vhost 含特殊字符时不再出错。

## API

```go
conn, e := driver.NewRabbitmqPool(app, "mq", tlsConfig, sasl, opts) // shared, do not Close
conn, e := driver.NewRabbitmq(app, "mq", tlsConfig, sasl, opts)     // caller closes
cfg, err := driver.ParseRabbitmqConfig(app, "mq", tlsConfig, sasl)

driver.CloseRabbitmqPool()
```

`NewRabbitmq` always installs a `rabbitmq.Logger` bridge to `app.Log`; additional
`rabbitmq.ConnectionOptions` are appended after the defaults.

## Declarations (`rabbitmqx`)

```go
type Declare struct {
    Queue      string
    Exchange   string
    RoutingKey string
}

consumer, err := rabbitmqx.NewConsumer(conn, declare)
consumer, err := rabbitmqx.DeclareConsumer(conn, declare) // declares the exchange
publisher, err := rabbitmqx.NewPublisher(conn, "events")
publisher, err := rabbitmqx.DeclarePublisher(conn, "events")
```

Both helpers append `ExchangeName` / `RoutingKey` / `ExchangeDeclare` options to whatever the
caller already passed.

> 中文提示：`DeclareXxx` 只是自动追加声明参数，其余行为与 `NewXxx` 完全一致。

## Health check

```go
check := driver.CheckRabbitmqHealth("mq", tlsConfig, sasl, nil)
```

It opens (or reuses) the pooled connection and creates an auto-delete consumer, which forces a
channel negotiation on the broker.
