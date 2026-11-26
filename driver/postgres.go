package driver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/alog"
	"github.com/aarioai/aago/pkg/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	POSTGRESQL = "postgresql"

	// https://www.postgresql.org/docs/current/errcodes-appendix.html
	pgErrCodeUniqueViolation = "23505"

	postgresConnDoneMsg    = "postgres conn done: "
	postgresTxDoneMsg      = "postgres tx done: "
	postgresTooManyRowsMsg = "postgres too many rows: "
	postgresTimeoutMsg     = "postgres timeout: "
	postgresErrorMsg       = "postgres error: "
)

type PostgresConfig struct {
	Config *pgxpool.Config
}

type PostgresClientData struct {
	Client   *pgxpool.Pool
	Database string
}

var (
	postgresClients sync.Map
	postgresPoolMu  sync.Mutex
)

// postgresEndpoint renders host:port, or the socket directory for unix sockets.
func postgresEndpoint(c *pgx.ConnConfig) string {
	if strings.HasPrefix(c.Host, "/") {
		return c.Host
	}
	return net.JoinHostPort(c.Host, strconv.Itoa(int(c.Port)))
}

// NewPostgres creates an independent pgx pool. Prefer NewPostgresPool for
// application-wide shared pools; the caller owns and must close this pool.
func NewPostgres(app *aa.App, section string) (string, *pgxpool.Pool, *ae.Error) {
	f, err := ParsePostgresConfig(app, section)
	if err != nil {
		return "", nil, ae.NewErr(err)
	}
	alog.Printf("connect postgres: %s@%s %s", f.Config.ConnConfig.User, postgresEndpoint(f.Config.ConnConfig), f.Config.ConnConfig.Database)
	// pgxpool 不会立即建立数据库连接（除非 MinConns > 0），需要立即验证连接时请调用 Ping
	pool, err := pgxpool.NewWithConfig(context.Background(), f.Config)
	if err != nil {
		return "", nil, NewPostgresError(err, "open postgres: "+postgresEndpoint(f.Config.ConnConfig))
	}
	return f.Config.ConnConfig.Database, pool, nil
}

// NewPostgresPool returns the process-wide shared pool for the normalized
// PostgreSQL configuration section.
// Warning: Do not close the returned pool as it is managed by the pool
// Warning: 使用完不要关闭 pool，释放是错误人为操作，可能会导致其他正在使用该 pool 的线程panic，这里不做过度处理。
func NewPostgresPool(app *aa.App, section string) (string, *pgxpool.Pool, *ae.Error) {
	key := normalizePoolSection(POSTGRESQL, section)
	// Fast path: the pool is created once, so reads never need the lock.
	if cached, ok := postgresClients.Load(key); ok {
		if data := cached.(PostgresClientData); data.Client != nil {
			return data.Database, data.Client, nil
		}
	}

	postgresPoolMu.Lock()
	defer postgresPoolMu.Unlock()
	if cached, ok := postgresClients.Load(key); ok {
		data := cached.(PostgresClientData)
		if data.Client != nil {
			return data.Database, data.Client, nil
		}
		postgresClients.Delete(key)
	}
	database, pool, e := NewPostgres(app, section)
	if e != nil {
		return "", nil, e
	}
	postgresClients.Store(key, PostgresClientData{Client: pool, Database: database})
	return database, pool, nil
}

// ClosePostgresPool closes shared PostgreSQL pools during application shutdown.
func ClosePostgresPool() {
	postgresPoolMu.Lock()
	defer postgresPoolMu.Unlock()
	postgresClients.Range(func(key, value any) bool {
		postgresClients.Delete(key)
		data := value.(PostgresClientData)
		if data.Client != nil {
			alog.Stopf("postgres client: %s", key)
			data.Client.Close()
		}
		return true
	})
}

// ParsePostgresConfig see postgres_config_demo.yaml
//
// Advanced connection options (TLS certificates, service file, ...) should be
// specified through the `url` key; the discrete keys below cover the common case.
func ParsePostgresConfig(app *aa.App, section string) (PostgresConfig, error) {
	var f PostgresConfig

	dsn := getSectionCfgWithFallback(app, POSTGRESQL, section, "dsn", "Dsn", "DSN", "url", "Url", "URL")
	if dsn == "" {
		var err error
		if dsn, err = buildPostgresDSN(app, section); err != nil {
			return f, err
		}
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return f, err
	}
	if err := parsePostgresPoolConfig(app, section, c); err != nil {
		return f, err
	}
	f.Config = c
	return f, nil
}

