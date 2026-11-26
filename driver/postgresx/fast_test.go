package postgresx_test

import (
	"reflect"
	"testing"

	"github.com/aarioai/aago/driver/postgresx"
)

func TestInsertManyBuildsNumberedPlaceholders(t *testing.T) {
	query, e := postgresx.InsertMany("public.users", "(name, created_at)", "(?, now())", 2)
	if e != nil {
		t.Fatal(e)
	}
	want := `INSERT INTO "public"."users" ("name","created_at") VALUES ($1, CURRENT_TIMESTAMP),($2, CURRENT_TIMESTAMP)`
	if query != want {
		t.Errorf("InsertMany = %q, want %q", query, want)
	}
}

func TestInValuesUsesBoundArguments(t *testing.T) {
	injected := `x' OR TRUE --`
	query, args, e := postgresx.InValues("code", map[string]struct{}{injected: {}, "safe": {}})
	if e != nil {
		t.Fatal(e)
	}
	if query != `"code" IN ($1,$2)` || len(args) != 2 || (args[0] != injected && args[1] != injected) {
		t.Errorf("InValues = %q %#v", query, args)
	}
}

func TestArgPairsUsesNumberedParameters(t *testing.T) {
	query, args, ok := postgresx.ArgPairs(19, []postgresx.ArgStmt{
		{Field: "name", Value: "Ada", Valid: true},
		{Field: "age", Value: 37, Valid: true},
	})
	if !ok || query != `"name"=$1,"age"=$2` || !reflect.DeepEqual(args, []any{"Ada", 37, 19}) {
		t.Errorf("ArgPairs = %q %#v %v", query, args, ok)
	}
}
