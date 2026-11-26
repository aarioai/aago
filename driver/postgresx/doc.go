// Package postgresx provides PostgreSQL helpers using pgx/v5 and pgxpool.
// Every statement builder (conditions, struct fragments, INSERT/IN/UNION,
// clamped increments and ORM clauses) is shared with mysqlx through
// driver/asql; postgresx only supplies the PostgreSQL dialect (double-quoted
// identifiers, "$n" markers and "LIMIT limit OFFSET offset").
package postgresx
