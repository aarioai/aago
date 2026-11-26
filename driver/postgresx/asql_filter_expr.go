package postgresx

import (
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver/asql"
)

// FilterExpr renders a shared asql expression using PostgreSQL identifiers and
// numbered bind parameters. An empty fragment with a nil error means
// "no condition".
func FilterExpr(field string, expression asql.FilterExpr) (string, []any, *ae.Error) {
	return asql.RenderFilterExpr(field, expression, asql.PostgresDialect.Quote, asql.PostgresDialect.Bind, asql.PostgresDialect.MaxIdentPart)
}

// quoteIdent validates and quotes a possibly qualified identifier.
func quoteIdent(name string) (string, *ae.Error) {
	return asql.PostgresDialect.Identifier(name)
}

// quoteColumnIdentifier validates and quotes a single column name.
func quoteColumnIdentifier(name string) (string, *ae.Error) {
	return asql.PostgresDialect.Column(name)
}
