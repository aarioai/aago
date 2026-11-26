package driver

import (
	"strings"
	"sync"
	"time"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/alog"
	"github.com/aarioai/aago/pkg/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type InfluxdbLogLevel uint

const (
	INFLUXDB = "influxdb"

	InfluxdbLogLevelError InfluxdbLogLevel = 0
	InfluxdbLogLevelWarn  InfluxdbLogLevel = 1
	InfluxdbLogLevelInfo  InfluxdbLogLevel = 2
	InfluxdbLogLevelDebug InfluxdbLogLevel = 3
)

var influxdbLogLevels = map[string]InfluxdbLogLevel{
	"0":       InfluxdbLogLevelError,
	"error":   InfluxdbLogLevelError,
	"1":       InfluxdbLogLevelWarn,
	"warn":    InfluxdbLogLevelWarn,
	"warning": InfluxdbLogLevelWarn,
	"2":       InfluxdbLogLevelInfo,
	"info":    InfluxdbLogLevelInfo,
	"3":       InfluxdbLogLevelDebug,
	"debug":   InfluxdbLogLevelDebug,
}

func ToInfluxdbLogLevel(s string) InfluxdbLogLevel {
	if level, ok := influxdbLogLevels[strings.ToLower(s)]; ok {
		return level
	}
	return InfluxdbLogLevelError
}

type InfluxdbConfig struct {
	Org    string `json:"org"`
	Bucket string `json:"bucket"`

	URL      string           `json:"url"`
	Auth     string           `json:"auth"`
	LogLevel InfluxdbLogLevel `json:"log_level"`

	BatchSize     uint          `json:"batch_size"`     // Maximum number of points sent to server in single request.
	FlushInterval time.Duration `json:"flush_interval"` // Interval, in which is buffer flushed if it has not been already written (by reaching batch size)
	Precision     time.Duration `json:"precision"`      // Precision to use in writes for timestamp. In unit of duration: time.Nanosecond, time.Microsecond, time.Millisecond, time.Second
	Gzip          bool          `json:"gzip"`           // Whether to use GZip compression in requests
	//DefaultTags      map[string]string `json:"default_tags"`       // Tags added to each point during writing. If a point already has a tag with the same key, it is left unchanged.
	RetryInterval    time.Duration `json:"retry_interval"`     // Default retry interval, if not sent by server.
	MaxRetries       uint          `json:"max_retries"`        // Maximum count of retry attempts of failed write
	RetryBufferLimit uint          `json:"retry_buffer_limit"` // Maximum number of points to keep for retry. Should be multiple of BatchSize.
	MaxRetryInterval time.Duration `json:"max_retry_interval"` // The maximum delay between each retry attempt
	MaxRetryTime     time.Duration `json:"max_retry_time"`     // The maximum total retry timeout
	ExponentialBase  uint          `json:"exponential_base"`   // The base for the exponential retry delay
}

func (c InfluxdbConfig) Options(defaultTags map[string]string) *influxdb2.Options {
	opts := influxdb2.DefaultOptions()
	opts.SetLogLevel(uint(c.LogLevel))
	if c.BatchSize > 0 {
		opts.SetBatchSize(c.BatchSize)
	}
	if c.FlushInterval > 0 {
		opts.SetFlushInterval(uint(c.FlushInterval.Milliseconds()))
	}
	if c.Precision > 0 {
		opts.SetPrecision(c.Precision)
	}
	if c.Gzip {
		opts.SetUseGZip(c.Gzip)
	}
	if len(defaultTags) > 0 {
		for k, v := range defaultTags {
			opts.AddDefaultTag(k, v)
		}
	}
	if c.RetryInterval > 0 {
		opts.SetRetryInterval(uint(c.RetryInterval.Milliseconds()))
	}
	if c.MaxRetries > 0 {
		opts.SetMaxRetries(c.MaxRetries)
	}
	if c.RetryBufferLimit > 0 {
		opts.SetRetryBufferLimit(c.RetryBufferLimit)
	}
	if c.MaxRetryInterval > 0 {
		opts.SetMaxRetryInterval(uint(c.MaxRetryInterval.Milliseconds()))
	}
	if c.MaxRetryTime > 0 {
		opts.SetMaxRetryTime(uint(c.MaxRetryTime.Milliseconds()))
	}
	if c.ExponentialBase > 0 {
		opts.SetExponentialBase(c.ExponentialBase)
	}

	return opts
}

type InfluxdbClientData struct {
	Client               influxdb2.Client
	DefaultOrg           string
	DefaultBucket        string
	DefaultQuery         api.QueryAPI
	DefaultWrite         api.WriteAPI
	DefaultWriteBlocking api.WriteAPIBlocking
}

var (
	influxdbClients sync.Map
)

// NewInfluxdb
// Note: better use NewInfluxdbPool instead
func NewInfluxdb(app *aa.App, section string, defaultTags map[string]string) (influxdb2.Client, string, string, *ae.Error) {
	c, err := ParseInfluxdbConfig(app, section)
	if err != nil {
		return nil, "", "", newConfigError("httpclient", section, err)
	}
	// use config [httpc_influxdb_xxx]
	httpClient, e := NewHttpClient(app, "httpc_"+section)
	if e != nil {
		return nil, "", "", e
	}

	opts := c.Options(defaultTags)
	opts.SetHTTPClient(httpClient)
	client := influxdb2.NewClientWithOptions(c.URL, c.Auth, opts)
	return client, c.Org, c.Bucket, nil
}

// NewInfluxdbPool
// Warning: Do not unset the returned client as it is managed by the pool
// Warning: 使用完不要unset client，释放是错误人为操作，可能会导致其他正在使用该client的线程panic，这里不做过度处理。
func NewInfluxdbPool(app *aa.App, section string, defaultTags map[string]string) (InfluxdbClientData, *ae.Error) {
	d, ok := influxdbClients.Load(section)
	if ok {
		t := d.(InfluxdbClientData)
		if t.Client != nil && t.DefaultQuery != nil && t.DefaultWrite != nil && t.DefaultWriteBlocking != nil {
			return t, nil
		}
		influxdbClients.Delete(section)
	}
	client, org, bucket, e := NewInfluxdb(app, section, defaultTags)
	if e != nil {
		return InfluxdbClientData{}, e
	}
	query := client.QueryAPI(org)
	write := client.WriteAPI(org, bucket)
	writeBlocking := client.WriteAPIBlocking(org, bucket)
	stored := InfluxdbClientData{
		Client:               client,
		DefaultOrg:           org,
		DefaultBucket:        bucket,
		DefaultQuery:         query,
		DefaultWrite:         write,
		DefaultWriteBlocking: writeBlocking,
	}
	actual, loaded := influxdbClients.LoadOrStore(section, stored)
	if loaded {
		client.Close()
	}
	return actual.(InfluxdbClientData), nil
}

// CloseInfluxdbPool
// Each process should utilize a single connection, which is managed by the main function.
// This connection should be closed when the main function terminates.
func CloseInfluxdbPool() {
	influxdbClients.Range(func(k, v interface{}) bool {
		influxdbClients.Delete(k)
		clientData := v.(InfluxdbClientData)
		if client := clientData.Client; client != nil {
			alog.Stopf("influxdb client: %s", k)
			client.Close()
		}
		return true
	})
}

func ParseInfluxdbConfig(app *aa.App, section string) (InfluxdbConfig, error) {
	url, err1 := tryGetSectionCfgWithFallback(app, INFLUXDB, section, "url", "URL")
	auth, err2 := tryGetSectionCfgWithFallback(app, INFLUXDB, section, "auth", "Auth")
	org, err3 := tryGetSectionCfgWithFallback(app, INFLUXDB, section, "org", "Org")
	bucket, err4 := tryGetSectionCfgWithFallback(app, INFLUXDB, section, "bucket", "Bucket")
	if err := ae.FirstError(err1, err2, err3, err4); err != nil {
		return InfluxdbConfig{}, err
	}

	logLevel := getSectionCfgWithFallback(app, INFLUXDB, section, "logLevel", "LogLevel", "log_level")
	batchSize := getSectionCfgWithFallback(app, INFLUXDB, section, "batchSize", "BatchSize", "batch_size")
	flushInterval, err5 := getSectionCfgDurationWithFallback(app, INFLUXDB, section, "flushInterval", "FlushInterval", "flush_interval")
	precision, err6 := getSectionCfgDurationWithFallback(app, INFLUXDB, section, "precision", "Precision")
	gzip, err7 := getSectionCfgBoolWithFallback(app, INFLUXDB, section, "gzip", "Gzip")
	retryInterval, err8 := getSectionCfgDurationWithFallback(app, INFLUXDB, section, "retryInterval", "RetryInterval", "retry_interval")
	maxRetries := getSectionCfgWithFallback(app, INFLUXDB, section, "maxRetries", "MaxRetries", "max_retries")
	retryBufferLimit := getSectionCfgWithFallback(app, INFLUXDB, section, "retryBufferLimit", "RetryBufferLimit", "retry_buffer_limit")
	maxRetryInterval, err9 := getSectionCfgDurationWithFallback(app, INFLUXDB, section, "maxRetryInterval", "MaxRetryInterval", "max_retry_interval")
	maxRetryTime, err10 := getSectionCfgDurationWithFallback(app, INFLUXDB, section, "maxRetryTime", "MaxRetryTime", "max_retry_time")
	exponentBase := getSectionCfgWithFallback(app, INFLUXDB, section, "exponentBase", "ExponentBase", "exponent_base")
	if err := ae.FirstError(err5, err6, err7, err8, err9, err10); err != nil {
		return InfluxdbConfig{}, err
	}

	c := InfluxdbConfig{
		URL:    url,
		Auth:   auth,
		Org:    org,
		Bucket: bucket,

		LogLevel:         ToInfluxdbLogLevel(logLevel),
		BatchSize:        types.ToUint(batchSize),
		FlushInterval:    flushInterval,
		Precision:        precision,
		Gzip:             gzip,
		RetryInterval:    retryInterval,
		MaxRetries:       types.ToUint(maxRetries),
		RetryBufferLimit: types.ToUint(retryBufferLimit),
		MaxRetryInterval: maxRetryInterval,
		MaxRetryTime:     maxRetryTime,
		ExponentialBase:  types.ToUint(exponentBase),
	}
	return c, nil
}

func NewInfluxdbError(err error) *ae.Error {
	return ae.NewErr(err)
}
