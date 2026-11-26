package driver

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/alog"
	"github.com/aarioai/aago/pkg/types"
	"github.com/aarioai/aago/pkg/utils"
	"github.com/redis/go-redis/v9"
)

var (
	REDIS        = "redis"
	redisClients sync.Map
)

// NewRedis
// Note: better use NewRedisPool instead
func NewRedis(app *aa.App, section string) (*redis.Client, *ae.Error) {
	opts, err := ParseRedisConfig(app, section)
	if err != nil {
		return nil, newConfigError(REDIS, section, err)
	}
	return redis.NewClient(opts), nil
}

// NewRedisPool go-redis 是redis官方推出的，自带连接池、线程安全，不必手动操作
// https://redis.uptrace.dev/guide/go-redis-debugging.html#connection-pool-size
// Warning: Do not unset the returned client as it is managed by the pool
// Warning: 使用完不要unset client，释放是错误人为操作，可能会导致其他正在使用该client的线程panic，这里不做过度处理。
func NewRedisPool(app *aa.App, section string) (*redis.Client, *ae.Error) {
	key := normalizePoolSection(REDIS, section)
	client, ok := redisClients.Load(key)
	if ok {
		if client != nil {
			return client.(*redis.Client), nil
		}
		redisClients.Delete(key)
	}
	created, e := NewRedis(app, section)
	if e != nil {
		return nil, e
	}
	actual, loaded := redisClients.LoadOrStore(key, created)
	if loaded {
		_ = created.Close()
		return actual.(*redis.Client), nil
	}
	return created, nil
}

// CloseRedisPool
// Each process should utilize a single connection, which is managed by the main function.
// This connection should be closed when the main function terminates.
func CloseRedisPool() {
	redisClients.Range(func(k, v interface{}) bool {
		redisClients.Delete(k)
		client := v.(*redis.Client)
		if client != nil {
			alog.Stopf("redis client: %s", k)
			_ = client.Close()
		}
		return true
	})
}

