package mysqlx_test

import (
	"testing"

	"github.com/aarioai/aago/driver/mysqlx"
)

type stru struct {
	Name string `db:"name"`
	Age  int    `db:"age"`
}

func TestAnd(t *testing.T) {
	u := stru{
		Name: "Aario",
		Age:  18,
	}
	if got, want := mysqlx.And(u, "name", "age"), "`name`='Aario' AND `age`=18"; got != want {
		t.Errorf("And(u, \"name\", \"age\") = %q, want %q", got, want)
	}
	if got, want := mysqlx.And(u, "Name", "Age"), "`name`='Aario' AND `age`=18"; got != want {
		t.Errorf("And(u, \"Name\", \"Age\") = %q, want %q", got, want)
	}
}

func TestAndArgs(t *testing.T) {
	u := stru{
		Name: "Aario",
		Age:  18,
	}
	sql, args, e := mysqlx.AndArgs(u, "name", "age")
	if e != nil {
		t.Fatalf("AndArgs error: %v", e)
	}
	if sql != "`name`=? AND `age`=?" {
		t.Errorf("AndArgs sql = %q", sql)
	}
	if len(args) != 2 || args[0] != "Aario" || args[1] != 18 {
		t.Errorf("AndArgs args = %#v", args)
	}
}

func TestAndFailsClosed(t *testing.T) {
	u := stru{Name: "Aario", Age: 18}
	if got := mysqlx.And(u, "name`=1 OR 1=1 --"); got != "1!=1" {
		t.Errorf("And(invalid field) = %q, want fail-closed", got)
	}
	if got := mysqlx.AndWithWhere(u, "name`=1 OR 1=1 --"); got != " WHERE 1!=1 " {
		t.Errorf("AndWithWhere(invalid field) = %q, want fail-closed", got)
	}
}

func TestOr(t *testing.T) {
	u := stru{
		Name: "Aario",
		Age:  18,
	}
	if got, want := mysqlx.Or(u, "name", "age"), "`name`='Aario' OR `age`=18"; got != want {
		t.Errorf("Or(u, \"name\", \"age\") = %q, want %q", got, want)
	}
	if got, want := mysqlx.Or(u, "Name", "Age"), "`name`='Aario' OR `age`=18"; got != want {
		t.Errorf("Or(u, \"Name\", \"Age\") = %q, want %q", got, want)
	}
}

func TestLike(t *testing.T) {
	u := stru{Name: "Aario%", Age: 18}
	if got, want := mysqlx.Like(u, "name"), "`name` LIKE '%Aario!%%' ESCAPE '!'"; got != want {
		t.Errorf("Like = %q, want %q", got, want)
	}
}

func TestAndWithWhere(t *testing.T) {
	u := stru{
		Name: "Aario",
		Age:  18,
	}
	if got, want := mysqlx.AndWithWhere(u, "name", "age"), " WHERE `name`='Aario' AND `age`=18 "; got != want {
		t.Errorf("AndWithWhere = %q, want %q", got, want)
	}
	if got, want := mysqlx.AndWithWhere(u, "Name", "Age"), " WHERE `name`='Aario' AND `age`=18 "; got != want {
		t.Errorf("AndWithWhere = %q, want %q", got, want)
	}
}

func TestOrWithWhere(t *testing.T) {
	u := stru{
		Name: "Aario",
		Age:  18,
	}
	if got, want := mysqlx.OrWithWhere(u, "name", "age"), " WHERE `name`='Aario' OR `age`=18 "; got != want {
		t.Errorf("OrWithWhere = %q, want %q", got, want)
	}
	if got, want := mysqlx.OrWithWhere(u, "Name", "Age"), " WHERE `name`='Aario' OR `age`=18 "; got != want {
		t.Errorf("OrWithWhere = %q, want %q", got, want)
	}
}

func TestComma(t *testing.T) {
	u := stru{Name: "Aario", Age: 18}
	if got, want := mysqlx.Comma(u, "name", "age"), "`name`, `age`"; got != want {
		t.Errorf("Comma = %q, want %q", got, want)
	}
	if got, want := mysqlx.CommaWithEnd(u, "name", "age"), "`name`, `age`, "; got != want {
		t.Errorf("CommaWithEnd = %q, want %q", got, want)
	}
	if got, want := mysqlx.CommaWithHead(u, "name", "age"), ",`name`, `age`"; got != want {
		t.Errorf("CommaWithHead = %q, want %q", got, want)
	}
	if got := mysqlx.Comma(u); got != "" {
		t.Errorf("Comma() = %q, want empty", got)
	}
}
