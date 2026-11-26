package driver_test

import (
	"testing"
	"time"

	"github.com/aarioai/aago/driver"
)

func TestParseHttpClientConfigFallback(t *testing.T) {
	app := newTestApp(t, "./httpclient_demo.yaml")
	config, err := driver.ParseHttpClientConfig(app, "test")
	if err != nil {
		t.Fatal(err)
	}

	if config.DialKeepAliveIdle != 15*time.Second ||
		config.DialKeepAliveInterval != 15*time.Second ||
		config.DialKeepAliveCount != 9 ||
		config.MaxIdleConns != 0 ||
		config.MaxIdleConnsPerHost != 0 ||
		config.TLSServerName != "" {
		t.Errorf("HTTP client config = %#v", config)
	}
	if config.Timeout != 0 || config.DialTimeout != 0 || config.TLSHandshakeTimeout != 0 {
		t.Errorf("explicit 0s timeouts should disable the timeout: %#v", config)
	}
	if config.DisableCompression || config.ForceAttemptHTTP2 || config.TLSInsecureSkipVerify {
		t.Errorf("boolean config was not applied: %#v", config)
	}
	if len(config.TLSNextProtos) != 0 {
		t.Errorf("TLSNextProtos = %#v, want none", config.TLSNextProtos)
	}
}
