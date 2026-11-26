package redisx

import (
	"context"
	"log"
	"time"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver"
	"github.com/aarioai/aago/pkg/types"
	"github.com/redis/go-redis/v9"
)

// 一般用于 SUnion
func Uint64s(vs []string, err error) ([]uint64, *ae.Error) {
	if err != nil {
		return nil, driver.NewRedisError(err)
	}
	if len(vs) == 0 {
		return nil, ae.ErrorNoRowsAvailable
	}
	ids := make([]uint64, len(vs))
	for i, v := range vs {
		ids[i], err = types.ParseUint64(v)
		if err != nil {
			log.Printf("redis uint64s %s is not uint64\n", v)
		}
	}
	return ids, nil
}
func HIncrBy(ctx context.Context, rdb *redis.Client, expires time.Duration, k string, field string, incr int64) (int64, *ae.Error) {
	if expires <= 0 {
		reply, err := rdb.HIncrBy(ctx, k, field, incr).Result()
		return reply, driver.NewRedisError(err)
	}
	ttl, _ := rdb.TTL(ctx, k).Result()
	if ttl > 0 && ttl < expires {
		reply, err := rdb.HIncrBy(ctx, k, field, incr).Result()
		return reply, driver.NewRedisError(err)
	}

	var incrCmd *redis.IntCmd
	var expireCmd *redis.BoolCmd
	// Command results are only available after the pipeline has been executed.
	_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		incrCmd = pipe.HIncrBy(ctx, k, field, incr)
		expireCmd = pipe.Expire(ctx, k, expires)
		return nil
	})
	if err != nil {
		return 0, driver.NewRedisError(err)
	}
	if err = ae.FirstError(incrCmd.Err(), expireCmd.Err()); err != nil {
		return 0, driver.NewRedisError(err)
	}
	return incrCmd.Val(), nil
}

func HIncr(ctx context.Context, rdb *redis.Client, ttl time.Duration, k string, field string) (int64, *ae.Error) {
	return HIncrBy(ctx, rdb, ttl, k, field, 1)
}

func HMIncr(ctx context.Context, rdb *redis.Client, expires time.Duration, k string, fields []string) ([]int64, *ae.Error) {
	if len(fields) == 0 {
		return nil, ae.ErrorEmptyInput
	}
	cmds := make([]*redis.IntCmd, len(fields))
	var expireCmd *redis.BoolCmd
	// Preserve an existing TTL shorter than the requested expiry; otherwise
	// refresh it to avoid extending the key beyond the requested lifetime.
	ttl, _ := rdb.TTL(ctx, k).Result()

	_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for i, field := range fields {
			cmds[i] = pipe.HIncrBy(ctx, k, field, 1)
		}
		if expires > 0 && (ttl <= 0 || ttl >= expires) {
			expireCmd = pipe.Expire(ctx, k, expires)
		}
		return nil
	})
	if err != nil {
		return nil, driver.NewRedisError(err)
	}
	if expireCmd != nil && expireCmd.Err() != nil {
		return nil, driver.NewRedisError(expireCmd.Err())
	}

	replies := make([]int64, len(fields))
	for i, cmd := range cmds {
		if err = cmd.Err(); err != nil {
			return nil, driver.NewRedisError(err)
		}
		replies[i] = cmd.Val()
	}
	return replies, nil
}

func HMIncrIds(ctx context.Context, rdb *redis.Client, expires time.Duration, k string, ids []uint64) ([]int64, *ae.Error) {
	fields := make([]string, len(ids))
	for i, id := range ids {
		fields[i] = types.FormatUint(id)
	}
	return HMIncr(ctx, rdb, expires, k, fields)
}
