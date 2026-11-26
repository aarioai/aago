package driver_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/driver"
	"github.com/redis/go-redis/v9"
)

func TestRedis(t *testing.T) {
	app := newTestApp(t, "./redis_demo.yaml")

	want := redis.Options{
		Network:               "tcp",
		Addr:                  "127.0.0.1:6379",
		Protocol:              3,
		Username:              "replace-with-your-username",
		Password:              "replace-with-your-password",
		DB:                    1,
		MaxRetries:            3,
		MinRetryBackoff:       10 * time.Millisecond,
		MaxRetryBackoff:       time.Second,
		DialTimeout:           5 * time.Second,
		DialerRetries:         5,
		DialerRetryTimeout:    100 * time.Millisecond,
		ReadTimeout:           5 * time.Second,
		WriteTimeout:          5 * time.Second,
		ReadBufferSize:        32768,
		WriteBufferSize:       32768,
		ConnMaxIdleTime:       30 * time.Minute,
		FailingTimeoutSeconds: 15,
	}

	for _, section := range []string{"test", "redis_test"} {
		testRedisConfig(t, app, section, want)
	}

	wantCache := want
	wantCache.DB = 2
	testRedisConfig(t, app, "cache", wantCache)
}

func testRedisConfig(t *testing.T, app *aa.App, section string, want redis.Options) {
	t.Helper()
	got, err := driver.ParseRedisConfig(app, section)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("ParseRedisConfig(%q) = %#v, want %#v", section, *got, want)
	}
}
