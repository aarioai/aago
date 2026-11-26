package postgresx

import (
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver/asql"
)

// InsertMany builds a multi-row INSERT statement with numbered placeholders.
// Example: InsertMany("users", "(name, created_at)", "(?, now())", len(ts)).
// It returns an empty string when num <= 0.
func InsertMany(table, columns, valuePattern string, num int) (string, *ae.Error) {
	return asql.InsertMany(&asql.PostgresDialect, table, columns, valuePattern, num)
}

// In renders `"id" IN (ids)`; an empty set fails closed with "1!=1".
func In(field string, ids map[uint64]struct{}) string {
	return asql.InUint64Set(&asql.PostgresDialect, field, ids)
}

// InUint renders `"id" IN (ids)` for uint identifiers.
func InUint(field string, ids map[uint]struct{}) string {
	return asql.InUintSet(&asql.PostgresDialect, field, ids)
}

// InValues renders `"code" IN ($1,$2)` and returns the values as arguments.
func InValues(field string, values map[string]struct{}) (string, []any, *ae.Error) {
	return asql.InStringSet(&asql.PostgresDialect, field, values)
}

// ArgStmt describes one column assignment of a partial UPDATE.
type ArgStmt = asql.ArgStmt

// ArgPairs renders `"a"=$1,"b"=$2` for the valid assignments plus the
// condition id. It reports false when nothing has to be updated.
func ArgPairs(conditionID any, statements []ArgStmt) (string, []any, bool) {
	return asql.ArgPairs(&asql.PostgresDialect, conditionID, statements)
}
