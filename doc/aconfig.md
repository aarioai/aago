# 配置与服务发现（aa/aconfig）

`aconfig` 负责：YAML 配置展开、环境识别、文本密钥加载、Consul 注册/发现，以及 gRPC resolver。

## 1. 创建与加载

```go
cfg, err := aconfig.New("config/app.yaml", nil)
```

- `path`：YAML 主配置文件路径。
- `valueProcessor`：`func(key, value string) (string, error)`，对**每个**配置项生效（可做解密、占位符替换）。

加载步骤：

1. 读取 YAML → 递归展开成 **点分键**（`app.http.port`）。
2. 若有 `text_config_dirs`，加载目录中的文本文件（密钥、证书等）。
3. 应用 `valueProcessor`。
4. `initializeConfig()`：解析 `env`、`mock`、`time_format`、`timezone_id`；设置 `time.Local`。

## 2. YAML 结构与 DEFAULT 继承

```yaml
env: production
mock: false
time_format: "2006-01-02 15:04:05"
timezone_id: Asia/Shanghai

app:
  DEFAULT:
    http:
      port: 8080
      read_timeout: 10s
  api:
    http:
      port: 9000
  admin:
    http:
      read_timeout: 30s
```

规则（见 [`flattenYAML`](../aa/aconfig/parse_yaml.go)）：

- 同级 `DEFAULT` 会被**深度合并**到其他兄弟节点：
  - 标量为空时取 DEFAULT；
  - map 先克隆 DEFAULT，再用当前值覆盖；
  - 列表/数组直接替换；
  - 字符串 `__DELETED__` 表示删除该 DEFAULT 子键。
- 结构不一致（map ↔ 非 map）时不合并。
- 最终键形如 `app.api.http.port`、`app.admin.http.read_timeout`。

保留键：

| 键 | 说明 |
|----|------|
| `env` | 环境名，见下节 |
| `mock` | 是否 mock 模式 |
| `text_config_dirs` | 空格分隔的文本配置目录 |
| `time_format` | 默认 `2006-02-01 15:04:05` |
| `timezone_id` | IANA 时区，如 `Asia/Shanghai` |

## 3. 读取 API

| 方法 | 行为 |
|------|------|
| `MustGetString(key)` | 按优先级取值，找不到返回 error |
| `GetString(key, default...)` | 取不到或为空时返回默认值 |
| `MustGet(key)` / `Get(key, default...)` | 返回 `*atype.Atype`（记得 `Close()`，或用 `Release*`） |
| `MustGetDuration(key)` / `GetDuration(key, def)` | `time.Duration` |
| `Extend(maps...)` | 合并"其他配置"（如数据库下发），覆盖已有键 |
| `Reload()` | 重新加载 YAML 与文本配置 |
| `All()` / `Dump()` / `Log()` | 查看/打印配置（注意脱敏） |
| `PanicIfNotEqual(key, want)` | 启动期校验 |

**取值优先级**：`otherConfig`（`Extend` 写入） > `baseConfig`（YAML） > `textConfig`（文本文件）。

文本配置的键规则：`目录名 + "." + 文件名`，例如 `secret/db.pem` → `secret.db.pem`。

## 4. 环境（Env）

```go
type Env string

const (
	EnvLocal       Env = "local"
	EnvDevelopment Env = "development"
	EnvTest        Env = "test"
	EnvStaging     Env = "staging"
	EnvProduction  Env = "production"
)
```

`NewEnv(s)` 支持别名与前缀/后缀：

| 输入 | 结果 |
|------|------|
| `testing` | `test` |
| `prod` / `pro` / `product` | `production` |
| `dev_user` | `development` |
| `project_testing` | `test` |

比较辅助：

```go
cfg.Env.IsProduction()
cfg.Env.BeforeStaging()   // local / development / test
cfg.Env.AfterDevelopment() // test / staging / production
```

> 推荐在 `boot` 阶段根据 `cfg.Env` 选择配置分组与开关，而不是在业务代码里读取环境字符串。

## 5. Consul 集成

```go
client, err := api.NewClient(&api.Config{Address: "127.0.0.1:8500"})
cfg.SetDefaultConsul(client)
```

| 方法 | 说明 |
|------|------|
| `SetConsul(name, client)` | 命名客户端 |
| `SetDefaultConsul(client)` | `DEFAULT` 客户端（注册/发现默认使用） |
| `Consul(name)` / `DefaultConsul()` | 取客户端（不存在返回 nil，调用方需判空） |
| `RegisterGRPCService(name, id, addr, checkAddr, port)` | 注册 gRPC 服务，健康检查间隔 15s / 超时 5s / 5m 后摘除 |
| `DeregisterGRPCService(id)` | 注销 |
| `DiscoverGRPCServices(name)` | 获取健康实例（无实例返回 error） |

地址归一化：空 / `localhost` / `0.0.0.0` / `::` 会替换为局域网 IP（失败则 `127.0.0.1`），方便容器内注册。

## 6. gRPC resolver

[`aa/aconfig/consul`](../aa/aconfig/consul) 实现了 `resolver.Builder`：

```go
import (
	cr "github.com/aarioai/aago/aa/aconfig/consul"
	"google.golang.org/grpc"
)

conn, err := grpc.NewClient(
	"consul:///user-service",
	grpc.WithResolvers(cr.NewBuilder(cfg.DefaultConsul())),
)
```

- Scheme：`consul`。
- 目标 endpoint（或 URL host）即服务名。
- 首次 `ResolveNow` + 后台 `Watch`（默认 15s）。
- 负载均衡：`round_robin`。
- 地址属性包含 `consul.service.id`、`consul.service.meta`、`consul.node.id`、`consul.node.name`。

## 7. 只读配置

```go
ro := aconfig.NewReadonly()
ro.Set("token", "abc")   // 已存在时不覆盖，返回 false
ro.Get("token")
ro.GetBytes("token")
```

内部使用 `sync.Map`，适合初始化后只读的共享数据。

## 8. 并发语义

- `Reload` / `Extend` 通过 `onWrite`（`atomic.Bool` + CAS）串行化。
- 读操作在写入期间使用 `cfgMtx.RLock()`；写入完成后直接读 map。
- **不要在配置写入的同时拷贝整个 map**；需要快照请使用 `All()`。
