package mysqlx_test

import (
	"reflect"
	"testing"

	"github.com/aarioai/aago/aa/atype/aenum"
	"github.com/aarioai/aago/driver/mysqlx"
)

func TestCond(t *testing.T) {
	var cond = &mysqlx.Cond{}
	cond.And("t.id", "100")
	cond.Write("AND", aenum.StsInvalid("t.status"))
	cond.Try("t.ranking_woman DESC, t.vuid", 0, 20)

	if cond.Err() != nil {
		t.Fatalf("Cond error: %v", cond.Err())
	}
	if cond.Stmt() != " WHERE `t`.`id`='100' AND t.status<0 ORDER BY `t`.`ranking_woman` DESC, `t`.`vuid` LIMIT 0,20" {
		t.Errorf("test cond failed `%s`", cond.Stmt())
	}
}

func TestCondStmtArgs(t *testing.T) {
	var cond = &mysqlx.Cond{}
	cond.And("t.id", "100")
	cond.Or("t.name", ":@Aario@")

	sql, args := cond.StmtArgs()
	if sql != " WHERE `t`.`id`=? OR `t`.`name` LIKE ? ESCAPE '!' LIMIT 0,10" {
		t.Errorf("StmtArgs sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"100", "%Aario%"}) {
		t.Errorf("StmtArgs args = %#v", args)
	}
}

func TestCondSkipsEmptyCondition(t *testing.T) {
	var cond = &mysqlx.Cond{}
	cond.And("t.a", "1").And("t.b", "").Or("t.c", "2")
	if cond.Err() != nil {
		t.Fatalf("Cond error: %v", cond.Err())
	}
	want := " WHERE `t`.`a`='1' OR `t`.`c`='2' LIMIT 0,10"
	if got := cond.Stmt(); got != want {
		t.Errorf("Stmt() = %q, want %q", got, want)
	}
}

func TestCondFailsClosed(t *testing.T) {
	var cond = &mysqlx.Cond{}
	cond.And("t.a", "1").And("t.b", ":@@").Or("t.c", "2")
	if cond.Err() == nil {
		t.Fatal("want builder error")
	}
	want := " WHERE 1!=1 LIMIT 0,10"
	if got := cond.Stmt(); got != want {
		t.Errorf("Stmt() = %q, want %q", got, want)
	}
	sql, args := cond.StmtArgs()
	if sql != want || len(args) != 0 {
		t.Errorf("StmtArgs() = %q, %#v, want %q, nil", sql, args, want)
	}
}

func TestCondInvalidConnector(t *testing.T) {
	var cond = &mysqlx.Cond{}
	cond.Write("; DROP TABLE x", "1=1")
	if cond.Err() == nil {
		t.Fatal("want connector error")
	}
	if got := cond.Stmt(); got != " WHERE 1!=1 LIMIT 0,10" {
		t.Errorf("Stmt() = %q", got)
	}
}

func TestCondOrderByRejectsSQL(t *testing.T) {
	for _, orderBy := range []string{
		"id DESC; DROP TABLE users",
		"id DESC,",
		"id DESC LIMIT 1",
		"`id` DESC",
	} {
		cond := &mysqlx.Cond{}
		cond.OrderBy(orderBy)
		if cond.Err() == nil {
			t.Errorf("OrderBy(%q) error = nil, want error", orderBy)
			continue
		}
		if got := cond.Stmt(); got != " WHERE 1!=1 LIMIT 0,10" {
			t.Errorf("OrderBy(%q) Stmt() = %q, want fail-closed", orderBy, got)
		}
	}
}

func TestCondOrderByValidatesTry(t *testing.T) {
	cond := &mysqlx.Cond{}
	cond.Try("users.id DESC, users.name", 0, 10)
	want := " ORDER BY `users`.`id` DESC, `users`.`name` LIMIT 0,10"
	if got := cond.Stmt(); got != want {
		t.Errorf("Stmt() = %q, want %q", got, want)
	}
}
