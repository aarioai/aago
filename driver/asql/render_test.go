package asql_test

import (
	"reflect"
	"testing"

	"github.com/aarioai/aago/driver/asql"
)

func TestRenderFilterExprAcrossDialects(t *testing.T) {
	expr, e := asql.NewFilterExpr(":@a%_!")
	if e != nil {
		t.Fatal(e)
	}
	mysqlSQL, mysqlArgs, e := asql.RenderFilterExpr("t.name", expr, '`', func(int) string { return "?" })
	if e != nil {
		t.Fatal(e)
	}
	if mysqlSQL != "`t`.`name` LIKE ? ESCAPE '!'" || !reflect.DeepEqual(mysqlArgs, []any{"%a!%!_!!"}) {
		t.Fatalf("mysql render = %q %#v", mysqlSQL, mysqlArgs)
	}

	postgresSQL, postgresArgs, e := asql.RenderFilterExpr("t.name", expr, '"', func(index int) string {
		return "$" + string(rune('0'+index))
	}, 63)
	if e != nil {
		t.Fatal(e)
	}
	if postgresSQL != `"t"."name" LIKE $1 ESCAPE '!'` || !reflect.DeepEqual(postgresArgs, mysqlArgs) {
		t.Fatalf("postgres render = %q %#v", postgresSQL, postgresArgs)
	}
}

func TestQuoteIdentifierValidatesAndQuotes(t *testing.T) {
	got, e := asql.QuoteIdentifier("schema.table_1", '"', 63)
	if e != nil || got != `"schema"."table_1"` {
		t.Fatalf("QuoteIdentifier = %q, %v", got, e)
	}
	for _, invalid := range []string{"", "t..col", `a"; DROP TABLE t`, "1column"} {
		if _, e := asql.QuoteIdentifier(invalid, '"', 63); e == nil {
			t.Errorf("QuoteIdentifier accepted %q", invalid)
		}
	}
}

func TestStructFieldsAndNormalizeWhere(t *testing.T) {
	type row struct {
		Name string `db:"name"`
		Age  int    `db:"age"`
	}
	fields, values, e := asql.StructFields(row{Name: "Ada", Age: 37}, "db", false, "age", "name")
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(fields, []string{"age", "name"}) || !reflect.DeepEqual(values, []any{37, "Ada"}) {
		t.Fatalf("StructFields = %#v %#v", fields, values)
	}
	if got := asql.NormalizeWhere("AND", "(`a`=1 AND OR `b`=2)"); got != " WHERE (`a`=1 OR `b`=2) " {
		t.Fatalf("NormalizeWhere = %q", got)
	}
}
