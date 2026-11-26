package asql

import (
	"strings"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/atype"
)

// Cond incrementally builds a WHERE / ORDER BY / LIMIT suffix for a dialect.
//
// Conditions added through Concat are kept twice: escaped literals for Stmt and
// bind markers plus arguments for StmtArgs. Prefer StmtArgs with untrusted
// input.
//
// The zero value has no dialect; either construct it with NewCond or let a
// dialect specific wrapper call EnsureDialect. A builder error fails closed:
// Stmt and StmtArgs render "WHERE 1!=1".
type Cond struct {
	Constraint   strings.Builder
	Dialect      *Dialect
	orderBy      string
	offset       uint
	limit        uint16
	err          *ae.Error
	placeholders strings.Builder
	args         []any
}

func NewCond(dialect *Dialect, paging atype.Paging) *Cond {
	return &Cond{Dialect: dialect, offset: paging.Offset, limit: paging.Limit}
}

// EnsureDialect sets the dialect unless one is already configured.
func (c *Cond) EnsureDialect(dialect *Dialect) *Cond {
	if c.Dialect == nil {
		c.Dialect = dialect
	}
	return c
}

// Err returns the first builder error. Stmt fails closed when it is set.
func (c *Cond) Err() *ae.Error {
	return c.err
}

func (c *Cond) setErr(e *ae.Error) {
	if c.err == nil && e != nil {
		c.err = e
	}
}

// dialectOrFail records an error when the dialect is missing.
func (c *Cond) dialectOrFail() *Dialect {
	if c.Dialect == nil {
		c.setErr(badSqlError("missing SQL dialect"))
		return nil
	}
	return c.Dialect
}

// Args appends values for the placeholders written through Write/WriteString.
func (c *Cond) Args(args ...any) *Cond {
	c.args = append(c.args, args...)
	return c
}

// WriteString appends a trusted SQL fragment. Never pass user input directly;
// use '?' or '$n' placeholders and Args for values.
func (c *Cond) WriteString(fragment string) *Cond {
	c.Constraint.WriteByte(' ')
	c.Constraint.WriteString(fragment)
	c.placeholders.WriteByte(' ')
	c.placeholders.WriteString(fragment)
	return c
}

func (c *Cond) Write(operator, fragment string) *Cond {
	conn, ok := normalizeConnector(operator)
	if !ok {
		c.setErr(badSqlError("invalid connector: " + operator))
		return c
	}
	if c.placeholders.Len() == 0 {
		conn = ""
	} else {
		conn += " "
	}
	return c.WriteString(conn + fragment)
}

// Concat appends a FilterExpr condition for field.
func (c *Cond) Concat(operator, field, grammar string) *Cond {
	conn, ok := normalizeConnector(operator)
	if !ok {
		c.setErr(badSqlError("invalid connector: " + operator))
		return c
	}
	dialect := c.dialectOrFail()
	if dialect == nil {
		return c
	}
	expr, e := NewFilterExpr(grammar)
	if e != nil {
		c.setErr(e)
		return c
	}
	// Markers continue from the arguments already collected.
	next := len(c.args)
	placeholder, args, e := RenderFilterExpr(field, expr, dialect.Quote, func(index int) string {
		return dialect.Bind(index + next)
	}, dialect.MaxIdentPart)
	if e != nil {
		c.setErr(e)
		return c
	}
	if placeholder == "" {
		return c // empty condition: no filter
	}
	literal, ok := dialect.InlineArgs(placeholder, args, next)
	if !ok {
		c.setErr(badSqlError("filter value cannot be rendered as a SQL literal"))
		return c
	}
	c.add(conn, literal, placeholder, args)
	return c
}

func (c *Cond) add(conn, literal, placeholder string, args []any) {
	if c.placeholders.Len() > 0 {
		c.Constraint.WriteByte(' ')
		c.Constraint.WriteString(conn)
		c.Constraint.WriteByte(' ')
		c.placeholders.WriteByte(' ')
		c.placeholders.WriteString(conn)
		c.placeholders.WriteByte(' ')
	}
	c.Constraint.WriteString(literal)
	c.placeholders.WriteString(placeholder)
	c.args = append(c.args, args...)
}

