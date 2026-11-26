package asql

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/atype"
)

// StructFields resolves names as struct field names when byFieldName is true,
// otherwise it resolves them as struct tags. The returned slices preserve the
// requested ordering.
func StructFields(value any, tagName string, byFieldName bool, names ...string) ([]string, []any, *ae.Error) {
	v := reflect.ValueOf(value)
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil, badSqlError("nil struct")
		}
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return nil, nil, badSqlError("expected struct")
	}
	t := v.Type()

	fields := make([]string, 0, len(names))
	values := make([]any, 0, len(names))
	for _, name := range names {
		found := false
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			match := field.Name == name
			if !byFieldName {
				match = field.Tag.Get(tagName) == name
			}
			if !match {
				continue
			}
			if !v.Field(i).CanInterface() {
				return nil, nil, badSqlError("field is not exported: " + name)
			}
			fields = append(fields, field.Tag.Get(tagName))
			values = append(values, v.Field(i).Interface())
			found = true
			break
		}
		if !found {
			return nil, nil, badSqlError("field not found: " + name)
		}
	}
	return fields, values, nil
}

// StructFieldSelection resolves the requested fields, treating them as Go
// field names when the first one is capitalised and as `db` tags otherwise.
func StructFieldSelection(value any, fields ...string) ([]string, []any, *ae.Error) {
	if len(fields) == 0 {
		return nil, nil, nil
	}
	first := fields[0]
	byFieldName := first == "" || (first[0] >= 'A' && first[0] <= 'Z')
	return StructFields(value, atype.DbTagName, byFieldName, fields...)
}

// StructColumns renders the requested struct fields as quoted column names.
func StructColumns(dialect *Dialect, value any, fields ...string) (string, *ae.Error) {
	names, _, e := StructFieldSelection(value, fields...)
	if e != nil {
		return "", e
	}
	quoted := make([]string, len(names))
	for i, name := range names {
		if quoted[i], e = dialect.Column(name); e != nil {
			return "", e
		}
	}
	return strings.Join(quoted, ", "), nil
}

// StructConditions renders "col=? AND col2=?" (or the dialect's markers) from
// struct values. The values are returned as arguments, never inlined.
func StructConditions(dialect *Dialect, value any, operator string, like bool, fields ...string) (string, []any, *ae.Error) {
	if dialect == nil || dialect.Bind == nil {
		return "", nil, badSqlError("missing SQL dialect")
	}
	connector, ok := normalizeConnector(operator)
	if !ok {
		return "", nil, badSqlError("invalid connector: " + operator)
	}
	names, values, e := StructFieldSelection(value, fields...)
	if e != nil {
		return "", nil, e
	}
	parts := make([]string, len(names))
	args := make([]any, len(names))
	for i, name := range names {
		column, e := dialect.Column(name)
		if e != nil {
			return "", nil, e
		}
		marker := dialect.Bind(i + 1)
		if like {
			text, isText := values[i].(string)
			if !isText {
				text = fmt.Sprint(values[i])
			}
			args[i] = "%" + escapeLike(text) + "%"
			parts[i] = column + " LIKE " + marker + ` ESCAPE '!'`
			continue
		}
		args[i] = values[i]
		parts[i] = column + "=" + marker
	}
	return strings.Join(parts, " "+connector+" "), args, nil
}

// StructConditionsLiteral renders StructConditions with escaped literals.
// It returns "1!=1" (fail closed) when a field or value cannot be rendered.
func StructConditionsLiteral(dialect *Dialect, value any, operator string, like bool, fields ...string) string {
	condition, args, e := StructConditions(dialect, value, operator, like, fields...)
	if e != nil {
		return "1!=1"
	}
	literal, ok := dialect.InlineArgs(condition, args, 0)
	if !ok {
		return "1!=1"
	}
	return literal
}

// WhereClause wraps a non-empty condition into a WHERE clause.
func WhereClause(condition string) string {
	if condition == "" {
		return ""
	}
	return " WHERE " + condition + " "
}

var (
	whereEmptyParenRe    = mustCompile(`\(\s*(OR|AND)*\s*\)`)
	whereRepeatedOpRe    = mustCompile(`((OR|AND)\s+)+(OR|AND)`)
	whereLeadingTailRe   = mustCompile(`(^(\s*(OR|AND))+)|(((OR|AND)+\s*)$)`)
	whereLeadingParenRe  = mustCompile(`\((\s*(OR|AND))+`)
	whereTrailingParenRe = mustCompile(`(\s*(OR|AND))+\)`)
)

func mustCompile(pattern string) *regexp.Regexp { return regexp.MustCompile(pattern) }

// NormalizeWhere cleans repeated, dangling and empty logical connectors and
// wraps a non-empty expression in a WHERE clause.
func NormalizeWhere(conditions ...string) string {
	var b strings.Builder
	for _, condition := range conditions {
		b.WriteByte(' ')
		b.WriteString(condition)
	}
	where := whereEmptyParenRe.ReplaceAllString(b.String(), " ")
	where = whereRepeatedOpRe.ReplaceAllString(where, "$3")
	where = whereLeadingTailRe.ReplaceAllString(where, "")
	where = whereLeadingParenRe.ReplaceAllString(where, "(")
	where = whereTrailingParenRe.ReplaceAllString(where, ")")
	where = strings.TrimSpace(where)
	if where == "" {
		return ""
	}
	return " WHERE " + where + " "
}
