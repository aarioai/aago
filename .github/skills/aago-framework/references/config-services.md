# 配置与服务发现参考（aa/aconfig）

## 创建

```go
cfg, err := aconfig.New("config/app.yaml", nil)
// 带值处理器（解密、占位符替换）
cfg, err := aconfig.New(path, func(key, value string) (string, error) {
	return decryptIfNeeded(key, value)
})
```

## 保留键

| 键 | 说明 |
|----|------|
| `env` | 环境，见 `Env` |
| `mock` | mock 模式 |
| `text_config_dirs` | 空格分隔的文本配置目录 |
| `time_format` | 默认 `2006-02-01 15:04:05` |
| `timezone` | IANA 时区，设置后同时修改 `time.Local` |

## YAML 展开规则

- 嵌套 map 展开为点分键：`app.api.http.port`。
- 同级 `DEFAULT` 深度合并到兄弟节点：
  - 标量为空 → 取 DEFAULT；
  - map → 克隆 DEFAULT 后覆盖；
  - 数组 → 直接替换；
  - `__DELETED__` → 删除该 DEFAULT 子键。

```yaml
app:
  DEFAULT:
    http: { port: 8080, read_timeout: 10s }
  api:
    http: { port: 9000 }
```

得到 `app.api.http.port=9000`、`app.api.http.read_timeout=10s`。

## 读取

```go
s, err := cfg.MustGetString("app.api.http.port")
s = cfg.GetString("app.api.http.port", "8080")

v, err := cfg.MustGet("app.api.http.port") // *atype.Atype，需 Close 或用 Release*
n, err := v.ReleaseInt64()

d, err := cfg.MustGetDuration("app.api.http.read_timeout")
d = cfg.GetDuration("app.api.http.read_timeout", 10*time.Second)

cfg.Extend(map[string]string{"app.api.http.port": "9001"}) // 覆盖
cfg.Reload()
```

优先级：`Extend` 写入的 `otherConfig` > YAML `baseConfig` > 文本配置 `textConfig`。

文本配置键：`目录名 + "." + 文件名`（如 `secret/db.pem` → `secret.db.pem`）。

## 环境

```go
const (
	EnvLocal Env = "local"
	EnvDevelopment Env = "development"
	EnvTest Env = "test"
	EnvStaging Env = "staging"
	EnvProduction Env = "production"
)

cfg.Env.IsProduction()
cfg.Env.BeforeStaging()
cfg.Env.AfterDevelopment()
```

`NewEnv` 别名：`dev/develop`、`testing`、`stag`、`prod/product/pro`；支持 `xxx_testing`、`testing_xxx` 形式。

## Consul

```go
import "github.com/hashicorp/consul/api"

client, err := api.NewClient(&api.Config{Address: "127.0.0.1:8500"})
cfg.SetDefaultConsul(client)

err = cfg.RegisterGRPCService("user-service", "user-1", "0.0.0.0", "0.0.0.0", 9000)
err = cfg.DeregisterGRPCService("user-1")
services, meta, err := cfg.DiscoverGRPCServices("user-service")
```

- 地址为空/`localhost`/`0.0.0.0`/`::` 时替换为局域网 IP。
- 健康检查：`GRPC checkAddr:port`，间隔 15s，超时 5s，5m 后摘除。
- `Consul(name)` / `DefaultConsul()` 不存在时返回 nil，需判空。

## gRPC resolver

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

- scheme：`consul`；endpoint（或 URL host）为服务名。
- 首次立即解析，之后每 15s 监听；`round_robin` 负载均衡。
- 地址属性：`consul.service.id`、`consul.service.meta`、`consul.node.id`、`consul.node.name`。

## 只读配置

```go
ro := aconfig.NewReadonly()
ro.Set("token", "abc") // 已存在返回 false
s := ro.Get("token")
b := ro.GetBytes("token")
```

## 并发

- `Reload` / `Extend` 通过 `onWrite` CAS 互斥。
- 写入期间读操作用 `cfgMtx.RLock()`；写完成后直接读 map。
- 需要完整快照时用 `All()`，不要在写期间直接遍历内部 map。

## 相关源码

- [aconfig](../../../../aa/aconfig)
- [consul resolver](../../../../aa/aconfig/consul)
