---
name: aago-redisx-cache
description: >-
  Use the aago driver redisx helpers: hash writes with TTL management, hash reads with
  nil-tolerant variants, pipelined counters, atomic locks, and hourly/daily time-slot keys.
  Use when implementing caching, counters, rate metrics, or locks on Redis.
---

# Redis caching with redisx

## When to use this skill

- Adding a Redis-backed cache, hash payload, counter, leaderboard slice or lock.
- Fixing "counter always returns 0", "TTL keeps growing", or "lock always succeeds" style bugs.
- Choosing between hourly/daily time-slot keys for metric aggregation.

## Ground rules

1. **Client comes from the pool.** `driver.NewRedisPool(app, section)`; never `Close()` it.
2. **TTL is managed for you.** `HSet`/`HIncrBy` only set a TTL when the key has none, so steady
   writes do not extend the lifetime. Pass `expires <= 0` to leave the TTL untouched.
3. **Pipelined reads happen after `Pipelined` returns.** Use the helpers, not ad-hoc pipelines.
4. **Errors are `error` from the client, `*ae.Error` from helpers.**

## Writes

```go
// pairs, slices, maps or structs with `redis` tags
err := redisx.HSet(ctx, rdb, time.Hour, "user:100", map[string]any{"name": "Aario", "age": 18})

type cache struct {
    Name string `redis:"name"`
    Age  int    `redis:"age,omitempty"`
}
err = redisx.HSet(ctx, rdb, redisx.DefaultTTL, "user:100", cache{Name: "Aario"})
```

Invalid input (no arguments, nil struct pointer, odd argument count, struct without tags) returns
`ae.ErrInvalidInput` or `redisx.ErrInvalidHashArgs` before touching Redis.

## Reads

```go
vals, ok, err := redisx.TryHMGetString(ctx, rdb, "user:100", "name", "age") // ok=false when all nil
vals, err = redisx.MustHMGet(ctx, rdb, "user:100", "name", "age")           // error on any nil
m, err := redisx.HGetAllInt(ctx, rdb, "counters:hourly", true)               // true: strict parse
err = redisx.HScan(ctx, rdb, "user:100", &dst, "name", "age")
err = redisx.HGetAll(ctx, rdb, "user:100", &dst)
```

## Counters

```go
n, e := redisx.HIncr(ctx, rdb, redisx.HourlyTTL, "views:2026-11-25", "post:100")
n, e = redisx.HIncrBy(ctx, rdb, redisx.HourlyTTL, "views:2026-11-25", "post:100", -1)
ns, e := redisx.HMIncrIds(ctx, rdb, redisx.HourlyTTL, "views", ids)
```

## Locks

```go
if e := redisx.ApplyLock(ctx, rdb, 10*time.Second, "report:2026-11-25"); e != nil {
    if e.Code == ae.Conflict {
        return // another worker holds the lock
    }
    return
}
```

`ApplyLock` returns a conflict error when `SET NX` does not acquire the key.

## Time-slot keys

```go
key := redisx.HourlyKey(6, false, func(i uint8) string { return fmt.Sprintf("stat:h%d", i) })
keys := redisx.HourlyKeys(6, true, func(i uint8) string { return fmt.Sprintf("stat:h%d", i) })
```

- `interval` must divide 24 (hours) or 30 (days); anything else collapses to one slot.
- `BatchKeys` puts the current slot last, so aggregation can read history first.
- Constants: `HourlyTTL` (24h), `DailyTTL` (72h), `DefaultTTL` (1h).

## Verification

- Argument validation can be unit-tested with a `nil` client (the helpers return before use).
- `go test ./driver/redisx/...`, `go vet ./...`.
- Full reference: `../../../../aago/doc/driver/redisx.md`.
