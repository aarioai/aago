package driver

import (
	"crypto/tls"
	"net/url"
	"sync"
	"time"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/alog"
	"github.com/aarioai/aago/pkg/types"
	"github.com/rabbitmq/amqp091-go"
	"github.com/wagslane/go-rabbitmq"
)

const (
	RABBITMQ = "rabbitmq"
)

type RabbitmqConfig struct {
	ConnectionOptions rabbitmq.Config
	ReconnectInterval time.Duration

	Host     string
	User     string
	Password string
}

var (
	rabbitmqClients sync.Map
)

// NewRabbitmq
// Note: better use NewRabbitmqPool instead
func NewRabbitmq(app *aa.App, section string, tlsConfig *tls.Config, sasl []amqp091.Authentication, opts []func(*rabbitmq.ConnectionOptions)) (*rabbitmq.Conn, *ae.Error) {
	c, err := ParseRabbitmqConfig(app, section, tlsConfig, sasl)
	if err != nil {
		return nil, newConfigError(RABBITMQ, section, err)
	}

	defaultOpts := []func(*rabbitmq.ConnectionOptions){
		rabbitmq.WithConnectionOptionsLogger(NewRabbitMQLogger(app.Log)),
		rabbitmq.WithConnectionOptionsConfig(c.ConnectionOptions),
	}
	if c.ReconnectInterval > 0 {
		defaultOpts = append(defaultOpts, rabbitmq.WithConnectionOptionsBaseReconnectInterval(c.ReconnectInterval))
	}

	if len(opts) > 0 {
		defaultOpts = append(defaultOpts, opts...)
	}
	conn, err := rabbitmq.NewConn(c.Url(), defaultOpts...)
	if err != nil {
		return nil, NewRabbitmqError(err)
	}
	return conn, nil
}

// NewRabbitmqPool
// Warning: Do not unset the returned client as it is managed by the pool
// Warning: 使用完不要unset client，释放是错误人为操作，可能会导致其他正在使用该client的线程panic，这里不做过度处理。
func NewRabbitmqPool(app *aa.App, section string, tlsConfig *tls.Config, sasl []amqp091.Authentication, opts []func(*rabbitmq.ConnectionOptions)) (*rabbitmq.Conn, *ae.Error) {
	client, ok := rabbitmqClients.Load(section)
	if ok {
		if client != nil {
			return client.(*rabbitmq.Conn), nil
		}
		rabbitmqClients.Delete(section)
	}
	created, e := NewRabbitmq(app, section, tlsConfig, sasl, opts)
	if e != nil {
		return nil, e
	}
	actual, loaded := rabbitmqClients.LoadOrStore(section, created)
	if loaded {
		_ = created.Close()
		return actual.(*rabbitmq.Conn), nil
	}
	return created, nil
}

// CloseRabbitmqPool
// Each process should utilize a single connection, which is managed by the main function.
// This connection should be closed when the main function terminates.
func CloseRabbitmqPool() {
	rabbitmqClients.Range(func(k, v interface{}) bool {
		rabbitmqClients.Delete(k)
		client := v.(*rabbitmq.Conn)
		if client != nil {
			alog.Stopf("rabbitmq client: %s", k)
			_ = client.Close()
		}
		return true
	})
}

func ParseRabbitmqConfig(app *aa.App, section string, tlsConfig *tls.Config, sasl []amqp091.Authentication) (RabbitmqConfig, error) {
	host, err1 := tryGetSectionCfgWithFallback(app, RABBITMQ, section, "host", "Host")
	user, err2 := tryGetSectionCfgWithFallback(app, RABBITMQ, section, "user", "User", "username", "Username")
	password, err3 := tryGetSectionCfgWithFallback(app, RABBITMQ, section, "passwd", "Passwd", "password", "Password")
	if err := ae.FirstError(err1, err2, err3); err != nil {
		return RabbitmqConfig{}, err
	}
	vhost := getSectionCfgWithFallback(app, RABBITMQ, section, "vhost", "Vhost", "VHost")
	channelMax := getSectionCfgWithFallback(app, RABBITMQ, section, "channelMax", "ChannelMax", "channel_max")
	frameSize := getSectionCfgWithFallback(app, RABBITMQ, section, "frameSize", "FrameSize", "frame_size")
	heartbeat, err4 := getSectionCfgDurationWithFallback(app, RABBITMQ, section, "heartbeat", "Heartbeat")
	reconnectInterval, err5 := getSectionCfgDurationWithFallback(app, RABBITMQ, section, "reconnectInterval", "ReconnectInterval", "reconnect_interval")
	if err := ae.FirstError(err4, err5); err != nil {
		return RabbitmqConfig{}, err
	}

	if vhost == "" {
		vhost = "/"
	}
	amqpConfig := rabbitmq.Config{
		SASL:            sasl,
		Vhost:           vhost,
		ChannelMax:      types.ToUint16(channelMax), // 0 max channels means 2^16 - 1
		FrameSize:       types.ToInt(frameSize),     // 0 max bytes means unlimited
		Heartbeat:       heartbeat,                  // less than 1s uses the server's interval
		TLSClientConfig: tlsConfig,
		Properties:      nil,
		Locale:          app.Config.TimezoneID,
	}
	c := RabbitmqConfig{
		ConnectionOptions: amqpConfig,
		ReconnectInterval: reconnectInterval,

		Host:     host,
		User:     user,
		Password: password,
	}
	return c, nil
}
func (c RabbitmqConfig) Url() string {
	u := url.URL{
		Scheme: "amqp",
		Host:   c.Host,
		Path:   c.ConnectionOptions.Vhost,
	}
	if c.User != "" || c.Password != "" {
		u.User = url.UserPassword(c.User, c.Password)
	}
	return u.String()
}
func NewRabbitmqError(err error, details ...any) *ae.Error {
	if err == nil {
		return nil
	}
	return ae.NewErr(err, details...)
}
