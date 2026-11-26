package mysqlx

import (
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver/asql"
)

// InsertMany builds a parameterized multi-row INSERT statement.
// Example: InsertMany("user", "(nickname, avatar, created_at)", "(?, ?, now())", len(ts)).
// It returns an empty string when num <= 0.
func InsertMany(table, columns, valuePattern string, num int) (string, *ae.Error) {
	return asql.InsertMany(&asql.MySQLDialect, table, columns, valuePattern, num)
}

// In renders "`id` IN (ids)"; an empty set fails closed with "1!=1".
func In(field string, ids map[uint64]struct{}) string {
	return asql.InUint64Set(&asql.MySQLDialect, field, ids)
}

// InUint renders "`id` IN (ids)" for uint identifiers.
func InUint(field string, ids map[uint]struct{}) string {
	return asql.InUintSet(&asql.MySQLDialect, field, ids)
}

// InValues renders "`code` IN (?,?)" and returns the values as arguments.
func InValues(field string, ids map[string]struct{}) (string, []any, *ae.Error) {
	return asql.InStringSet(&asql.MySQLDialect, field, ids)
}

// ArgStmt describes one column assignment of a partial UPDATE.
type ArgStmt = asql.ArgStmt

// ArgPairs renders "`a`=?,`b`=?" for the valid assignments plus the condition
// id. It reports false when nothing has to be updated.
func ArgPairs(conditionID any, statements []ArgStmt) (string, []any, bool) {
	return asql.ArgPairs(&asql.MySQLDialect, conditionID, statements)
}
