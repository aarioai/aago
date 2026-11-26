package driver_test

import (
	"strings"
	"testing"
	"time"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/driver"
)

func TestParseYAML(t *testing.T) {
	app := newTestApp(t, "./mysql_demo.yaml")

	testMySQLConfig(t, app, "mysql_test", "test", "127.0.0.1:3306", "Aario", "Luexu.com")
	testMySQLConfig(t, app, "test", "test", "127.0.0.1:3306", "Aario", "Luexu.com")
	testMySQLConfig(t, app, "user", "user_1314", "127.0.0.1:3306", "Aario", "Luexu.com")

	dsnConfig, err := driver.ParseMysqlConfig(app, "hello")
	if err != nil {
		t.Fatal(err)
	}
	dsn := dsnConfig.Config.FormatDSN()
	if !strings.Contains(dsn, "tcp(mysql.example:3307)/hello") {
		t.Errorf("mysql dsn %q not match", dsn)
	}
	c := dsnConfig.Config
	if c.User != "user" || c.Passwd != "password" || c.Addr != "mysql.example:3307" || c.DBName != "hello" {
		t.Errorf("mysql DSN connection fields not parsed: %#v", c)
	}
	if !c.ParseTime || !c.ClientFoundRows || c.TLSConfig != "skip-verify" ||
		c.Timeout != 2*time.Second || c.ReadTimeout != 3*time.Second ||
		c.WriteTimeout != 4*time.Second || c.Params["customOption"] != "value" {
		t.Errorf("mysql DSN options not parsed: %#v", c)
	}

	generatedConfig, err := driver.ParseMysqlConfig(app, "test")
	if err != nil {
		t.Fatal(err)
	}
	generatedDSN := generatedConfig.Config.FormatDSN()
	if !strings.Contains(generatedDSN, "tcp(127.0.0.1:3306)/test") {
		t.Errorf("mysql DSN not generated from config: %q", generatedDSN)
	}
	if g := generatedConfig.Config; g.User != "Aario" || g.Passwd != "Luexu.com" ||
		g.Addr != "127.0.0.1:3306" || g.DBName != "test" || g.WriteTimeout != 5*time.Second {
		t.Errorf("mysql options not retained: %#v", g)
	}
}

func testMySQLConfig(t *testing.T, app *aa.App, section, database, addr, user, password string) {
	t.Helper()
	mysqlConfig, err := driver.ParseMysqlConfig(app, section)
	if err != nil {
		t.Fatal(err.Error())
	}
	c := mysqlConfig.Config
	if c.Addr != addr {
		t.Errorf("test mysql addr %s not match %s", c.Addr, addr)
	}
	if c.User != user {
		t.Errorf("test mysql user %s not match %s", c.User, user)
	}
	if c.Passwd != password {
		t.Errorf("test mysql password %s not match %s", c.Passwd, password)
	}
	if c.DBName != database {
		t.Errorf("test mysql database %s not match %s", c.DBName, database)
	}
	if c.WriteTimeout != 5*time.Second {
		t.Errorf("test mysql write timeout %d not match %d", c.WriteTimeout, 5*time.Second)
	}
}