// buildPostgresDSN builds a connection URL from the discrete keys. A
// `dsn`/`url` configuration takes precedence and bypasses this function.
func buildPostgresDSN(app *aa.App, section string) (string, error) {
	user := getSectionCfgWithFallback(app, POSTGRESQL, section, "user", "User", "username", "Username")
	if user == "" {
		return "", fmt.Errorf("required config not found: %s.%s.%s", POSTGRESQL, section, "user")
	}
	password := getSectionCfgWithFallback(app, POSTGRESQL, section, "passwd", "Passwd", "password", "Password") // password 可以为空
	database := getSectionCfgWithFallback(app, POSTGRESQL, section, "db", "Db", "DB", "database", "Database")
	if database == "" {
		database = normalizePoolSection(POSTGRESQL, section)
	}
	scheme := getSectionCfgWithFallback(app, POSTGRESQL, section, "scheme", "Scheme")
	if scheme == "" {
		scheme = "postgres"
	}
	network, host, port, err := parsePostgresAddr(app, section)
	if err != nil {
		return "", err
	}

	sslMode := getSectionCfgWithFallback(app, POSTGRESQL, section, "sslmode", "Sslmode", "SslMode", "SSLMode", "tls", "Tls", "TLS")
	if sslMode == "" {
		sslMode = "prefer"
	}

	u := &url.URL{
		Scheme: scheme,
		User:   url.UserPassword(user, password),
		Path:   "/" + database,
	}
	query := url.Values{}
	query.Set("sslmode", sslMode)
	if network == "unix" {
		// unix domain socket directory
		query.Set("host", host)
		query.Set("port", strconv.Itoa(port))
	} else {
		u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	}

	if v := getSectionCfgWithFallback(app, POSTGRESQL, section, "timeout", "Timeout"); v != "" {
		// 与 mysql 一致：connTimeout, readTimeout, writeTimeout，PostgreSQL 只用到 connTimeout
		if conn, _, _ := parseTimeouts(v); conn > 0 {
			// pgx 的 connect_timeout 以秒为单位
			query.Set("connect_timeout", strconv.Itoa(int((conn+time.Second-1)/time.Second)))
		}
	}
	if v := getSectionCfgWithFallback(app, POSTGRESQL, section, "applicationName", "ApplicationName", "application_name"); v != "" {
		query.Set("application_name", v)
	}
	if v := getSectionCfgWithFallback(app, POSTGRESQL, section, "searchPath", "SearchPath", "search_path"); v != "" {
		query.Set("search_path", v)
	}
	query.Set("timezone", app.Config.TimeLocation.String())

	u.RawQuery = query.Encode()
	return u.String(), nil
}

// parsePostgresAddr resolves the `net`/`addr` pair into a network, host and
// port. `net` defaults to "tcp", or "unix" when `addr` is an absolute path.
// Unix domain socket hosts are returned as a directory path.
func parsePostgresAddr(app *aa.App, section string) (network string, host string, port int, err error) {
	network = strings.ToLower(strings.TrimSpace(getSectionCfgWithFallback(app, POSTGRESQL, section, "net", "Net", "network", "Network")))
	addr := strings.TrimSpace(getSectionCfgWithFallback(app, POSTGRESQL, section, "addr", "Addr"))
	if addr == "" {
		return "", "", 0, fmt.Errorf("required config not found: %s.%s.addr", POSTGRESQL, section)
	}

	if network == "" {
		if strings.HasPrefix(addr, "/") {
			network = "unix"
		} else {
			network = "tcp"
		}
	}
	if network == "unix" || network == "unixpacket" || strings.HasPrefix(addr, "/") {
		// Unix socket addresses must include the socket filename and port.
		if dir, p, ok := splitPostgresSocket(addr); ok {
			return "unix", dir, p, nil
		}
		return "", "", 0, fmt.Errorf("invalid config %s.%s.addr: expected a Unix socket path ending in .s.PGSQL.<port>", POSTGRESQL, section)
	}

	h, p, err := splitPostgresHostPort(addr)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid config %s.%s.addr: %s", POSTGRESQL, section, err.Error())
	}
	return "tcp", h, p, nil
}

// splitPostgresHostPort accepts "host:port" or "[::1]:port".
func splitPostgresHostPort(addr string) (string, int, error) {
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	if h == "" {
		return "", 0, errors.New("missing host in address")
	}
	if p == "" {
		return "", 0, errors.New("missing port in address")
	}
	port, err := parsePostgresPort(p)
	if err != nil {
		return "", 0, err
	}
	return h, port, nil
}

func parsePostgresPort(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("missing port in address")
	}
	p, err := strconv.ParseUint(value, 10, 16)
	if err != nil || p == 0 {
		return 0, errors.New("invalid value for port: " + value)
	}
	return int(p), nil
}

// splitPostgresSocket reports whether the path points to a socket file
// (.s.PGSQL.<port>) and returns its directory and port.
func splitPostgresSocket(socket string) (string, int, bool) {
	name := path.Base(socket)
	if !strings.HasPrefix(name, ".s.PGSQL.") {
		return "", 0, false
	}
	p, err := strconv.ParseUint(strings.TrimPrefix(name, ".s.PGSQL."), 10, 16)
	if err != nil || p == 0 {
		return "", 0, false
	}
	return path.Dir(socket), int(p), true
}

