package driver

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/alog"
	"github.com/aarioai/aago/pkg/types"
	"github.com/aarioai/aago/pkg/utils"
	"github.com/go-sql-driver/mysql"
)

const (
	MYSQL           = "mysql"
	sqlBadConnMsg   = "sql bad conn: "
	sqlSkipMsg      = "sql skip: "
	sqlRemoveArgMsg = "sql remove argument: "
	sqlConnDoneMsg  = "sql conn done: "
	sqlTxDoneMsg    = "sql tx done: "
	sqlErrorMsg     = "sql error: "
)

var (
	duplicateKeyPattern = regexp.MustCompile(`Duplicate\s+entry\s+'([^']*)'\s+for\s+key\s+'([^']*)'`)
	mysqlPoolMu         sync.Mutex
)

type MysqlPoolConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type MysqlConfig struct {
	Config *mysql.Config
	Pool   MysqlPoolConfig
}

type MysqlClientData struct {
	Client   *sql.DB
	Database string
}

var (
	mysqlClients sync.Map
)

// NewMysql
// Note: better use NewMysqlPool instead
func NewMysql(app *aa.App, section string) (string, *sql.DB, *ae.Error) {
	f, err := ParseMysqlConfig(app, section)
	if err != nil {
		return "", nil, newConfigError(MYSQL, section, err)
	}
	app.Log.Infof(app.GlobalContext, "connect mysql: %s@%s %s", f.Config.User, f.Config.Addr, f.Config.DBName)
	dsn := f.Config.FormatDSN()
	// sqlx.Open并不会立即建立一个数据库的网络连接, 也不会对数据库链接参数的合法性做检验, 它仅仅是初始化一个sql.DB对象. 当真正进行第一次数据库查询操作时, 此时才会真正建立网络连接;
	// sqlx.Open返回的sql.DB对象是协程并发安全的.
	// sqlx.DB表示操作数据库的抽象接口的对象，但不是所谓的数据库连接对象，sqlx.DB对象只有当需要使用时才会创建连接，如果想立即验证连接，需要用Ping()方法;
	// 每次db.Query操作后, 都建议调用rows.Close(). 因为 db.Query() 会从数据库连接池中获取一个连接, 这个底层连接在结果集(rows)未关闭前会被标记为处于繁忙状态。当遍历读到最后一条记录时，会发生一个内部EOF错误，自动调用rows.Close(),但如果提前退出循环，rows不会关闭，连接不会回到连接池中，连接也不会关闭, 则此连接会一直被占用. 因此通常我们使用 defer rows.Close() 来确保数据库连接可以正确放回到连接池中; 不过阅读源码发现rows.Close()操作是幂等操作，即一个幂等操作的特点是其任意多次执行所产生的影响均与一次执行的影响相同, 所以即便对已关闭的rows再执行close()也没关系.
	// 需要import 	_ "github.com/go-sqlx-driver/mysql"
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", conn, NewMysqlError(err, fmt.Sprintf("open mysql: %s", dsn))
	}

	// It is rare to Close a db, as the db handle is meant to be long-lived and shared between many goroutines.
	conn.SetMaxIdleConns(f.Pool.MaxIdleConns) // 设置闲置的连接数
	conn.SetMaxOpenConns(f.Pool.MaxOpenConns) // 设置最大打开的连接数，默认值为0表示不限制
	conn.SetConnMaxLifetime(f.Pool.ConnMaxLifetime)
	conn.SetConnMaxIdleTime(f.Pool.ConnMaxIdleTime)
	return f.Config.DBName, conn, nil
}

// NewMysqlPool
// Warning: Do not unset the returned client as it is managed by the pool
// Warning: 使用完不要unset client，释放是错误人为操作，可能会导致其他正在使用该client的线程panic，这里不做过度处理。
func NewMysqlPool(app *aa.App, section string) (string, *sql.DB, *ae.Error) {
	key := normalizePoolSection(MYSQL, section)
	mysqlPoolMu.Lock()
	defer mysqlPoolMu.Unlock()
	if d, ok := mysqlClients.Load(key); ok {
		clientData := d.(MysqlClientData)
		if clientData.Client != nil {
			return clientData.Database, clientData.Client, nil
		}
		mysqlClients.Delete(key)
	}
	schema, db, e := NewMysql(app, section)
	if e != nil {
		return "", nil, e
	}
	mysqlClients.Store(key, MysqlClientData{
		Database: schema,
		Client:   db,
	})
	return schema, db, nil
}

