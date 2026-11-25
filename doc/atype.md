# 业务类型（aa/atype）

`atype` 提供两类东西：**通用值包装** `Atype`，以及**业务语义类型**（金额、日期、版本、区划、分页等）。

## 1. Atype：值包装与转换

```go
v := atype.New("42")
defer v.Close() // 归还对象池

s := v.String()
n := v.DefaultInt64(0)
```

- `New(...)` 从 `sync.Pool` 取对象；`Close()` 归还。不归还也不会泄漏（GC 可回收），但会失去池化收益。
- 带 `Release` 前缀的方法会在返回前自动 `Close`：

```go
s := v.ReleaseString()
b := v.ReleaseBytes()
n := v.ReleaseDefaultInt64(0)
```

- `Reload(v any)` 替换内部值并返回自身，便于复用。
- `Raw()` 取原始值；`Get(keys...)` 读取嵌套 map（`Get("user", "1", "name")`）。

转换函数（包级）：

| 函数 | 说明 |
|------|------|
| `String(d any) string` | 覆盖 `string/[]byte/Char/Date/bool/各整数/浮点`；其他类型 `fmt.Sprint` |
| `Bytes(d any) []byte` | `[]byte` 直接返回；`string` 转换 |
| `Bool/Int8/.../Int64/Int64Base/Uint8/.../Uint64Base/Float32/Float64` | 失败返回 error |
| `Slice(v any) ([]any, error)` | 反射取切片 |
| `ToAnySlice[T]` / `ToStrings[T]` | 泛型转换 |

> 注意：`byte` 是 `uint8` 别名，`String(byte('A'))` 得到 `"97"`；要得到 `"A"` 请用 `atype.Char('A')`。

## 2. 金额与小数

| 类型 | 说明 |
|------|------|
| `Decimal` | 定点小数（`Thousandth = 10` 等常量定义精度换算） |
| `Money` | 金额，`MoneyScale = 4`（4 位小数） |
| `VMoney` | 虚拟/展示金额，`VCent = 100` |
| `DecimalFormat` | 千分位、保留位数、舍入方式 |
| `RoundType` | `Round/Floor/Ceil/RoundReverse/RoundTrim/RoundAway` |

```go
m, e := c.QueryAtype.Money("price", atype.Money(1), atype.Money(1000000))
```

解析规则：字符串 → 定点整数，带范围校验；格式化时按 `DecimalFormat` 输出。

## 3. 日期与时间

| 类型 | 说明 |
|------|------|
| `Date` | `Y-m-d`，字符串常量形式 |
| `Datetime` | `Y-m-d H:i:s` |
| `Timestamp` | 秒级时间戳 |
| `Second` | `time.Duration` 的秒表示 |
| `Weekday` | 周一~周日，`ToWeekday("mon")` / `NewWeekday(1)` |
| `Year` / `YearMonth` / `YMD` | 业务日期片段 |

`atype.EnableZeroDate` 控制是否允许零值日期（默认 `true`）。
时间格式与时区来自 `aconfig.TimeFormat` / `aconfig.TimeLocation`。

## 4. 分页

```go
type Paging struct {
	Page     uint   `json:"page"`
	PageEnd  uint   `json:"page_end"`
	PageSize uint8  `json:"page_size"`
	Offset   uint   `json:"offset"`
	Limit    uint16 `json:"limit"`
	Prev     uint   `json:"prev"`
	Next     uint   `json:"next"`
}
```

```go
p := c.QueryAtype.Paging()                    // page / page_size / page_end
p = c.QueryAtype.PagingWithSize(50)           // 限制最大 page_size
p = c.QueryAtype.PagingWithFirstPageEnd(1000) // 首页 page_end 缺省值
```

- 默认 `page_size = atype.DefaultPageSize`（10）。
- `Offset/Limit/Prev/Next` 由 `NewPaging` 计算，直接用于数据库查询。

## 5. 路径与文件

| 类型 | 用途 |
|------|------|
| `Path` / `StdPath` | Unicode / 标准路径 |
| `Filename` / `StdFilename` | 文件名 |
| `Ext` / `Extension` | `Ext("png").Extension() == ".png"` |
| `PathPattern` / `FilenamePattern` / `UrlPattern` | 形如 `/upload/{uid}/avatar.png` 的模板，`ReplaceAll(name, type, to)` |
| `AudioPath(s)` / `VideoPath(s)` / `ImagePath(s)` | 带扩展名校验的业务路径 |
| `UUID`、`Email`、`Word`、`Alphabetical`、`AlphaDigits`、`Uppers` | 校验型字符串 |

