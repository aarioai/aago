# redisx — Redis helpers

Package: `github.com/aarioai/aago/driver/redisx`

## TTL constants

```go
redisx.HourlyTTL  // 24h — one key per hour, kept for a day
redisx.DailyTTL   // 72h — one key per day, kept for three days
redisx.DefaultTTL // 1h
```

> 中文提示：TTL 特意设置得比时间片长，宕机恢复后旧键仍然可用。

## Hashes

```go
err := redisx.HSet(ctx, rdb, time.Hour, "k", "f1", "v1")                       // pairs
err = redisx.HSet(ctx, rdb, time.Hour, "k", []string{"f1", "v1"})              // slice
err = redisx.HSet(ctx, rdb, time.Hour, "k", map[string]any{"f1": "v1"})        // map
err = redisx.HSet(ctx, rdb, time.Hour, "k", struct{                       // struct
    A string `redis:"a"`
    B int    `redis:"b,omitempty"`
}{A: "x"})
```

Rules:

- The key TTL is only (re)set when the key has none, so steady updates do not extend it.
- `expires <= 0` writes the fields without touching the TTL.
- Invalid input returns `ae.ErrInvalidInput` (nil struct pointer, no arguments) or
  `redisx.ErrInvalidHashArgs` (odd number of arguments, unsupported type, struct without tags).

Reading:

```go
values, ok, err := redisx.TryHMGetString(ctx, rdb, "k", "a", "b") // ok=false when all fields are nil
values, err = redisx.MustHMGet(ctx, rdb, "k", "a", "b")          // ErrUnexpectedNilResult when any field is missing
m, err := redisx.HGetAllInt(ctx, rdb, "k", true)                  // map[string]int
err = redisx.HScan(ctx, rdb, "k", &dest, "a", "b")
err = redisx.HGetAll(ctx, rdb, "k", &dest)
```

> 中文提示：`HSet` 之前的结构体分支总是返回 `ErrInvalidInput`，且会污染调用方的切片，现已修复。

## Counters

```go
n, e := redisx.HIncr(ctx, rdb, time.Hour, "counter", "views")        // +1
n, e = redisx.HIncrBy(ctx, rdb, time.Hour, "counter", "views", -5)
ns, e := redisx.HMIncr(ctx, rdb, time.Hour, "counter", []string{"a", "b"})
ns, e = redisx.HMIncrIds(ctx, rdb, time.Hour, "counter", ids)
```

Pipelined increments read their values **after** `Pipelined` returns, so counters now report the
real value (earlier versions returned 0 for pipelined calls).

> 中文提示：`HIncrBy` / `HMIncr` 之前在 pipeline 内读取结果，永远返回 0，现已修正。

## Locks

```go
if e := redisx.ApplyLock(ctx, rdb, 10*time.Second, "report:2026-11-25"); e != nil {
    if e.Code == ae.Conflict { /* already locked */ }
}
```

`ApplyLock` uses `SET NX`; a failed acquisition returns a conflict error instead of `nil`.

> 中文提示：抢锁失败现在返回冲突错误，不再静默成功。

## Time-slot keys

```go
key := redisx.HourlyKey(6, false, func(i uint8) string { return fmt.Sprintf("stat:%d", i) })
keys := redisx.HourlyKeys(6, true, func(i uint8) string { return fmt.Sprintf("stat:%d", i) })
key = redisx.DailyKey(5, false, func(i uint8) string { return fmt.Sprintf("stat:%d", i) })
```

`interval` must divide 24 (hours) or 30 (days); an invalid interval falls back to a single slot.
`BatchKeys` puts the current slot last so callers can aggregate historical slots first.

> 中文提示：`HourlyKeys` 会把当前时段排在最后，便于先聚合历史数据。

## Misc

```go
ids, e := redisx.Uint64s(rdb.SUnion(ctx, keys...).Result())
```
