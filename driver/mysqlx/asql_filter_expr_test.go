package mysqlx_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/aarioai/aago/driver/asql"
	"github.com/aarioai/aago/driver/mysqlx"
)

func TestNewFilterExpr(t *testing.T) {
	cases := map[string]asql.FilterExpr{
		"Aario":             {EqualTo: "Aario"},
		":Aario":            {EqualTo: "Aario"},
		":Aario,Tom":        {In: []string{"Aario", "Tom"}},
		":Aario@":           {StartWith: "Aario"},
		":@Aario":           {EndWith: "Aario"},
		":@Aario@":          {Like: "Aario"},
		":2019-06-01~":      {GreaterOrEqualTo: "2019-06-01"},
		":~2019-06-01":      {LessThan: "2019-06-01"},
		":2019-06-01~06-02": {GreaterOrEqualTo: "2019-06-01", LessThan: "06-02"},
		":a~b~c":            {GreaterOrEqualTo: "a", LessThan: "b~c"},
	}
	for in, want := range cases {
		got, e := asql.NewFilterExpr(in)
		if e != nil {
			t.Errorf("NewFilterExpr(%q) error: %v", in, e)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("NewFilterExpr(%q) = %+v, want %+v", in, got, want)
		}
	}
}

func TestNewFilterExprEmpty(t *testing.T) {
	got, e := asql.NewFilterExpr("")
	if e != nil {
		t.Fatalf("NewFilterExpr(\"\") error: %v", e)
	}
	sql, args, e := mysqlx.FilterExpr("t.name", got)
	if e != nil || sql != "" || len(args) != 0 {
		t.Errorf("empty FilterExpr = %q, %#v, %v; want empty", sql, args, e)
	}
}

func TestNewFilterExprInvalid(t *testing.T) {
	inputs := []string{
		"::Aario", ":Aario:", "::Aario:", ":",
		":@", ":@@", ":~", ":a,,b", ":a,b,",
		":" + strings.Repeat("a,", 513),
		strings.Repeat("a", 4097),
	}
	for _, in := range inputs {
		if _, e := asql.NewFilterExpr(in); e == nil {
			t.Errorf("NewFilterExpr(%q) did not fail", in)
		}
		cond := &mysqlx.Cond{}
		cond.And("t.name", in)
		if cond.Err() == nil {
			t.Errorf("Cond.And(%q) error = nil, want error", in)
		} else if sql := cond.Stmt(); sql != " WHERE 1!=1 LIMIT 0,10" {
			t.Errorf("Cond.And(%q).Stmt() = %q, want fail-closed", in, sql)
		}
	}
}

func TestFilterExprRendering(t *testing.T) {
	cases := map[string]string{
		":2019-06-01~":      "`t`.`name`>='2019-06-01'",
		":~2019-06-01":      "`t`.`name`<'2019-06-01'",
		":2019-06-01~06-02": "(`t`.`name`>='2019-06-01' AND `t`.`name`<'06-02')",
		":Aario,Tom":        "`t`.`name` IN ('Aario','Tom')",
		":Aario@":           "`t`.`name` LIKE 'Aario%' ESCAPE '!'",
		":@Aario":           "`t`.`name` LIKE '%Aario' ESCAPE '!'",
		":@Aario@":          "`t`.`name` LIKE '%Aario%' ESCAPE '!'",
		"Aario":             "`t`.`name`='Aario'",
	}
	for in, want := range cases {
		cond := &mysqlx.Cond{}
		cond.And("t.name", in)
		got := strings.TrimSuffix(strings.TrimPrefix(cond.Stmt(), " WHERE "), " LIMIT 0,10")
		if got != want {
			t.Errorf("Cond.And(%q).Stmt() = %q, want %q", in, got, want)
		}
	}
}

func TestFilterExprArgs(t *testing.T) {
	expr, e := asql.NewFilterExpr(":Aario,Tom")
	if e != nil {
		t.Fatalf("NewFilterExpr error: %v", e)
	}
	sql, args, e := mysqlx.FilterExpr("t.name", expr)
	if e != nil {
		t.Fatalf("FilterExpr error: %v", e)
	}
	if sql != "`t`.`name` IN (?,?)" {
		t.Errorf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"Aario", "Tom"}) {
		t.Errorf("args = %#v", args)
	}
}

func TestCondValuesAreEscaped(t *testing.T) {
	cases := map[string]string{
		`" OR 1=1 -- `: "`t`.`name`='\" OR 1=1 -- '",
		`a\`:           "`t`.`name`='a\\\\'",
		`O'Reilly`:     "`t`.`name`='O''Reilly'",
	}
	for in, want := range cases {
		cond := &mysqlx.Cond{}
		cond.And("t.name", in)
		got := strings.TrimSuffix(strings.TrimPrefix(cond.Stmt(), " WHERE "), " LIMIT 0,10")
		if got != want {
			t.Errorf("Cond.And(%q).Stmt() = %q, want %q", in, got, want)
		}
	}
}

func TestFilterExprInvalidFieldFailsClosed(t *testing.T) {
	for _, field := range []string{"", "a`=1 OR 1=1 --", "t..name", "t.name;drop", "t.*"} {
		expr, e := asql.NewFilterExpr("x")
		if e != nil {
			t.Fatalf("NewFilterExpr error: %v", e)
		}
		if _, _, e := mysqlx.FilterExpr(field, expr); e == nil {
			t.Errorf("FilterExpr(%q) did not fail", field)
		}
	}
}

func TestCondLikeWildcardsAreEscaped(t *testing.T) {
	cases := map[string]string{
		":@100%": "`t`.`name` LIKE '%100!%' ESCAPE '!'",
		":@a_b":  "`t`.`name` LIKE '%a!_b' ESCAPE '!'",
		":@a!b":  "`t`.`name` LIKE '%a!!b' ESCAPE '!'",
	}
	for in, want := range cases {
		cond := &mysqlx.Cond{}
		cond.And("t.name", in)
		got := strings.TrimSuffix(strings.TrimPrefix(cond.Stmt(), " WHERE "), " LIMIT 0,10")
		if got != want {
			t.Errorf("Cond.And(%q).Stmt() = %q, want %q", in, got, want)
		}
	}
}

func TestWhere(t *testing.T) {
	if got := mysqlx.Where(); got != "" {
		t.Errorf("Where() = %q, want empty", got)
	}
	if got := mysqlx.Where("`a`=1", "AND"); got != " WHERE `a`=1 " {
		t.Errorf("Where() = %q", got)
	}
	// A dangling connector is removed and duplicated connectors collapse.
	if got := mysqlx.Where("`a`=1 AND OR `b`=2"); got != " WHERE `a`=1 OR `b`=2 " {
		t.Errorf("Where() = %q", got)
	}
}
