package postgresx_test

import (
	"reflect"
	"testing"

	"github.com/aarioai/aago/driver/asql"
	"github.com/aarioai/aago/driver/postgresx"
)

func TestCondUsesPostgresPlaceholders(t *testing.T) {
	cond := &postgresx.Cond{}
	cond.And("t.id", "100").Or("t.name", ":@Ada@").OrderBy("t.id DESC, t.name").Limit(5, 20)
	query, args := cond.StmtArgs()
	want := ` WHERE "t"."id"=$1 OR "t"."name" LIKE $2 ESCAPE '!' ORDER BY "t"."id" DESC, "t"."name" LIMIT 20 OFFSET 5`
	if query != want {
		t.Errorf("StmtArgs query = %q, want %q", query, want)
	}
	if !reflect.DeepEqual(args, []any{"100", "%Ada%"}) {
		t.Errorf("StmtArgs args = %#v", args)
	}
}

func TestCondLiteralHandlesNumberedPlaceholders(t *testing.T) {
	cond := &postgresx.Cond{}
	cond.And("first", "one").And("second", "two")
	want := ` WHERE "first"=E'one' AND "second"=E'two' LIMIT 10 OFFSET 0`
	if got := cond.Stmt(); got != want {
		t.Errorf("Stmt() = %q, want %q", got, want)
	}
}

func TestFilterExprUsesPostgresDialect(t *testing.T) {
	expr, e := asql.NewFilterExpr(":Aario,Tom")
	if e != nil {
		t.Fatal(e)
	}
	query, args, e := postgresx.FilterExpr("t.name", expr)
	if e != nil {
		t.Fatal(e)
	}
	if query != `"t"."name" IN ($1,$2)` || !reflect.DeepEqual(args, []any{"Aario", "Tom"}) {
		t.Errorf("FilterExpr = %q %#v", query, args)
	}
}

func TestCondFailsClosedOnInvalidOrderBy(t *testing.T) {
	cond := &postgresx.Cond{}
	cond.And("id", "1").OrderBy("id DESC; DROP TABLE users")
	if cond.Err() == nil {
		t.Fatal("expected invalid order error")
	}
	query, args := cond.StmtArgs()
	if query != " WHERE 1!=1 LIMIT 10 OFFSET 0" || len(args) != 0 {
		t.Errorf("StmtArgs = %q %#v", query, args)
	}
}

func TestCondLiteralFailsClosedOnNullByte(t *testing.T) {
	cond := &postgresx.Cond{}
	cond.And("name", "a\x00b")
	if cond.Err() == nil {
		t.Fatal("expected invalid literal error")
	}
	query, args := cond.StmtArgs()
	if query != " WHERE 1!=1 LIMIT 10 OFFSET 0" || len(args) != 0 {
		t.Errorf("StmtArgs = %q %#v", query, args)
	}
}
