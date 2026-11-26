package redisx

import (
	"context"
	"time"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver"
	"github.com/redis/go-redis/v9"
)

// 申请原子性的锁
// ApplyLock sets a lock key only when it does not exist yet.
// It returns ae.NewConflict when the lock is already held, nil on success.
func ApplyLock(ctx context.Context, rdb *redis.Client, expires time.Duration, k string) *ae.Error {
	ok, err := rdb.SetNX(ctx, k+":lock", 1, expires).Result()
	if err != nil {
		return driver.NewRedisError(err)
	}
	if !ok {
		return ae.NewConflict(k)
	}
	return nil
}
