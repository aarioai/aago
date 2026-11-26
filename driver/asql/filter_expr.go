package asql

import (
	"slices"
	"strings"

	"github.com/aarioai/aago/aa/ae"
)

const (
	maxFilterExprLen = 4096
	maxFilterExprIn  = 512
)

// FilterExpr is a compact condition grammar parsed from one query value.
//
// The leading ':' introduces an operator:
//
//	Aario                                          field = 'Aario'
//	:Aario,Tom                                     field IN ('Aario','Tom')
//	:@Aario@                                       field LIKE '%Aario%'
//	:@Aario                                        field LIKE '%Aario'
//	:Aario@                                        field LIKE 'Aario%'
//	:2019-06-01 00:00:00~2019-06-01 01:00:00       field >= ... AND field < ...
//	:2019-06-01 00:00:00~                          field >= ...
//	:~2019-06-01 00:00:00                          field < ...
//
// '@', '~' and ',' are reserved inside the grammar; there is no escape sequence.
// The legacy '::Aario' / ':Aario:' forms are rejected instead of silently
// degrading to an equality match.
//
// Values are never trusted. RenderFilterExpr returns placeholders and separate
// arguments; prefer it (or a driver's Cond.StmtArgs) for untrusted input.
type FilterExpr struct {
	EqualTo          string
	UnequalTo        string
	Like             string
	In               []string
	GreaterOrEqualTo string
	LessThan         string
	GreaterThan      string
	LessOrEqualTo    string
	StartWith        string
	EndWith          string
}

// NewFilterExpr parses the compact grammar. An empty value means "no condition"
// and returns the zero FilterExpr with a nil error.
func NewFilterExpr(v string, lengths ...int) (FilterExpr, *ae.Error) {
	if v == "" {
		return FilterExpr{}, nil
	}
	maxExprLen := maxFilterExprLen
	maxExprIn := maxFilterExprIn

	switch len(lengths) {
	case 1:
		maxExprLen = lengths[0]
	case 2:
		maxExprLen = lengths[0]
		maxExprIn = lengths[1]
	}

	if len(v) > maxExprLen {
		return FilterExpr{}, badSqlError("filter expression value too long")
	}
	if v[0] != ':' {
		return FilterExpr{EqualTo: v}, nil
	}
	// Reject the removed grammar so old callers fail loudly.
	if strings.HasPrefix(v, "::") || (len(v) > 2 && strings.HasSuffix(v, ":")) {
		return FilterExpr{}, badSqlError("legacy '::' / ':' grammar is not supported, use '@'")
	}

	body := v[1:]
	if body == "" {
		return FilterExpr{}, badSqlError("missing value after ':'")
	}
	if body[0] == '@' {
		rest := body[1:]
		if rest == "" {
			return FilterExpr{}, badSqlError("missing value after '@'")
		}
		if strings.HasSuffix(rest, "@") {
			value := rest[:len(rest)-1]
			if value == "" {
				return FilterExpr{}, badSqlError("empty LIKE value")
			}
			return FilterExpr{Like: value}, nil
		}
		return FilterExpr{EndWith: rest}, nil
	}
	if strings.HasSuffix(body, "@") {
		value := body[:len(body)-1]
		if value == "" {
			return FilterExpr{}, badSqlError("empty prefix value")
		}
		return FilterExpr{StartWith: value}, nil
	}
	if before, after, ok := strings.Cut(body, "~"); ok {
		lower, upper := before, after
		if lower == "" && upper == "" {
			return FilterExpr{}, badSqlError("empty range")
		}
		return FilterExpr{GreaterOrEqualTo: lower, LessThan: upper}, nil
	}
	if strings.IndexByte(body, ',') >= 0 {
		values := strings.Split(body, ",")
		if len(values) > maxExprIn {
			return FilterExpr{}, badSqlError("too many IN values")
		}
		if slices.Contains(values, "") {
			return FilterExpr{}, badSqlError("empty IN value")
		}
		return FilterExpr{In: values}, nil
	}
	return FilterExpr{EqualTo: body}, nil
}