// CloseMysqlPool
// Each process should utilize a single connection, which is managed by the main function.
// This connection should be closed when the main function terminates.
func CloseMysqlPool() {
	mysqlPoolMu.Lock()
	defer mysqlPoolMu.Unlock()
	mysqlClients.Range(func(k, v any) bool {
		mysqlClients.Delete(k)
		clientData := v.(MysqlClientData)
		if client := clientData.Client; client != nil {
			alog.Stopf("mysql client: %s", k)
			alog.OnError(client.Close())
		}
		return true
	})
}

// ParseMysqlConfig see mysql_config_demo.yaml
func ParseMysqlConfig(app *aa.App, section string) (MysqlConfig, error) {
	pool, err := parseMysqlPoolOptions(app, section)
	if err != nil {
		return MysqlConfig{}, err
	}

	f := MysqlConfig{
		Config: nil,
		Pool:   pool,
	}
	dsn := getSectionCfgWithFallback(app, MYSQL, section, "dsn", "Dsn", "DSN")
	if dsn != "" {
		if f.Config, err = mysql.ParseDSN(dsn); err != nil {
			return f, err
		}
		f.Config.Loc = app.Config.TimeLocation
		return f, nil
	}
	c := mysql.NewConfig()
	c.Loc = app.Config.TimeLocation
	c.User = getSectionCfgWithFallback(app, MYSQL, section, "user", "User", "username", "Username")
	if c.User == "" {
		return f, fmt.Errorf("required config not found: %s.%s.%s", MYSQL, section, "user")
	}
	c.Passwd = getSectionCfgWithFallback(app, MYSQL, section, "passwd", "Passwd", "password", "Password") // enable empty
	c.Net = getSectionCfgWithFallback(app, MYSQL, section, "net", "Net", "network", "Network")
	c.Addr = getSectionCfgWithFallback(app, MYSQL, section, "addr", "Addr")
	c.DBName = getSectionCfgWithFallback(app, MYSQL, section, "db", "Db", "DB", "database", "Database")
	if c.DBName == "" {
		c.DBName = normalizePoolSection(MYSQL, section)
	}
	c.ConnectionAttributes = getSectionCfgWithFallback(app, MYSQL, section, "connAttributes", "ConnAttributes", "conn_attributes")
	c.Collation = getSectionCfgWithFallback(app, MYSQL, section, "collation")
	maxAllowedPacket := getSectionCfgWithFallback(app, MYSQL, section, "maxAllowedPacket", "MaxAllowedPacket", "max_allowed_packet")
	if maxAllowedPacket != "" {
		if c.MaxAllowedPacket, err = strconv.Atoi(maxAllowedPacket); err != nil {
			return f, errors.New("invalid value for maxAllowedPacket: " + err.Error())
		}
	}

	c.ServerPubKey = getSectionCfgWithFallback(app, MYSQL, section, "serverPubKey", "ServerPubKey", "server_pub_key")
	c.TLSConfig = getSectionCfgWithFallback(app, MYSQL, section, "tls", "Tls", "TLS")
	timeout := getSectionCfgWithFallback(app, MYSQL, section, "timeout", "Timeout")
	c.Timeout, c.ReadTimeout, c.WriteTimeout = parseTimeouts(timeout)
	c.AllowAllFiles, err = getSectionCfgBoolWithFallback(app, MYSQL, section, "allowAllFiles", "AllowAllFiles", "allow_all_files")
	if err != nil {
		return f, err
	}
	c.AllowCleartextPasswords, err = getSectionCfgBoolWithFallback(app, MYSQL, section, "allowCleartextPasswords", "AllowCleartextPasswords", "allow_cleartext_passwords")
	if err != nil {
		return f, err
	}
	c.AllowFallbackToPlaintext, err = getSectionCfgBoolWithFallback(app, MYSQL, section, "allowFallbackToPlaintext", "AllowFallbackToPlaintext", "allow_fallback_to_plaintext")
	if err != nil {
		return f, err
	}
	c.AllowNativePasswords, err = getSectionCfgBoolWithFallback(app, MYSQL, section, "allowNativePasswords", "AllowNativePasswords", "allow_native_passwords")
	if err != nil {
		return f, err
	}
	c.AllowOldPasswords, err = getSectionCfgBoolWithFallback(app, MYSQL, section, "allowOldPasswords", "AllowOldPasswords", "allow_old_passwords")
	if err != nil {
		return f, err
	}
	c.CheckConnLiveness, err = getSectionCfgBoolWithFallback(app, MYSQL, section, "checkConnLiveness", "CheckConnLiveness", "check_conn_liveness")
	if err != nil {
		return f, err
	}
	c.ClientFoundRows, err = getSectionCfgBoolWithFallback(app, MYSQL, section, "clientFoundRows", "ClientFoundRows", "client_found_rows")
	if err != nil {
		return f, err
	}
	c.ColumnsWithAlias, err = getSectionCfgBoolWithFallback(app, MYSQL, section, "columnsWithAlias", "ColumnsWithAlias", "columns_with_alias")
	if err != nil {
		return f, err
	}
	c.InterpolateParams, err = getSectionCfgBoolWithFallback(app, MYSQL, section, "interpolateParams", "InterpolateParams", "interpolate_params")
	if err != nil {
		return f, err
	}
	c.MultiStatements, err = getSectionCfgBoolWithFallback(app, MYSQL, section, "multiStatements", "MultiStatements", "multi_statements")
	if err != nil {
		return f, err
	}
	c.ParseTime, err = getSectionCfgBoolWithFallback(app, MYSQL, section, "parseTime", "ParseTime", "parse_time")
	if err != nil {
		return f, err
	}
	f.Config = c
	return f, nil
}

