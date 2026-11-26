package driver

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/alog"
	"github.com/aarioai/aago/pkg/types"
	"github.com/aarioai/aago/pkg/utils"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

const (
	MONGODB = "mongodb"
)

type MongodbConnectionOptions struct {
	// https://www.mongodb.com/docs/drivers/go/current/fundamentals/connections/connection-options/
	ConnectTimeout         time.Duration `json:"connect_timeout" query:"connectTimeoutMS"`
	Direct                 bool          `json:"direct" query:"directConnection"`
	HeartbeatFrequency     time.Duration `json:"heartbeat_frequency" query:"heartbeatFrequencyMS"`
	MaxIdleTime            time.Duration `json:"max_idle_time" query:"maxIdleTimeMS"`
	MaxPoolSize            uint64        `json:"max_pool_size" query:"maxPoolSize"` // default 100
	MinPoolSize            uint64        `json:"min_pool_size" query:"minPoolSize"`
	ReplicaSet             string        `json:"replica_set" query:"replicaSet"`
	ServerSelectionTimeout time.Duration `json:"server_selection_timeout" query:"serverSelectionTimeoutMS"`
	Timeout                time.Duration `json:"timeout" query:"timeoutMS"`
	TLS                    bool          `json:"tls" query:"tls"`
	TLSInsecureSkipVerify  bool          `json:"tls_insecure_skip_verify" query:"tlsInsecure"`
	WriteConcern           string        `json:"write_concern" query:"w"`
	WriteConcernJournal    bool          `json:"write_concern_journal" query:"j"`
}

func (c *MongodbConnectionOptions) LoadTo(opts *options.ClientOptions) {
	if c.ConnectTimeout > 0 {
		opts.SetConnectTimeout(c.ConnectTimeout)
	}
	if c.Direct {
		opts.SetDirect(true)
	}
	if c.HeartbeatFrequency > 0 {
		opts.SetHeartbeatInterval(c.HeartbeatFrequency)
	}

	if c.MaxIdleTime > 0 {
		opts.SetMaxConnIdleTime(c.MaxIdleTime)
	}

	if c.MaxPoolSize > 0 {
		opts.SetMaxPoolSize(c.MaxPoolSize)
	}
	if c.MinPoolSize > 0 {
		opts.SetMinPoolSize(c.MinPoolSize)
	}
	if c.ReplicaSet != "" {
		opts.SetReplicaSet(c.ReplicaSet)
	}
	if c.ServerSelectionTimeout > 0 {
		opts.SetServerSelectionTimeout(c.ServerSelectionTimeout)
	}
	if c.Timeout > 0 {
		opts.SetTimeout(c.Timeout)
	}
	if c.TLS {
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: c.TLSInsecureSkipVerify}) // #nosec G402 - explicitly controlled by configuration
	}
	if c.WriteConcern != "" {
		journal := c.WriteConcernJournal
		wc := &writeconcern.WriteConcern{W: c.WriteConcern, Journal: &journal}
		if n, err := types.ParseInt(c.WriteConcern); err == nil {
			wc.W = n
		}
		opts.SetWriteConcern(wc)
	}
}

type MongodbCredential struct {
	AuthMechanism           string            `json:"auth_mechanism"`            // 可以为空
	AuthMechanismProperties map[string]string `json:"auth_mechanism_properties"` // 可以为空
	AuthSource              string            `json:"auth_source"`
	Username                string            `json:"username"`
	Password                string            `json:"password"`
}
type MongodbOptions struct {
	Protocol          string `json:"protocol"`
	Hosts             string `json:"hosts"` // e.g localhost,192.158.1.111:27017
	DB                string `json:"db"`
	Credential        *MongodbCredential
	ConnectionOptions *MongodbConnectionOptions
}
type MongodbClientData struct {
	Client *mongo.Client
	DB     string
}

var (
	mongodbClients sync.Map
)

// NewMongodb
// Note: better use NewMongodbPool instead
func NewMongodb(app *aa.App, section string) (*mongo.Client, string, *ae.Error) {
	o, err := ParseMongodbConfig(app, section)
	if err != nil {
		return nil, "", newConfigError(MONGODB, section, err)
	}
	opts := o.ClientOptions()
	var client *mongo.Client
	if client, err = mongo.Connect(opts); err != nil {
		return nil, "", NewMongodbError(err, "connect to mongodb instance "+o.Hosts)
	}
	return client, o.DB, nil
}

// NewMongodbPool mongodb 自带连接池
// Warning: Do not unset the returned client as it is managed by the pool
// Warning: 使用完不要unset client，释放是错误人为操作，可能会导致其他正在使用该client的线程panic，这里不做过度处理。
func NewMongodbPool(app *aa.App, section string) (*mongo.Client, string, *ae.Error) {
	d, ok := mongodbClients.Load(section)
	if ok {
		clientData := d.(MongodbClientData)
		if clientData.Client != nil {
			return clientData.Client, clientData.DB, nil
		}
		mongodbClients.Delete(section)
	}
	client, db, e := NewMongodb(app, section)
	if e != nil {
		return nil, "", e
	}
	stored := MongodbClientData{
		Client: client,
		DB:     db,
	}
	actual, loaded := mongodbClients.LoadOrStore(section, stored)
	if loaded {
		_ = client.Disconnect(context.Background())
	}
	clientData := actual.(MongodbClientData)
	return clientData.Client, clientData.DB, nil
}

