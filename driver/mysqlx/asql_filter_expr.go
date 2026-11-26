package mysqlx

import (
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver/asql"
)

// FilterExpr renders a shared asql expression using MySQL identifiers and '?'
// placeholders. An empty fragment with a nil error means "no condition".
func FilterExpr(field string, expression asql.FilterExpr) (string, []any, *ae.Error) {
	return asql.RenderFilterExpr(field, expression, asql.MySQLDialect.Quote, asql.MySQLDialect.Bind, asql.MySQLDialect.MaxIdentPart)
}

// quoteIdent validates and back-quotes a possibly qualified identifier.
func quoteIdent(name string) (string, *ae.Error) {
	return asql.MySQLDialect.Identifier(name)
}

// quoteColumnIdentifier validates and back-quotes a single column name.
func quoteColumnIdentifier(name string) (string, *ae.Error) {
	return asql.MySQLDialect.Column(name)
}