func parseMysqlPoolOptions(app *aa.App, section string) (c MysqlPoolConfig, err error) {
	maxIdleConns := getSectionCfgWithFallback(app, MYSQL, section, "poolMaxIdleConns", "PoolMaxIdleConns", "pool_max_idle_conns")
	maxOpenConns := getSectionCfgWithFallback(app, MYSQL, section, "poolMaxOpenConns", "PoolMaxOpenConns", "pool_max_open_conns")
	connMaxLifeTime := getSectionCfgWithFallback(app, MYSQL, section, "poolConnMaxLifeTime", "PoolConnMaxLifeTime", "pool_conn_max_life_time")
	connMaxIdleTime := getSectionCfgWithFallback(app, MYSQL, section, "poolConnMaxIdleTime", "PoolConnMaxIdleTime", "pool_conn_max_idle_time")

	if maxIdleConns != "" {
		if c.MaxIdleConns, err = strconv.Atoi(maxIdleConns); err != nil {
			return
		}
	}
	if maxOpenConns != "" {
		if c.MaxOpenConns, err = strconv.Atoi(maxOpenConns); err != nil {
			return
		}
	}
	if connMaxLifeTime != "" {
		if c.ConnMaxLifetime, err = types.ParseDuration(connMaxLifeTime); err != nil {
			return
		}
	}
	if connMaxIdleTime != "" {
		if c.ConnMaxIdleTime, err = types.ParseDuration(connMaxIdleTime); err != nil {
			return
		}
	}
	return
}

func NewMysqlError(err error, details ...any) *ae.Error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	caller := utils.Caller(1)

	switch {
	case errors.Is(err, driver.ErrBadConn):
		return ae.NewError(caller + sqlBadConnMsg + msg).WithDetail(details...)
	case errors.Is(err, driver.ErrSkip):
		return ae.NewError(caller + sqlSkipMsg + msg).WithDetail(details...)
	case errors.Is(err, driver.ErrRemoveArgument):
		return ae.NewError(caller + sqlRemoveArgMsg + msg).WithDetail(details...)
	case errors.Is(err, sql.ErrNoRows):
		return ae.ErrorNotFound // can't WithDetail, locked
	case errors.Is(err, sql.ErrConnDone):
		return ae.NewError(caller + sqlConnDoneMsg + msg).WithDetail(details...)
	case errors.Is(err, sql.ErrTxDone):
		return ae.NewError(caller + sqlTxDoneMsg + msg).WithDetail(details...)
	}

	// 处理重复键错误
	if matches := duplicateKeyPattern.FindStringSubmatch(msg); len(matches) == 3 {
		return ae.NewConflict("sql key").WithDetail(details...)
	}

	return ae.NewError(caller + sqlErrorMsg + msg).WithDetail(details...)
}
