package driver_test

import (
	"testing"
	"time"

	"github.com/aarioai/aago/driver"
)

func TestParseInfluxdbConfigFallback(t *testing.T) {
	config, err := driver.ParseInfluxdbConfig(newTestApp(t, "./influxdb_demo.yaml"), "service_metrics")
	if err != nil {
		t.Fatal(err)
	}
	if config.URL != "http://127.0.0.1:8086" || config.Org != "my-org" ||
		config.Bucket != "service-metrics" || config.BatchSize != 5000 ||
		config.FlushInterval != time.Second || config.Precision != time.Millisecond ||
		!config.Gzip || config.RetryInterval != 5*time.Second {
		t.Errorf("InfluxDB config = %#v", config)
	}
}

func TestParseMongodbConfigFallback(t *testing.T) {
	config, err := driver.ParseMongodbConfig(newTestApp(t, "./mongodb_demo.yaml"), "stats")
	if err != nil {
		t.Fatal(err)
	}
	if config.Hosts != "127.0.0.1:27017" || config.DB != "statistics" ||
		config.Credential == nil || config.Credential.Username != "replace-with-your-username" ||
		config.Credential.AuthSource != "admin" {
		t.Errorf("MongoDB config = %#v", config)
	}
	if config.ConnectionOptions == nil || config.ConnectionOptions.ConnectTimeout != 10*time.Second ||
		config.ConnectionOptions.Direct || config.ConnectionOptions.MaxPoolSize != 100 ||
		!config.ConnectionOptions.WriteConcernJournal {
		t.Errorf("MongoDB connection options = %#v", config.ConnectionOptions)
	}
}

func TestParseRabbitmqConfigFallback(t *testing.T) {
	config, err := driver.ParseRabbitmqConfig(newTestApp(t, "./rabbitmq_demo.yaml"), "events", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if config.Host != "127.0.0.1:5672" || config.User != "replace-with-your-username" ||
		config.ConnectionOptions.Vhost != "/events" ||
		config.ConnectionOptions.ChannelMax != 2047 ||
		config.ConnectionOptions.Heartbeat != 15*time.Second ||
		config.ReconnectInterval != 5*time.Second {
		t.Errorf("RabbitMQ config = %#v", config)
	}
}

func TestParseRedisConfigFallback(t *testing.T) {
	config, err := driver.ParseRedisConfig(newTestApp(t, "./redis_demo.yaml"), "cache")
	if err != nil {
		t.Fatal(err)
	}
	if config.Addr != "127.0.0.1:6379" || config.DB != 2 || config.MaxRetries != 3 ||
		config.PoolTimeout != 0 || config.PoolFIFO || config.DisableIdentity ||
		config.ConnMaxIdleTime != 30*time.Minute || config.FailingTimeoutSeconds != 15 {
		t.Errorf("Redis config = %#v", config)
	}
}