解析入口：`ctx.BodyAtype.ImagePaths("files")`、`ctx.QueryAtype.Date("from", loc)` 等。

## 6. 版本

```go
v := atype.ToVersion("1.2.3-beta")
vs := v.Struct()             // VersionStruct{Major, Minor, Patch, Tag, Main}
v2 := atype.PackVersion(1, 2, 3, atype.TagRelease)
v.Compare(v2)
```

- 版本打包为 `Major*1_000_000 + Minor*1_000 + Patch`。
- `VersionTag`：`TagRelease`、`TagAlpha`、`TagBeta`、`TagRC` 等。
- `Version.MinorVersion()`、`Version.Add(...)` 用于灰度/升级判断。

## 7. 行政区划与地理位置

| 类型 | 说明 |
|------|------|
| `Distri` | 24 位区划编码（省 + 市 + 区） |
| `Province` / `Dist` | 从 `Distri` 派生 |
| `Coordinate` / `Location` / `Point` / `Position` | 经纬度、地址、坐标系（支持 SRID/顺序） |
| `IP` | 二进制/文本 IP，`OK()`、`Is4()`、`Net()`、`To16()` |

`Distri.Province()` / `Distri.Dist()` 用于按省/区聚合。

## 8. 可空包装

数据库可空列使用 `Null*` 类型，底层是 `sql.NullString`（或 JSON 文本）：

`NullString`、`NullInt64`、`NullFloat64`、`NullJson`、`NullUint8s`、`NullUint16s`、`NullUint24s`、`NullUint32s`、`NullUint64s`、`NullInt8s`、`NullInt16s`、`NullInt32s`、`NullInt64s`、`NullUints`、`NullableSid` 等。

共同模式：`NewXxx(s)`、`ToXxx(v)`、`Xxx()` 取切片，以及 `Valid` 字段。

> `NullUint8s` 使用数字数组 JSON（`[1,2,3]`），由 `MarshalUint8s` / `UnmarshalUint8s` 实现，避免 `encoding/json` 把 `[]byte` 编码成 Base64。

## 9. 枚举（aa/atype/aenum）

| 枚举 | 常量示例 |
|------|----------|
| `Sex` | `SexUnknown`、`SexMale`、`SexFemale` |
| `Status` | `Pending`、`Created`、`Passed`、`SysTakeover` 等业务状态 |
| `Country` | ISO 国家/地区 |
| `Continent` | 大洲 |

解析：`ctx.QueryAtype.Sex("sex")`、`ctx.QueryAtype.Status("status")`、`ctx.QueryAtype.StatusEnum("status", aenum.Passed)`。

## 10. 校验函数

```go
atype.IsEmail(s)
atype.IsUUID(s)
atype.IsPath(s) / IsUnicodePath(s)
atype.IsFilename(s) / IsUnicodeFilename(s)
atype.IsWord(s) / IsAlphabetical(s) / IsNumberString(s) / IsBin(s)
atype.IsLower/IsLower(s) / IsUpper/IsUppers(s)
```

## 11. 在 HTTP 层使用

```go
// 查询参数命名空间
page := c.QueryAtype.Paging()
sex, e := c.QueryAtype.Sex("sex")
from, e := c.QueryAtype.Datetime("from", cfg.TimeLocation)

// 请求体命名空间
avatars, e := c.BodyAtype.ImagePaths("avatars", true)
```

`httpsvr/atype` 只是适配层：`Query{Src QuerySource}` / `Body{Src BodySource}`，便于测试时替换数据源。

## 12. 约定

- 金额统一用 `Money`/`Decimal`，不要用 `float64` 传递金额。
- 时间统一用 `Datetime`/`Timestamp`，不要用字符串裸传。
- 业务编码（区划、状态）统一用对应类型，避免魔法数字。
- JSON 输出前若含 `int64/uint64` 且客户端是 JS，开启 `X-Stringify`（见 [httpsvr.md](./httpsvr.md#65-bigint-字符串化与字段过滤)）。
