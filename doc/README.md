# AaGo 文档

本目录描述 AaGo 框架的架构、各包职责、约定与性能特征。项目介绍见 [../README.md](../README.md)，本目录面向**使用框架开发业务**与**维护框架本身**的读者。

## 文档地图

| 文档 | 内容 | 主要读者 |
|------|------|----------|
| [architecture.md](./architecture.md) | 分层架构、包依赖、请求生命周期、启动流程、扩展点 | 全体 |
| [httpsvr.md](./httpsvr.md) | HTTP 框架：Router / Party / 中间件顺序 / Context / 参数解析 / DTO 响应 / Settings | 业务开发、框架开发 |
| [aconfig.md](./aconfig.md) | YAML 配置展开、DEFAULT 继承、环境（Env）、文本配置、Consul 注册发现、resolver | 业务开发、运维 |
| [atype.md](./atype.md) | `Atype` 与业务类型：金额、日期、版本、区划、分页、路径、枚举 | 业务开发 |
| [supporting-packages.md](./supporting-packages.md) | `ae` / `alog` / `acontext` / `helpers/debug` / `pkg/*` | 业务开发 |
| [conventions.md](./conventions.md) | 编码约定、错误处理、日志、测试与基准、提交前检查 | 贡献者 |
| [performance.md](./performance.md) | 热路径优化记录、基准数据、调优建议、已知限制 | 框架开发 |

## 5 分钟上手

```go
package main

import (
	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/aa/aconfig"
	"github.com/aarioai/aago/aa/acontext"
	"github.com/aarioai/aago/aa/httpsvr"
)

func main() {
	cfg, err := aconfig.New("config/app.yaml", nil)
	if err != nil {
		panic(err)
	}

	ctx, cancel := acontext.WithCancel(acontext.Background())
	defer cancel()
	app := aa.New(ctx, cancel, cfg)

	rt := httpsvr.NewRouter()
	rt.UseHooks(app.HTTP)                                       // 注入响应策略（Content-Type、错误处理、序列化）
	rt.UseRouter(httpsvr.Recover, httpsvr.RequestID, httpsvr.AccessLog)
	rt.Get("/v1/ping", func(c *httpsvr.Context) {
		_, _ = c.WriteData(map[string]string{"pong": "1"})
	})
	rt.Get("/v1/users/{uid:uint64}", getUser)

	if err := rt.Listen(":8080"); err != nil {
		app.Log.Error(ctx, err.Error())
	}
}

func getUser(c *httpsvr.Context) {
	uid, e := c.QueryUint64("uid")
	if e != nil {
		_, _ = c.WriteE(e)
		return
	}
	_, _ = c.WriteData(map[string]uint64{"uid": uid})
}
```

约定摘要（详见 [httpsvr.md](./httpsvr.md)）：

1. 同一个 `*httpsvr.Context` 贯穿整个请求；`Handler` 是 `func(*Context)`。
2. 中间件、控制器若要继续链路，必须调用 `c.Next()`；否则用 `c.StopExecution()` 明确终止。
3. 参数解析一律返回 `(value, *ae.Error)`；错误直接 `c.WriteE(e)` 或 `c.TryWrite(v, e)` 输出。
4. 业务错误默认放在响应体 `code/msg` 中（HTTP 200）；需要时由客户端或服务端开启 `ErrorAsStatus`。
5. `rt.UseHooks(app.HTTP)` 让每个请求拿到 `*httpsvr.Settings`，不同 App 之间互不影响。

## 版本与兼容性

- Go 版本：`go.mod` 声明为 `go 1.27`（使用 `encoding/json/v2`、`slices`、泛型等）。
- HTTP 栈只依赖标准库 `net/http`，不依赖第三方 Web 框架。
- 第三方依赖仅用于配置与微服务：`hashicorp/consul/api`、`grpc`、`yaml.v3`、`golang.org/x/*`。
