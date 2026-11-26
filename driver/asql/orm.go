package asql

import (
	"strings"

	"github.com/aarioai/aago/aa/ae"
)

// AssignmentClause renders "col=marker,..." for the columns of a partial
// UPDATE and returns the values in the same order.
func AssignmentClause(dialect *Dialect, data map[string]any) (string, []any, *ae.Error) {
	if dialect == nil || dialect.Bind == nil {
		return "", nil, badSqlError("missing SQL dialect")
	}
	var b strings.Builder
	args := make([]any, 0, len(data))
	for name, value := range data {
		column, e := dialect.Column(name)
		if e != nil {
			return "", nil, e
		}
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		args = append(args, value)
		b.WriteString(column)
		b.WriteByte('=')
		b.WriteString(dialect.Bind(len(args)))
	}
	return b.String(), args, nil
}

// SelectColumns renders the column list of a SELECT and returns the matching
// scan destinations.
func SelectColumns(dialect *Dialect, destinations map[string]any) (string, []any, *ae.Error) {
	var b strings.Builder
	dest := make([]any, 0, len(destinations))
	for name, destination := range destinations {
		column, e := dialect.Column(name)
		if e != nil {
			return "", nil, e
		}
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(column)
		dest = append(dest, destination)
	}
	return b.String(), dest, nil
}