func ParseRedisConfig(app *aa.App, section string) (*redis.Options, error) {
	addr, err := tryGetSectionCfgWithFallback(app, REDIS, section, "addr", "Addr")
	if err != nil {
		return nil, err
	}
	network := getSectionCfgWithFallback(app, REDIS, section, "net", "Net", "network", "Network")
	nodeAddr := getSectionCfgWithFallback(app, REDIS, section, "nodeAddr", "NodeAddr", "node_addr", "nodeAddress", "NodeAddress", "node_address")
	clientName := getSectionCfgWithFallback(app, REDIS, section, "clientName", "ClientName", "client_name")
	protocol := getSectionCfgWithFallback(app, REDIS, section, "protocol", "Protocol")
	user := getSectionCfgWithFallback(app, REDIS, section, "user", "User", "username", "Username")         // username 可以为空
	password := getSectionCfgWithFallback(app, REDIS, section, "passwd", "Passwd", "password", "Password") // password 可以为空
	db := getSectionCfgWithFallback(app, REDIS, section, "db", "Db", "DB", "database", "Database")
	maxRetries := getSectionCfgWithFallback(app, REDIS, section, "maxRetries", "MaxRetries", "max_retries")
	minRetryBackoff, err1 := getSectionCfgDurationWithFallback(app, REDIS, section, "minRetryBackoff", "MinRetryBackoff", "min_retry_backoff")
	maxRetryBackoff, err2 := getSectionCfgDurationWithFallback(app, REDIS, section, "maxRetryBackoff", "MaxRetryBackoff", "max_retry_backoff")
	dialerRetries := getSectionCfgWithFallback(app, REDIS, section, "dialerRetries", "DialerRetries", "dialer_retries")
	dialerRetryTimeout, err3 := getSectionCfgDurationWithFallback(app, REDIS, section, "dialerRetryTimeout", "DialerRetryTimeout", "dialer_retry_timeout")
	authEmptyPassword, err4 := getSectionCfgBoolWithFallback(app, REDIS, section, "authEmptyPassword", "AuthEmptyPassword", "auth_empty_password")

	var connTimeout, readTimeout, writeTimeout time.Duration
	if timeout := getSectionCfgWithFallback(app, REDIS, section, "timeout", "Timeout"); timeout != "" {
		connTimeout, readTimeout, writeTimeout = parseTimeouts(timeout)
	}
	dialTimeout, err5 := getSectionCfgDurationWithFallback(app, REDIS, section, "dialTimeout", "DialTimeout", "dial_timeout")
	if dialTimeout == 0 {
		dialTimeout = connTimeout
	}

	contextTimeoutEnabled, err6 := getSectionCfgBoolWithFallback(app, REDIS, section, "contextTimeoutEnabled", "ContextTimeoutEnabled", "context_timeout_enabled")
	readBufferSize := getSectionCfgWithFallback(app, REDIS, section, "readBufferSize", "ReadBufferSize", "read_buffer_size")
	writeBufferSize := getSectionCfgWithFallback(app, REDIS, section, "writeBufferSize", "WriteBufferSize", "write_buffer_size")
	pipelineReadBufferSize := getSectionCfgWithFallback(app, REDIS, section, "pipelineReadBufferSize", "PipelineReadBufferSize", "pipeline_read_buffer_size")
	pipelineWriteBufferSize := getSectionCfgWithFallback(app, REDIS, section, "pipelineWriteBufferSize", "PipelineWriteBufferSize", "pipeline_write_buffer_size")
	pipelinePoolSize := getSectionCfgWithFallback(app, REDIS, section, "pipelinePoolSize", "PipelinePoolSize", "pipeline_pool_size")
	pipelineMinIdleConns := getSectionCfgWithFallback(app, REDIS, section, "pipelineMinIdleConns", "PipelineMinIdleConns", "pipeline_min_idle_conns")

	poolFIFO, err7 := getSectionCfgBoolWithFallback(app, REDIS, section, "poolFIFO", "PoolFIFO", "pool_fifo")
	poolSize := getSectionCfgWithFallback(app, REDIS, section, "poolSize", "PoolSize", "pool_size")
	maxConcurrentDials := getSectionCfgWithFallback(app, REDIS, section, "maxConcurrentDials", "MaxConcurrentDials", "max_concurrent_dials")
	poolTimeout, err8 := getSectionCfgDurationWithFallback(app, REDIS, section, "poolTimeout", "PoolTimeout", "pool_timeout")
	minIdleConns := getSectionCfgWithFallback(app, REDIS, section, "minIdleConns", "MinIdleConns", "min_idle_conns")
	maxIdleConns := getSectionCfgWithFallback(app, REDIS, section, "maxIdleConns", "MaxIdleConns", "max_idle_conns")
	maxActiveConns := getSectionCfgWithFallback(app, REDIS, section, "maxActiveConns", "MaxActiveConns", "max_active_conns")
	connMaxIdleTime, err9 := getSectionCfgDurationWithFallback(app, REDIS, section, "connMaxIdleTime", "ConnMaxIdleTime", "conn_max_idle_time")
	connMaxLifetime, err10 := getSectionCfgDurationWithFallback(app, REDIS, section, "connMaxLifetime", "ConnMaxLifetime", "conn_max_lifetime")
	connMaxLifetimeJitter, err11 := getSectionCfgDurationWithFallback(app, REDIS, section, "connMaxLifetimeJitter", "ConnMaxLifetimeJitter", "conn_max_lifetime_jitter")
	connMaxLifetimeReapInterval, err12 := getSectionCfgDurationWithFallback(app, REDIS, section, "connMaxLifetimeReapInterval", "ConnMaxLifetimeReapInterval", "conn_max_lifetime_reap_interval")

	disableIdentity, err13 := getSectionCfgBoolWithFallback(app, REDIS, section, "disableIdentity", "DisableIdentity", "disable_identity")
	identitySuffix := getSectionCfgWithFallback(app, REDIS, section, "identitySuffix", "IdentitySuffix", "identity_suffix")
	unstableResp3, err14 := getSectionCfgBoolWithFallback(app, REDIS, section, "unstableResp3", "UnstableResp3", "unstable_resp3")
	failingTimeoutSeconds, err15 := parseRedisSeconds(getSectionCfgWithFallback(app, REDIS, section, "failingTimeoutSeconds", "FailingTimeoutSeconds", "failing_timeout_seconds"))
	clientSideCacheStrategy := getSectionCfgWithFallback(app, REDIS, section, "clientSideCacheStrategy", "ClientSideCacheStrategy", "client_side_cache_strategy")
	clientSideCacheRefreshOnInvalidate, err16 := getSectionCfgBoolWithFallback(app, REDIS, section, "clientSideCacheRefreshOnInvalidate", "ClientSideCacheRefreshOnInvalidate", "client_side_cache_refresh_on_invalidate")
	clientSideCacheCoalesceMisses, err17 := getSectionCfgBoolWithFallback(app, REDIS, section, "clientSideCacheCoalesceMisses", "ClientSideCacheCoalesceMisses", "client_side_cache_coalesce_misses")
	clientSideCacheInvalidationBatchWindow, err18 := getSectionCfgDurationWithFallback(app, REDIS, section, "clientSideCacheInvalidationBatchWindow", "ClientSideCacheInvalidationBatchWindow", "client_side_cache_invalidation_batch_window")

	if err := ae.FirstError(err1, err2, err3, err4, err5, err6, err7, err8, err9, err10, err11, err12, err13, err14, err15, err16, err17, err18); err != nil {
		return nil, err
	}

	opt := redis.Options{
		Network:                                network,
		Addr:                                   addr, //  127.0.0.1:6379
		NodeAddress:                            nodeAddr,
		ClientName:                             clientName, // executes `CLIENT SETNAME ClientName` for each conn
		Protocol:                               types.ToInt(protocol),
		Username:                               user,
		Password:                               password,
		AuthEmptyPassword:                      authEmptyPassword,
		DB:                                     types.ToInt(db),
		MaxRetries:                             types.ToInt(maxRetries),
		MinRetryBackoff:                        minRetryBackoff,
		MaxRetryBackoff:                        maxRetryBackoff,
		DialTimeout:                            dialTimeout,
		DialerRetries:                          types.ToInt(dialerRetries),
		DialerRetryTimeout:                     dialerRetryTimeout,
		ReadTimeout:                            readTimeout,
		WriteTimeout:                           writeTimeout,
		ContextTimeoutEnabled:                  contextTimeoutEnabled,
		ReadBufferSize:                         types.ToInt(readBufferSize),
		WriteBufferSize:                        types.ToInt(writeBufferSize),
		PipelineReadBufferSize:                 types.ToInt(pipelineReadBufferSize),
		PipelineWriteBufferSize:                types.ToInt(pipelineWriteBufferSize),
		PipelinePoolSize:                       types.ToInt(pipelinePoolSize),
		PipelineMinIdleConns:                   types.ToInt(pipelineMinIdleConns),
		PoolFIFO:                               poolFIFO,
		PoolSize:                               types.ToInt(poolSize),
		MaxConcurrentDials:                     types.ToInt(maxConcurrentDials),
		PoolTimeout:                            poolTimeout,
		MinIdleConns:                           types.ToInt(minIdleConns),
		MaxIdleConns:                           types.ToInt(maxIdleConns),
		MaxActiveConns:                         types.ToInt(maxActiveConns),
		ConnMaxIdleTime:                        connMaxIdleTime,
		ConnMaxLifetime:                        connMaxLifetime,
		ConnMaxLifetimeJitter:                  connMaxLifetimeJitter,
		ConnMaxLifetimeReapInterval:            connMaxLifetimeReapInterval,
		DisableIdentity:                        disableIdentity,
		IdentitySuffix:                         identitySuffix,
		UnstableResp3:                          unstableResp3,
		FailingTimeoutSeconds:                  failingTimeoutSeconds,
		ClientSideCacheStrategy:                redis.CSCStrategy(types.ToInt(clientSideCacheStrategy)),
		ClientSideCacheRefreshOnInvalidate:     clientSideCacheRefreshOnInvalidate,
		ClientSideCacheCoalesceMisses:          clientSideCacheCoalesceMisses,
		ClientSideCacheInvalidationBatchWindow: clientSideCacheInvalidationBatchWindow,
	}
	return &opt, nil
}

// parseRedisSeconds accepts a plain second count or a duration string such as "15s".
func parseRedisSeconds(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	if n, err := strconv.Atoi(value); err == nil {
		return n, nil
	}
	d, err := types.ParseDuration(value)
	if err != nil {
		return 0, err
	}
	return int((d + time.Second - 1) / time.Second), nil
}

// NewRedisError 处理 Redis 错误
// @TODO
func NewRedisError(err error, details ...any) *ae.Error {
	if err == nil {
		return nil
	}
	if errors.Is(err, redis.Nil) {
		return ae.New(ae.NotFound, "redis key not found").WithDetail(details...)
	}
	msg := err.Error()
	caller := utils.Caller(1)
	return ae.New(ae.InternalServerError, caller+" redis: "+msg).WithDetail(details...)
}