// CloseMongodbPool
// Each process should utilize a single connection, which is managed by the main function.
// This connection should be closed when the main function terminates.
func CloseMongodbPool(ctx context.Context) {
	mongodbClients.Range(func(k, v interface{}) bool {
		mongodbClients.Delete(k)
		clientData := v.(MongodbClientData)
		if client := clientData.Client; client != nil {
			alog.Stopf("mongodb client: %s", k)
			_ = client.Disconnect(ctx)
		}
		return true
	})
}

func (c *MongodbCredential) ToCredential() options.Credential {
	// PasswordSet: For GSSAPI, this must be true if a password is specified, even if the password is the empty string, and
	// false if no password is specified, indicating that the password should be taken from the context of the running
	// process. For other mechanisms, this field is ignored.
	passwordSet := c.AuthMechanism == "GSSAPI"
	credential := options.Credential{
		AuthMechanism:           c.AuthMechanism,
		AuthMechanismProperties: c.AuthMechanismProperties,
		AuthSource:              c.AuthSource,
		Username:                c.Username,
		Password:                c.Password,
		PasswordSet:             passwordSet,
	}
	return credential
}

func (o *MongodbOptions) ClientOptions() *options.ClientOptions {
	// Credentials must never be embedded in the URI; they are applied via SetAuth below.
	opts := options.Client().ApplyURI(o.Protocol + "://" + o.Hosts)
	if o.Credential != nil {
		opts.SetAuth(o.Credential.ToCredential())
	}
	if o.ConnectionOptions != nil {
		o.ConnectionOptions.LoadTo(opts)
	}
	return opts
}
func ParseMongodbConfig(app *aa.App, section string) (*MongodbOptions, error) {
	hosts, err := tryGetSectionCfgWithFallback(app, MONGODB, section, "hosts", "Hosts")
	if err != nil {
		return nil, err
	}
	protocol := getSectionCfgWithFallback(app, MONGODB, section, "protocol", "Protocol")
	if protocol == "" {
		protocol = MONGODB
	}
	db := getSectionCfgWithFallback(app, MONGODB, section, "db", "Db", "DB", "database", "Database")
	// credential
	var credential *MongodbCredential
	username := getSectionCfgWithFallback(app, MONGODB, section, "user", "User", "username", "Username")
	if username != "" {
		authMechanism := getSectionCfgWithFallback(app, MONGODB, section, "authMechanism", "AuthMechanism", "auth_mechanism")
		authSource := getSectionCfgWithFallback(app, MONGODB, section, "authSource", "AuthSource", "auth_source")
		password := getSectionCfgWithFallback(app, MONGODB, section, "passwd", "Passwd", "password", "Password")

		serviceName := getSectionCfgWithFallback(app, MONGODB, section, "serviceName", "ServiceName", "service_name", "SERVICE_NAME")
		canonHostName := getSectionCfgWithFallback(app, MONGODB, section, "canonicalizeHostName", "CanonicalizeHostName", "canonicalize_host_name", "CANONICALIZE_HOST_NAME")
		serviceRealm := getSectionCfgWithFallback(app, MONGODB, section, "serviceRealm", "ServiceRealm", "service_realm", "SERVICE_REALM")
		serviceHost := getSectionCfgWithFallback(app, MONGODB, section, "serviceHost", "ServiceHost", "service_host", "SERVICE_HOST")
		awsSessionToken := getSectionCfgWithFallback(app, MONGODB, section, "awsSessionToken", "AWSSessionToken", "aws_session_token", "AWS_SESSION_TOKEN")
		var properties map[string]string
		if !allEmpty(serviceName, canonHostName, serviceRealm, serviceHost, awsSessionToken) {
			properties = make(map[string]string)
			if serviceName != "" {
				properties["SERVICE_NAME"] = serviceName
			}
			if canonHostName != "" {
				properties["CANONICALIZE_HOST_NAME"] = canonHostName
			}
			if serviceRealm != "" {
				properties["SERVICE_REALM"] = serviceRealm
			}
			if serviceHost != "" {
				properties["SERVICE_HOST"] = serviceHost
			}
			if awsSessionToken != "" {
				properties["AWS_SESSION_TOKEN"] = awsSessionToken
			}
		}

		credential = &MongodbCredential{
			AuthMechanism:           authMechanism,
			AuthMechanismProperties: properties,
			AuthSource:              authSource,
			Username:                username,
			Password:                password,
		}
	}

	// connection options
	var opts *MongodbConnectionOptions
	connectTimeout, err1 := getSectionCfgDurationWithFallback(app, MONGODB, section, "connectTimeout", "ConnectTimeout", "connect_timeout")
	direct, err2 := getSectionCfgBoolWithFallback(app, MONGODB, section, "direct", "Direct")
	heartbeatFrequency, err3 := getSectionCfgDurationWithFallback(app, MONGODB, section, "heartbeatFrequency", "HeartbeatFrequency", "heartbeat_frequency")
	maxIdleTime, err4 := getSectionCfgDurationWithFallback(app, MONGODB, section, "maxIdleTime", "MaxIdleTime", "max_idle_time")
	maxPoolSize := getSectionCfgWithFallback(app, MONGODB, section, "maxPoolSize", "MaxPoolSize", "max_pool_size")
	minPoolSize := getSectionCfgWithFallback(app, MONGODB, section, "minPoolSize", "MinPoolSize", "min_pool_size")
	replicaSet := getSectionCfgWithFallback(app, MONGODB, section, "replicaSet", "ReplicaSet", "replica_set")
	serverSelectionTimeout, err5 := getSectionCfgDurationWithFallback(app, MONGODB, section, "serverSelectionTimeout", "ServerSelectionTimeout", "server_selection_timeout")
	timeout, err6 := getSectionCfgDurationWithFallback(app, MONGODB, section, "timeout", "Timeout")
	tlsEnabled, err7 := getSectionCfgBoolWithFallback(app, MONGODB, section, "tls", "TLS")
	tlsInsecureSkipVerify, err8 := getSectionCfgBoolWithFallback(app, MONGODB, section, "tlsInsecureSkipVerify", "TLSInsecureSkipVerify", "tls_insecure_skip_verify")
	writeConcern := getSectionCfgWithFallback(app, MONGODB, section, "writeConcern", "WriteConcern", "write_concern", "writerConcern", "WriterConcern", "writer_concern")
	writeConcernJournal, err9 := getSectionCfgBoolWithFallback(app, MONGODB, section, "writeConcernJournal", "WriteConcernJournal", "write_concern_journal", "writerConcernJournal", "WriterConcernJournal", "writer_concern_journal")
	if err := ae.FirstError(err1, err2, err3, err4, err5, err6, err7, err8, err9); err != nil {
		return nil, err
	}

	noOpts := allEmpty(
		getSectionCfgWithFallback(app, MONGODB, section, "connectTimeout", "ConnectTimeout", "connect_timeout"),
		getSectionCfgWithFallback(app, MONGODB, section, "direct", "Direct"),
		getSectionCfgWithFallback(app, MONGODB, section, "heartbeatFrequency", "HeartbeatFrequency", "heartbeat_frequency"),
		getSectionCfgWithFallback(app, MONGODB, section, "maxIdleTime", "MaxIdleTime", "max_idle_time"),
		maxPoolSize,
		minPoolSize,
		replicaSet,
		getSectionCfgWithFallback(app, MONGODB, section, "serverSelectionTimeout", "ServerSelectionTimeout", "server_selection_timeout"),
		getSectionCfgWithFallback(app, MONGODB, section, "timeout", "Timeout"),
		getSectionCfgWithFallback(app, MONGODB, section, "tls", "TLS"),
		getSectionCfgWithFallback(app, MONGODB, section, "tlsInsecureSkipVerify", "TLSInsecureSkipVerify", "tls_insecure_skip_verify"),
		writeConcern,
		getSectionCfgWithFallback(app, MONGODB, section, "writeConcernJournal", "WriteConcernJournal", "write_concern_journal", "writerConcernJournal", "WriterConcernJournal", "writer_concern_journal"),
	)
	if !noOpts {
		opts = &MongodbConnectionOptions{
			ConnectTimeout:         connectTimeout,
			Direct:                 direct,
			HeartbeatFrequency:     heartbeatFrequency,
			MaxIdleTime:            maxIdleTime,
			MaxPoolSize:            types.ToUint64(maxPoolSize),
			MinPoolSize:            types.ToUint64(minPoolSize),
			ReplicaSet:             replicaSet,
			ServerSelectionTimeout: serverSelectionTimeout,
			Timeout:                timeout,
			TLS:                    tlsEnabled,
			TLSInsecureSkipVerify:  tlsInsecureSkipVerify,
			WriteConcern:           writeConcern,
			WriteConcernJournal:    writeConcernJournal,
		}
	}

	return &MongodbOptions{
		Protocol:          protocol,
		Hosts:             strings.ReplaceAll(hosts, " ", ""),
		DB:                db,
		Credential:        credential,
		ConnectionOptions: opts,
	}, nil
}
func NewMongodbError(err error, details ...any) *ae.Error {
	if err == nil {
		return nil
	}

	msg := err.Error()
	caller := utils.Caller(1)

	switch {
	case errors.Is(err, mongo.ErrClientDisconnected):
		return ae.NewError(caller + sqlBadConnMsg + msg).WithDetail(details...)
	case errors.Is(err, mongo.ErrNoDocuments), errors.Is(err, mongo.ErrNilDocument), errors.Is(err, mongo.ErrFileNotFound):
		return ae.ErrorNotFound // can't WithDetail, locked
	}

	return ae.NewErr(err).WithDetail(details...)
}
