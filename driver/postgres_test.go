package driver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/aa/aconfig"
	"github.com/aarioai/aago/aa/acontext"
	"github.com/aarioai/aago/driver"
)

func TestParsePostgresConfig(t *testing.T) {
	app := testApp(t)
	for _, section := range []string{"test", "postgres_test"} {
		f, err := driver.ParsePostgresConfig(app, section)
		if err != nil {
			t.Fatal(err)
		}
		c := f.Config.ConnConfig
		if c.Host != "127.0.0.1" || c.Port != 5432 || c.User != "Aario" ||
			c.Password != "Luexu.com" || c.Database != "test" {
			t.Errorf("ParsePostgresConfig(%q) connection = %#v", section, c)
		}
		if c.TLSConfig == nil {
			t.Errorf("ParsePostgresConfig(%q) expected sslmode=prefer TLS config", section)
		}
	}

	dsnConfig, err := driver.ParsePostgresConfig(app, "hello")
	if err != nil {
		t.Fatal(err)
	}
	c := dsnConfig.Config.ConnConfig
	if c.Host != "127.0.0.1" || c.User != "user" || c.Database != "hello" {
		t.Errorf("DSN config = %#v", c)
	}
}

func TestParsePostgresConfigRejectsInvalidPoolOption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid_postgres.yaml")
	content := `postgresql:
  DEFAULT:
    user: u
    password: p
    database: d
    addr: localhost:5432
  bad_value:
    poolMaxConns: not-a-number
  bad_range:
    poolMaxConns: 2
    poolMinConns: 3
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, path)
	for _, section := range []string{"bad_value", "bad_range"} {
		if _, err := driver.ParsePostgresConfig(app, section); err == nil {
			t.Errorf("ParsePostgresConfig(%q) expected an error", section)
		}
	}
}

func TestParsePostgresPoolOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "postgres_pool.yaml")
	content := `postgresql:
  DEFAULT:
    user: u
    database: d
    addr: localhost:5432
    poolMaxConns: 12
    poolMinConns: 2
    poolMinIdleConns: 1
    poolMaxConnLifetime: 1h
    poolMaxConnLifetimeJitter: 2m
    poolMaxConnIdleTime: 30m
    poolHealthCheckPeriod: 10m
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, path)
	f, err := driver.ParsePostgresConfig(app, "test")
	if err != nil {
		t.Fatal(err)
	}

	c := f.Config
	if c.MaxConns != 12 || c.MinConns != 2 || c.MinIdleConns != 1 ||
		c.MaxConnLifetime != time.Hour || c.MaxConnLifetimeJitter != 2*time.Minute ||
		c.MaxConnIdleTime != 30*time.Minute || c.HealthCheckPeriod != 10*time.Minute {
		t.Errorf("PostgreSQL pool config = %#v", c)
	}
}

func TestPostgresDSNOptions(t *testing.T) {
	app := testApp(t)
	f, err := driver.ParsePostgresConfig(app, "test")
	if err != nil {
		t.Fatal(err)
	}
	dsn := f.Config.ConnString()
	if !strings.Contains(dsn, "connect_timeout=5") || !strings.Contains(dsn, "sslmode=prefer") {
		t.Errorf("PostgreSQL DSN options not encoded: %q", dsn)
	}
	c := f.Config.ConnConfig
	if c.ConnectTimeout != 5*time.Second {
		t.Errorf("connect_timeout = %v, want 5s", c.ConnectTimeout)
	}
}

func TestPostgresNetAddr(t *testing.T) {
	app := testApp(t)
	for _, tc := range []struct {
		section string
		host    string
		port    uint16
	}{
		{"test", "127.0.0.1", 5432},           // addr host:port
		{"unix", "/var/run/postgresql", 5432}, // socket directory derived from addr
	} {
		f, err := driver.ParsePostgresConfig(app, tc.section)
		if err != nil {
			t.Fatalf("ParsePostgresConfig(%q): %s", tc.section, err)
		}
		c := f.Config.ConnConfig
		if c.Host != tc.host || c.Port != tc.port {
			t.Errorf("ParsePostgresConfig(%q) addr = %s:%d, want %s:%d", tc.section, c.Host, c.Port, tc.host, tc.port)
		}
	}

	f, err := driver.ParsePostgresConfig(app, "unix")
	if err != nil {
		t.Fatal(err)
	}
	dsn := f.Config.ConnString()
	if !strings.Contains(dsn, "host=%2Fvar%2Frun%2Fpostgresql") || !strings.Contains(dsn, "port=5432") {
		t.Errorf("unix socket DSN = %q", dsn)
	}
}

func TestPostgresAddrIgnoresHostAndPort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "postgres_addr.yaml")
	content := `postgresql:
  DEFAULT:
    user: u
    database: d
    net: tcp
  addr_only:
    addr: db.example:6432
    host: ignored.example
    port: 7000
  legacy_only:
    host: ignored.example
    port: 6543
  missing_port:
    addr: db.example
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, path)

	for _, tc := range []struct {
		section string
		host    string
		port    uint16
	}{
		{"addr_only", "db.example", 6432},
	} {
		f, err := driver.ParsePostgresConfig(app, tc.section)
		if err != nil {
			t.Fatalf("ParsePostgresConfig(%q): %s", tc.section, err)
		}
		c := f.Config.ConnConfig
		if c.Host != tc.host || c.Port != tc.port {
			t.Errorf("ParsePostgresConfig(%q) address = %s:%d, want %s:%d", tc.section, c.Host, c.Port, tc.host, tc.port)
		}
	}
	for _, section := range []string{"legacy_only", "missing_port"} {
		if _, err := driver.ParsePostgresConfig(app, section); err == nil {
			t.Errorf("ParsePostgresConfig(%q) expected an address validation error", section)
		}
	}
}

func TestPostgresDSNEncodesCredentials(t *testing.T) {
	app := testApp(t)
	f, e := driver.ParsePostgresConfig(app, "test")
	if e != nil {
		t.Fatal(e)
	}
	dsn := f.Config.ConnString()
	want := "postgres://Aario:Luexu.com@127.0.0.1:5432/test?connect_timeout=5&sslmode=prefer&timezone=Local"
	if dsn != want {
		t.Errorf("PostgreSQL DSN = %q, want %q", dsn, want)
	}
}

func testApp(t *testing.T) *aa.App {
	t.Helper()
	return newTestApp(t, "./postgres_demo.yaml")
}

func newTestApp(t *testing.T, path string) *aa.App {
	t.Helper()
	config, err := aconfig.New(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := acontext.WithCancel(acontext.Background())
	t.Cleanup(cancel)
	return aa.New(ctx, cancel, config)
}