func (c *Cond) And(field, grammar string) *Cond {
	return c.Concat("AND", field, grammar)
}

func (c *Cond) Or(field, grammar string) *Cond {
	return c.Concat("OR", field, grammar)
}

func (c *Cond) OrderBy(expression string) *Cond {
	dialect := c.dialectOrFail()
	if dialect == nil {
		return c
	}
	fragment, e := FormatOrderBy(dialect, expression)
	if e != nil {
		c.setErr(e)
		return c
	}
	if fragment == "" {
		return c
	}
	if c.orderBy == "" {
		c.orderBy = fragment
	} else {
		c.orderBy += ", " + fragment
	}
	return c
}

func (c *Cond) Limit(offset uint, limit uint16) *Cond {
	c.offset = offset
	c.limit = limit
	return c
}

func (c *Cond) Try(orderBy string, offset uint, limit uint16) *Cond {
	if c.orderBy == "" {
		c.OrderBy(orderBy)
	}
	if c.limit == 0 {
		c.offset = offset
		c.limit = limit
	}
	return c
}

func (c *Cond) LimitN() uint16 {
	if c.limit > 0 {
		return c.limit
	}
	return 10
}

func (c *Cond) LimitStmt() string {
	if c.Dialect == nil || c.Dialect.Limit == nil {
		return ""
	}
	return c.Dialect.Limit(c.offset, c.LimitN())
}

func (c *Cond) OrderByStmt() string {
	return "ORDER BY " + c.orderBy
}

// Stmt renders the WHERE/ORDER BY/LIMIT suffix with escaped literals.
// Check Err first; a builder error renders "WHERE 1!=1" (fail closed).
func (c *Cond) Stmt() string {
	if c.err != nil {
		return c.stmt("1!=1")
	}
	return c.stmt(c.Constraint.String())
}

// StmtArgs renders the same suffix with bind markers and their arguments.
// Check Err first; a builder error renders "WHERE 1!=1" (fail closed).
func (c *Cond) StmtArgs() (string, []any) {
	if c.err != nil {
		return c.stmt("1!=1"), nil
	}
	return c.stmt(c.placeholders.String()), c.args
}

func (c *Cond) stmt(where string) string {
	var b strings.Builder
	if where = strings.TrimSpace(where); where != "" {
		b.WriteString(" WHERE ")
		b.WriteString(where)
	}
	if c.orderBy != "" {
		b.WriteString(" ORDER BY ")
		b.WriteString(c.orderBy)
	}
	if limit := c.LimitStmt(); limit != "" {
		b.WriteByte(' ')
		b.WriteString(limit)
	}
	return b.String()
}

// normalizeConnector validates an AND/OR connector.
func normalizeConnector(operator string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(operator)) {
	case "AND":
		return "AND", true
	case "OR":
		return "OR", true
	}
	return "", false
}

// FormatOrderBy validates and quotes every ORDER BY term.
func FormatOrderBy(dialect *Dialect, orderBy string) (string, *ae.Error) {
	if strings.TrimSpace(orderBy) == "" {
		return "", nil
	}
	if dialect == nil {
		return "", badSqlError("missing SQL dialect")
	}
	parts := strings.Split(orderBy, ",")
	formatted := make([]string, len(parts))
	for i, part := range parts {
		tokens := strings.Fields(strings.TrimSpace(part))
		if len(tokens) == 0 || len(tokens) > 2 {
			return "", badSqlError("invalid ORDER BY expression")
		}
		field, e := dialect.Identifier(tokens[0])
		if e != nil {
			return "", e
		}
		formatted[i] = field
		if len(tokens) == 2 {
			direction := strings.ToUpper(tokens[1])
			if direction != "ASC" && direction != "DESC" {
				return "", badSqlError("invalid ORDER BY direction: " + tokens[1])
			}
			formatted[i] += " " + direction
		}
	}
	return strings.Join(formatted, ", "), nil
}
