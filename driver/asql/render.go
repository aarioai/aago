package asql

import (
	"strings"

	"github.com/aarioai/aago/aa/ae"
)

const DefaultIdentifierPartLimit = 64

// Placeholder returns a dialect-specific bind marker for a 1-based argument index.
type Placeholder func(index int) string

// RenderFilterExpr renders a parsed filter using the supplied identifier quoting
// and bind-marker styles. Values are always returned separately as arguments.
func RenderFilterExpr(field string, expr FilterExpr, identifierQuote byte, placeholder Placeholder, identifierPartLimit ...int) (string, []any, *ae.Error) {
	quotedField, e := QuoteIdentifier(field, identifierQuote, identifierPartLimit...)
	if e != nil {
		return "", nil, e
	}
	if placeholder == nil {
		return "", nil, badSqlError("missing placeholder formatter")
	}

	var args []any
	bind := func(value string) string {
		args = append(args, value)
		return placeholder(len(args))
	}

	switch {
	case expr.EqualTo != "":
		return quotedField + "=" + bind(expr.EqualTo), args, nil
	case expr.UnequalTo != "":
		return quotedField + "!=" + bind(expr.UnequalTo), args, nil
	case expr.Like != "":
		return quotedField + " LIKE " + bind("%"+escapeLike(expr.Like)+"%") + ` ESCAPE '!'`, args, nil
	case len(expr.In) > 0:
		markers := make([]string, len(expr.In))
		for i, value := range expr.In {
			markers[i] = bind(value)
		}
		return quotedField + " IN (" + strings.Join(markers, ",") + ")", args, nil
	case expr.GreaterOrEqualTo != "":
		lower := bind(expr.GreaterOrEqualTo)
		if expr.LessThan != "" {
			upper := bind(expr.LessThan)
			return "(" + quotedField + ">=" + lower + " AND " + quotedField + "<" + upper + ")", args, nil
		}
		if expr.LessOrEqualTo != "" {
			upper := bind(expr.LessOrEqualTo)
			return "(" + quotedField + ">=" + lower + " AND " + quotedField + "<=" + upper + ")", args, nil
		}
		return quotedField + ">=" + lower, args, nil
	case expr.LessThan != "":
		return quotedField + "<" + bind(expr.LessThan), args, nil
	case expr.GreaterThan != "":
		lower := bind(expr.GreaterThan)
		if expr.LessOrEqualTo != "" {
			upper := bind(expr.LessOrEqualTo)
			return "(" + quotedField + ">" + lower + " AND " + quotedField + "<=" + upper + ")", args, nil
		}
		return quotedField + ">" + lower, args, nil
	case expr.LessOrEqualTo != "":
		return quotedField + "<=" + bind(expr.LessOrEqualTo), args, nil
	case expr.StartWith != "":
		return quotedField + " LIKE " + bind(escapeLike(expr.StartWith)+"%") + ` ESCAPE '!'`, args, nil
	case expr.EndWith != "":
		return quotedField + " LIKE " + bind("%"+escapeLike(expr.EndWith)) + ` ESCAPE '!'`, args, nil
	default:
		return "", nil, nil
	}
}

// QuoteIdentifier validates and quotes a possibly qualified identifier.
func QuoteIdentifier(identifier string, quote byte, maxPartBytes ...int) (string, *ae.Error) {
	if identifier == "" || quote == 0 {
		return "", badSqlError("empty identifier or quote character")
	}
	limit := DefaultIdentifierPartLimit
	if len(maxPartBytes) > 0 {
		limit = maxPartBytes[0]
	}
	parts := strings.Split(identifier, ".")
	var b strings.Builder
	b.Grow(len(identifier) + 2*len(parts))
	for i, part := range parts {
		if part == "" || len(part) > limit {
			return "", badSqlError("invalid identifier: " + identifier)
		}
		for j, r := range part {
			if !validIdentifierRune(r, j == 0) {
				return "", badSqlError("invalid identifier: " + identifier)
			}
		}
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteByte(quote)
		b.WriteString(part)
		b.WriteByte(quote)
	}
	return b.String(), nil
}

func escapeLike(value string) string {
	return strings.NewReplacer(`!`, `!!`, `%`, `!%`, `_`, `!_`).Replace(value)
}

func validIdentifierRune(r rune, first bool) bool {
	switch {
	case r == '_' || r == '$':
		return true
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		return true
	case r > 127:
		return true
	case !first && r >= '0' && r <= '9':
		return true
	}
	return false
}