func parsePostgresPoolConfig(app *aa.App, section string, config *pgxpool.Config) error {
	maxConns := getSectionCfgWithFallback(app, POSTGRESQL, section, "poolMaxConns", "PoolMaxConns", "pool_max_conns")
	minConns := getSectionCfgWithFallback(app, POSTGRESQL, section, "poolMinConns", "PoolMinConns", "pool_min_conns")
	minIdleConns := getSectionCfgWithFallback(app, POSTGRESQL, section, "poolMinIdleConns", "PoolMinIdleConns", "pool_min_idle_conns")
	maxConnLifetime := getSectionCfgWithFallback(app, POSTGRESQL, section, "poolMaxConnLifetime", "PoolMaxConnLifetime", "pool_max_conn_lifetime")
	maxConnLifetimeJitter := getSectionCfgWithFallback(app, POSTGRESQL, section, "poolMaxConnLifetimeJitter", "PoolMaxConnLifetimeJitter", "pool_max_conn_lifetime_jitter")
	maxConnIdleTime := getSectionCfgWithFallback(app, POSTGRESQL, section, "poolMaxConnIdleTime", "PoolMaxConnIdleTime", "pool_max_conn_idle_time")
	healthCheckPeriod := getSectionCfgWithFallback(app, POSTGRESQL, section, "poolHealthCheckPeriod", "PoolHealthCheckPeriod", "pool_health_check_period")

	maxConnsValue, err := parsePostgresInt32(maxConns)
	if err != nil {
		return fmt.Errorf("invalid config %s.%s.poolMaxConns: %s", POSTGRESQL, section, err.Error())
	}
	minConnsValue, err := parsePostgresInt32(minConns)
	if err != nil {
		return fmt.Errorf("invalid config %s.%s.poolMinConns: %s", POSTGRESQL, section, err.Error())
	}
	minIdleConnsValue, err := parsePostgresInt32(minIdleConns)
	if err != nil {
		return fmt.Errorf("invalid config %s.%s.poolMinIdleConns: %s", POSTGRESQL, section, err.Error())
	}
	maxConnLifetimeValue, err := parsePostgresDuration(maxConnLifetime)
	if err != nil {
		return fmt.Errorf("invalid config %s.%s.poolMaxConnLifetime: %s", POSTGRESQL, section, err.Error())
	}
	maxConnLifetimeJitterValue, err := parsePostgresDuration(maxConnLifetimeJitter)
	if err != nil {
		return fmt.Errorf("invalid config %s.%s.poolMaxConnLifetimeJitter: %s", POSTGRESQL, section, err.Error())
	}
	maxConnIdleTimeValue, err := parsePostgresDuration(maxConnIdleTime)
	if err != nil {
		return fmt.Errorf("invalid config %s.%s.poolMaxConnIdleTime: %s", POSTGRESQL, section, err.Error())
	}
	healthCheckPeriodValue, err := parsePostgresDuration(healthCheckPeriod)
	if err != nil {
		return fmt.Errorf("invalid config %s.%s.poolHealthCheckPeriod: %s", POSTGRESQL, section, err.Error())
	}
	if maxConnsValue > 0 {
		if minConnsValue > maxConnsValue {
			return fmt.Errorf("invalid config %s.%s: poolMinConns must not exceed poolMaxConns", POSTGRESQL, section)
		}
		if minIdleConnsValue > maxConnsValue {
			return fmt.Errorf("invalid config %s.%s: poolMinIdleConns must not exceed poolMaxConns", POSTGRESQL, section)
		}
	}
	if maxConnsValue > 0 {
		config.MaxConns = maxConnsValue
	}
	if minConnsValue > 0 {
		config.MinConns = minConnsValue
	}
	if minIdleConnsValue > 0 {
		config.MinIdleConns = minIdleConnsValue
	}
	if maxConnLifetimeValue > 0 {
		config.MaxConnLifetime = maxConnLifetimeValue
	}
	if maxConnLifetimeJitterValue > 0 {
		config.MaxConnLifetimeJitter = maxConnLifetimeJitterValue
	}
	if maxConnIdleTimeValue > 0 {
		config.MaxConnIdleTime = maxConnIdleTimeValue
	}
	if healthCheckPeriodValue > 0 {
		config.HealthCheckPeriod = healthCheckPeriodValue
	}
	return nil
}

func parsePostgresInt32(value string) (int32, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, errors.New("must not be negative")
	}
	return int32(n), nil
}

func parsePostgresDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" || value == "0s" {
		return 0, nil
	}
	return time.ParseDuration(value)
}

func NewPostgresError(err error, details ...any) *ae.Error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	caller := utils.Caller(1)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ae.ErrorNotFound // can't WithDetail, locked
	case errors.Is(err, pgx.ErrTxClosed), errors.Is(err, pgx.ErrTxCommitRollback):
		return ae.NewError(caller + postgresTxDoneMsg + msg).WithDetail(details...)
	case errors.Is(err, pgconn.ErrConnClosed):
		return ae.NewError(caller + postgresConnDoneMsg + msg).WithDetail(details...)
	case errors.Is(err, pgx.ErrTooManyRows):
		return ae.NewError(caller + postgresTooManyRowsMsg + msg).WithDetail(details...)
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return ae.NewError(caller + postgresTimeoutMsg + msg).WithDetail(details...)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgErrCodeUniqueViolation {
		return ae.NewConflict("postgres key").WithDetail(details...)
	}

	return ae.NewError(caller + postgresErrorMsg + msg).WithDetail(details...)
}
