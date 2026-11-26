package asql

import (
	"strings"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/pkg/types"
)

// InsertMany builds a parameterized multi-row INSERT statement.
// Example: InsertMany(dialect, "user", "(nickname, avatar)", "(?, now())", 3).
// It returns an empty statement when num <= 0.
func InsertMany(dialect *Dialect, table, columns, valuePattern string, num int) (string, *ae.Error) {
	if num <= 0 {
		return "", nil
	}
	if dialect == nil || dialect.Bind == nil {
		return "", badSqlError("missing SQL dialect")
	}
	quotedTable, e := dialect.Identifier(table)
	if e != nil {
		return "", e
	}
	quotedColumns, e := dialect.insertColumns(columns)
	if e != nil {
		return "", e
	}
	values, e := dialect.insertValues(valuePattern)
	if e != nil {
		return "", e
	}
	var b strings.Builder
	b.Grow(len(quotedTable) + len(quotedColumns) + num*(len(values)*2+8))
	b.WriteString("INSERT INTO ")
	b.WriteString(quotedTable)
	b.WriteByte(' ')
	b.WriteString(quotedColumns)
	b.WriteString(" VALUES ")
	next := 1
	for row := 0; row < num; row++ {
		if row > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('(')
		for i, value := range values {
			if i > 0 {
				b.WriteString(", ")
			}
			if value == insertBindValue {
				b.WriteString(dialect.Bind(next))
				next++
				continue
			}
			b.WriteString(value)
		}
		b.WriteByte(')')
	}
	return b.String(), nil
}

const insertBindValue = "?"

// insertColumns quotes the "(a, b)" column list of an INSERT statement.
func (d *Dialect) insertColumns(columns string) (string, *ae.Error) {
	if len(columns) < 3 || columns[0] != '(' || columns[len(columns)-1] != ')' {
		return "", badSqlError("invalid insert columns")
	}
	parts := strings.Split(columns[1:len(columns)-1], ",")
	quoted := make([]string, len(parts))
	for i, part := range parts {
		field, e := d.Column(strings.TrimSpace(part))
		if e != nil {
			return "", e
		}
		quoted[i] = field
	}
	return "(" + strings.Join(quoted, ",") + ")", nil
}

// insertValues normalizes the "(?, now())" value pattern; a "?" element marks
// a bind marker.
func (d *Dialect) insertValues(pattern string) ([]string, *ae.Error) {
	if len(pattern) < 3 || pattern[0] != '(' || pattern[len(pattern)-1] != ')' {
		return nil, badSqlError("invalid insert value pattern")
	}
	parts := strings.Split(pattern[1:len(pattern)-1], ",")
	values := make([]string, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		switch {
		case part == insertBindValue:
			values[i] = insertBindValue
		case strings.EqualFold(part, "now()"), strings.EqualFold(part, "current_timestamp"):
			values[i] = d.Now
		case strings.EqualFold(part, "default"):
			values[i] = "DEFAULT"
		default:
			return nil, badSqlError("invalid insert value pattern")
		}
	}
	return values, nil
}

// InUint64Set renders "field IN (ids)". It fails closed with "1!=1" for an
// empty set or an invalid field.
func InUint64Set(dialect *Dialect, field string, ids map[uint64]struct{}) string {
	values := make([]string, 0, len(ids))
	for id := range ids {
		values = append(values, types.FormatUint(id))
	}
	return inValues(dialect, field, values)
}

// InUintSet renders "field IN (ids)" for uint identifiers.
func InUintSet(dialect *Dialect, field string, ids map[uint]struct{}) string {
	values := make([]string, 0, len(ids))
	for id := range ids {
		values = append(values, types.FormatUint(id))
	}
	return inValues(dialect, field, values)
}

func inValues(dialect *Dialect, field string, values []string) string {
	if len(values) == 0 {
		return "1!=1"
	}
	column, e := dialect.Column(field)
	if e != nil {
		return "1!=1"
	}
	if len(values) == 1 {
		return column + "=" + values[0]
	}
	return column + " IN (" + strings.Join(values, ",") + ")"
}

// InStringSet renders "field IN (markers)" and returns the values as arguments.
// The values are never interpolated into the statement.
func InStringSet(dialect *Dialect, field string, values map[string]struct{}) (string, []any, *ae.Error) {
	if len(values) == 0 {
		return "1!=1", nil, nil
	}
	if dialect == nil || dialect.Bind == nil {
		return "", nil, badSqlError("missing SQL dialect")
	}
	column, e := dialect.Column(field)
	if e != nil {
		return "", nil, e
	}
	args := make([]any, 0, len(values))
	for value := range values {
		args = append(args, value)
	}
	return column + " IN (" + dialect.Placeholders(len(args)) + ")", args, nil
}

// ArgStmt describes one column assignment of a partial UPDATE.
type ArgStmt struct {
	Field   string
	Value   any
	Valid   bool
	Ignores []string // fields to skip, e.g. columns out of sync
}

// ArgPairs renders "field=marker,..." for the valid statements plus the
// trailing condition id. It returns false when nothing has to be updated.
func ArgPairs(dialect *Dialect, conditionID any, statements []ArgStmt) (string, []any, bool) {
	if dialect == nil || dialect.Bind == nil {
		return "", nil, false
	}
	assignments := make([]string, 0, len(statements))
	args := make([]any, 0, len(statements)+1)
	for _, statement := range statements {
		if !statement.Valid || statement.ignored() {
			continue
		}
		field, e := dialect.Column(statement.Field)
		if e != nil {
			return "", nil, false
		}
		args = append(args, statement.Value)
		assignments = append(assignments, field+"="+dialect.Bind(len(args)))
	}
	if len(assignments) == 0 {
		return "", nil, false
	}
	if conditionID != nil {
		args = append(args, conditionID)
	}
	return strings.Join(assignments, ","), args, true
}

func (s ArgStmt) ignored() bool {
	for _, ignored := range s.Ignores {
		if ignored == s.Field {
			return true
		}
	}
	return false
}
