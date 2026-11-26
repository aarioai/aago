package postgresx_test

import (
	"reflect"
	"testing"

	"github.com/aarioai/aago/driver/postgresx"
)

type testUser struct {
	Name string `db:"name"`
	Age  int    `db:"age"`
}

func TestStructConditions(t *testing.T) {
	user := testUser{Name: "O'Reilly", Age: 42}
	condition, args, e := postgresx.AndArgs(user, "name", "age")
	if e != nil {
		t.Fatal(e)
	}
	if condition != `"name"=$1 AND "age"=$2` || !reflect.DeepEqual(args, []any{"O'Reilly", 42}) {
		t.Errorf("AndArgs = %q %#v", condition, args)
	}
	if got := postgresx.And(user, "name"); got != `"name"=E'O\'Reilly'` {
		t.Errorf("And = %q", got)
	}
	if got := postgresx.Where("AND", `("a"=1 AND OR "b"=2)`); got != ` WHERE ("a"=1 OR "b"=2) ` {
		t.Errorf("Where = %q", got)
	}
}

func TestStructComma(t *testing.T) {
	user := testUser{Name: "Ada", Age: 37}
	if got, want := postgresx.Comma(user, "name", "age"), `"name", "age"`; got != want {
		t.Errorf("Comma = %q, want %q", got, want)
	}
	if got, want := postgresx.CommaWithEnd(user, "Name", "Age"), `"name", "age", `; got != want {
		t.Errorf("CommaWithEnd = %q, want %q", got, want)
	}
	if got := postgresx.Comma(user); got != "" {
		t.Errorf("Comma() = %q, want empty", got)
	}
}

func TestStructConditionsFailClosed(t *testing.T) {
	user := testUser{Name: "Ada", Age: 37}
	if got := postgresx.Or(user, `name" OR TRUE --`); got != "1!=1" {
		t.Errorf("Or(invalid field) = %q, want fail-closed", got)
	}
	sql, args, e := postgresx.LikeArgs(user, "name")
	if e != nil || sql != `"name" LIKE $1 ESCAPE '!'` || !reflect.DeepEqual(args, []any{"%Ada%"}) {
		t.Errorf("LikeArgs = %q %#v %v", sql, args, e)
	}
}
