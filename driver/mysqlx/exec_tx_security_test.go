package mysqlx_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"

	"github.com/aarioai/aago/driver/mysqlx"
)

var (
	registerTxDriver sync.Once
	rollbackCalls    int
)

type rollbackTestDriver struct{}

func (rollbackTestDriver) Open(string) (driver.Conn, error) {
	return rollbackTestConn{}, nil
}

type rollbackTestConn struct{}

func (rollbackTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not supported")
}

func (rollbackTestConn) Close() error {
	return nil
}

func (rollbackTestConn) Begin() (driver.Tx, error) {
	return rollbackTestTransaction{}, nil
}

type rollbackTestTransaction struct{}

func (rollbackTestTransaction) Commit() error {
	return nil
}

func (rollbackTestTransaction) Rollback() error {
	rollbackCalls++
	return nil
}

func TestRecoverRollsBackAndRepanics(t *testing.T) {
	registerTxDriver.Do(func() {
		sql.Register("mysqlx_recover_test", rollbackTestDriver{})
	})
	rollbackCalls = 0
	db, err := sql.Open("mysqlx_recover_test", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tx, e := mysqlx.NewDriver("", db, nil).Begin(context.Background(), nil)
	if e != nil {
		t.Fatalf("Begin error: %v", e)
	}
	wantPanic := "transaction panic"
	func() {
		defer func() {
			if got := recover(); got != wantPanic {
				t.Errorf("recovered panic = %#v, want %#v", got, wantPanic)
			}
		}()
		defer tx.Recover()()
		panic(wantPanic)
	}()
	if rollbackCalls != 1 {
		t.Errorf("rollback calls = %d, want 1", rollbackCalls)
	}
}
