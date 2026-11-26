package mysqlx

import (
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver/asql"
)

// Comma renders the requested struct fields as back-quoted column names.
// Fields are resolved as `db` tags unless the first one is a Go field name.
func Comma(value any, fields ...string) string {
	columns, _ := asql.StructColumns(&asql.MySQLDialect, value, fields...)
	return columns
}

// CommaWithEnd is Comma with a trailing ", ".
func CommaWithEnd(value any, fields ...string) string {
	if columns := Comma(value, fields...); columns != "" {
		return columns + ", "
	}
	return ""
}

// CommaWithHead is Comma with a leading ",".
func CommaWithHead(value any, fields ...string) string {
	if columns := Comma(value, fields...); columns != "" {
		return "," + columns
	}
	return ""
}

// And renders "`a`='x' AND `b`='y'" with escaped literals.
// It fails closed with "1!=1" when a field or value cannot be rendered.
func And(value any, fields ...string) string {
	return asql.StructConditionsLiteral(&asql.MySQLDialect, value, "AND", false, fields...)
}

// Or renders the fields joined with OR and escaped literals.
func Or(value any, fields ...string) string {
	return asql.StructConditionsLiteral(&asql.MySQLDialect, value, "OR", false, fields...)
}

// Like renders "`a` LIKE '%x%'" conditions joined with OR.
func Like(value any, fields ...string) string {
	return asql.StructConditionsLiteral(&asql.MySQLDialect, value, "OR", true, fields...)
}

// AndArgs renders the AND condition with '?' placeholders and its arguments.
func AndArgs(value any, fields ...string) (string, []any, *ae.Error) {
	return asql.StructConditions(&asql.MySQLDialect, value, "AND", false, fields...)
}

// OrArgs renders the OR condition with '?' placeholders and its arguments.
func OrArgs(value any, fields ...string) (string, []any, *ae.Error) {
	return asql.StructConditions(&asql.MySQLDialect, value, "OR", false, fields...)
}

// LikeArgs renders LIKE conditions with '?' placeholders and its arguments.
func LikeArgs(value any, fields ...string) (string, []any, *ae.Error) {
	return asql.StructConditions(&asql.MySQLDialect, value, "OR", true, fields...)
}

// AndWithWhere wraps And into a WHERE clause.
func AndWithWhere(value any, fields ...string) string {
	return asql.WhereClause(And(value, fields...))
}

// OrWithWhere wraps Or into a WHERE clause.
func OrWithWhere(value any, fields ...string) string {
	return asql.WhereClause(Or(value, fields...))
}

// AndWithWhereArgs wraps AndArgs into a WHERE clause.
func AndWithWhereArgs(value any, fields ...string) (string, []any, *ae.Error) {
	condition, args, e := AndArgs(value, fields...)
	if e != nil || condition == "" {
		return condition, args, e
	}
	return asql.WhereClause(condition), args, nil
}

// OrWithWhereArgs wraps OrArgs into a WHERE clause.
func OrWithWhereArgs(value any, fields ...string) (string, []any, *ae.Error) {
	condition, args, e := OrArgs(value, fields...)
	if e != nil || condition == "" {
		return condition, args, e
	}
	return asql.WhereClause(condition), args, nil
}

// Where cleans duplicated connectors and wraps the expression in a WHERE clause.
func Where(conditions ...string) string {
	return asql.NormalizeWhere(conditions...)
}
