# 编码约定

本文件约束框架与业务代码的共同风格。已有代码若与此不一致，以"新代码遵守"为原则逐步收敛。

## 1. 包与依赖

- `pkg/*` 不依赖 `aa/*`。
- `aa/httpsvr` 不依赖 `aa/aconfig`、`aa/alog`。
- 业务代码依赖框架，框架不反向依赖业务。
- 新增工具优先放 `pkg/`（无状态、可复用）；与框架生命周期相关的放 `aa/`。

## 2. HTTP 处理器

```go
func getUser(c *httpsvr.Context) {
	uid, e := c.QueryUint64("uid")
	if e != nil {
		_, _ = c.WriteE(e)
		return
	}
	data, e := svc.User(c.StdContext(), uid)
	if e != nil {
		_, _ = c.WriteE(e)
		return
	}
	_, _ = c.WriteData(data)
}
```

约定：

1. **不要忽略错误**。参数错误与业务错误都通过 `*ae.Error` 返回并由 `WriteE` / `TryWrite` 输出。
2. 中间件与多 handler 路由：继续链路必须 `c.Next()`；推荐 `defer c.Next()`。
3. 需要中断时显式 `c.StopExecution()`，不要依赖"忘记调用 Next"。
4. 不要在 goroutine 中使用 `*httpsvr.Context`；只传 `c.StdContext()` 与纯数据。
5. 不要直接写 HTTP 状态码来表达业务错误（除非开启 `ErrorAsStatus` 或健康检查类接口）。
6. 大响应 / 高频响应优先使用 `WriteData`；纯文本用 `WriteString`。

## 3. 错误处理（ae）

- 业务错误统一 `*ae.Error`，使用语义化构造器：
  `ae.NewBadParam`、`ae.NewConflict`、`ae.NewUnsupportedMedia`、`ae.NewErr(err, detail)`。
- 常数错误在包级初始化并 `Lock()`。
- **成功路径不要创建 `ae.Error`**：`ae.New` 会采样调用栈。
- 服务端错误（5xx）需要记录日志；客户端错误（4xx）通常不需要。
- 标准 `error` 与 `*ae.Error` 不混用：`*ae.Error` 不实现 `error` 接口。

```go
var ErrUserNotFound = ae.NewBadParam("uid", "user not found").Lock()

if user == nil {
	return nil, ErrUserNotFound
}
```

## 4. 日志

- 使用 `alog.LogInterface`，不要在框架代码里直接 `log.Print`（`helpers/debug` 与 `httpc` 除外，它们面向标准库日志重定向）。
- 需要上下文时传 `ctx`：`app.Log.E(ctx, e)`、`app.Log.Errorf(ctx, ...)`。
- 生产级别建议 `LevelInfo` 或更高；调试信息用 `Debugf`。
- 日志内容不要包含密码、token、完整身份证/手机号。

## 5. 参数解析

- 必填：`QueryString(p, true)`、`QueryUint64(p)`（默认 required=true）。
- 可选：`QueryString(p, false)`、`QueryFast(p)`（不返回错误）。
- 带默认值：`QueryString(p, false, atype.New("x"))`。
- 正则：预编译 `regexp.MustCompile` 后传入，避免每次编译。
- 批量/数组参数用 `QueryInt64s` / `BodyStrings` 等，不要手工 split。

## 6. 类型

- 金额用 `atype.Money` / `atype.Decimal`，禁止 `float64` 传递金额。
- 时间用 `atype.Datetime` / `atype.Timestamp`，时区统一取 `cfg.TimeLocation`。
- 业务编码（区划、状态、性别）用 `atype` / `aenum` 类型。
- `Atype` 用完后 `Close()`，或使用 `Release*` 方法。

## 7. 命名与注释

- 导出符号必须有注释，首词为符号名（Go 惯例）。
- 注释解释"为什么"，不复述"是什么"。
- 缩写保持项目习惯：`Ctx`、`ID`、`URL`、`HTTP`、`JSON`、`DTO`、`MW`。
- 包注释写在 `doc.go` 或主文件顶部（根包见 [airis.go](../airis.go)）。

## 8. 性能约定

- 热路径（每请求执行）避免：反射、正则编译、创建错误、字符串拼接、`fmt.Sprintf`。
- 需要缓存时优先放在 `Context`（请求级）或包级只读变量（进程级）。
- 修改热路径必须给出基准数据，见 [performance.md](./performance.md)。
- 不要用 `sync.Pool` 池化包含指针且生命周期不确定的对象，除非能保证 `Close`。

## 9. 测试

| 类型 | 位置 | 要求 |
|------|------|------|
| 单元测试 | 与源码同包的 `_test.go` | 覆盖边界（空值、越界、非法输入） |
| HTTP 测试 | `aa/httpsvr/*_test.go` | 用 `httptest.NewRequest` + `httptest.NewRecorder` |
| 基准 | `*_bench_test.go` | `b.ReportAllocs()` + `b.ResetTimer()` |
| 压力测试 | `stress_test.go` | `-short` 下跳过 |

约定：

- 表驱动测试优先；错误信息包含输入与期望。
- 新增行为必须补测试；修复缺陷时先写会失败的用例（或至少覆盖该边界）。
- 测试不要依赖外部网络/Consul；需要时用 `httptest` 或接口替身。

```go
func TestFoo(t *testing.T) {
	tests := []struct{ name, in string; want int }{
		{"empty", "", 0},
		{"normal", "42", 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Foo(tt.in); got != tt.want {
				t.Fatalf("Foo(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
```

## 10. 提交前检查

```bash
gofmt -l .          # 必须无输出
go build ./...
go test ./...
go vet ./...        # 必须无输出
```

涉及 HTTP 热路径时追加：

```bash
go test ./aa/httpsvr -run '^$' -bench 'BenchmarkHTTPX|BenchmarkRouter' -benchmem -count=3
```

## 11. 兼容性

- 导出的结构体字段与函数签名变更视为破坏性变更，需要同步更新 `doc/` 与 skill。
- 新增可选行为用可选参数（`...bool`、`...string`）或新方法，避免改变现有语义。
- 修改响应格式（`Body` 字段、默认 Content-Type）需在文档中标注。

## 12. 文档

- 架构/用法变化更新 `doc/` 对应文件。
- 性能相关改动更新 [performance.md](./performance.md)。
- 新增框架能力时同步更新 `.github/skills/aago-framework`。
