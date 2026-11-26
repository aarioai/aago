package postgresx_test

import (
	"testing"

	"github.com/aarioai/aago/driver/postgresx"
)

func TestPostgresSafeIncrementHelpers(t *testing.T) {
	if got, want := postgresx.SafeIncrementInt16Expression("count", -2), `"count"=CASE WHEN "count"-2<=0 THEN 0 ELSE "count"-2 END`; got != want {
		t.Errorf("SafeIncrementInt16Expression = %q, want %q", got, want)
	}
	if got := postgresx.SafeIncrementExpression(`count"=1 --`, 1, "100"); got != "" {
		t.Errorf("invalid identifier result = %q, want empty", got)
	}
	if got := postgresx.SafeConflictIncrementClause("users", []postgresx.ConflictIncrement{
		postgresx.ConflictInt16Increment("count"),
	}); got != `"count"=CASE WHEN "users"."count"-0<=-"excluded"."count" THEN 0 WHEN "excluded"."count">=32767-"users"."count" THEN 32767 ELSE "users"."count"+"excluded"."count" END` {
		t.Errorf("SafeConflictIncrementClause = %q", got)
	}
}
