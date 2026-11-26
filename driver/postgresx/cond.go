package postgresx

import (
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/atype"
	"github.com/aarioai/aago/driver/asql"
)

// Cond builds a PostgreSQL WHERE / ORDER BY / LIMIT suffix on top of
// asql.Cond. The zero value is ready to use and defaults to the PostgreSQL
// dialect.
type Cond struct {
	asql.Cond
}

// NewCond creates a condition seeded with the paging values.
func NewCond(paging atype.Paging) *Cond {
	return &Cond{Cond: *asql.NewCond(&asql.PostgresDialect, paging)}
}

func (c *Cond) base() *asql.Cond {
	return c.Cond.EnsureDialect(&asql.PostgresDialect)
}

// Err returns the first builder error. Stmt fails closed when it is set.
func (c *Cond) Err() *ae.Error { return c.base().Err() }

// Args appends values for the placeholders written through Write/WriteString.
func (c *Cond) Args(args ...any) *Cond {
	c.base().Args(args...)
	return c
}

// WriteString appends a trusted SQL fragment. Use '$n' and Args for values.
func (c *Cond) WriteString(fragment string) *Cond {
	c.base().WriteString(fragment)
	return c
}

// Write appends a trusted fragment with an AND/OR connector.
func (c *Cond) Write(operator, fragment string) *Cond {
	c.base().Write(operator, fragment)
	return c
}

// Concat appends an asql.FilterExpr condition for field.
func (c *Cond) Concat(operator, field, grammar string) *Cond {
	c.base().Concat(operator, field, grammar)
	return c
}

// And appends an AND condition parsed from the asql.FilterExpr grammar.
func (c *Cond) And(field, grammar string) *Cond {
	c.base().And(field, grammar)
	return c
}

// Or appends an OR condition parsed from the asql.FilterExpr grammar.
func (c *Cond) Or(field, grammar string) *Cond {
	c.base().Or(field, grammar)
	return c
}

// OrderBy appends validated ORDER BY terms.
func (c *Cond) OrderBy(expression string) *Cond {
	c.base().OrderBy(expression)
	return c
}

// Limit sets the pagination window.
func (c *Cond) Limit(offset uint, limit uint16) *Cond {
	c.base().Limit(offset, limit)
	return c
}

// Try fills the order and paging values that are still empty.
func (c *Cond) Try(orderBy string, offset uint, limit uint16) *Cond {
	c.base().Try(orderBy, offset, limit)
	return c
}

// LimitN returns the effective row limit (defaults to 10).
func (c *Cond) LimitN() uint16 { return c.base().LimitN() }

// LimitStmt renders the PostgreSQL "LIMIT limit OFFSET offset" clause.
func (c *Cond) LimitStmt() string { return c.base().LimitStmt() }

// OrderByStmt renders the ORDER BY clause.
func (c *Cond) OrderByStmt() string { return c.base().OrderByStmt() }

// Stmt renders the suffix with escaped literals; errors render "WHERE 1!=1".
func (c *Cond) Stmt() string { return c.base().Stmt() }

// StmtArgs renders the suffix with '$n' placeholders and its arguments.
func (c *Cond) StmtArgs() (string, []any) { return c.base().StmtArgs() }
