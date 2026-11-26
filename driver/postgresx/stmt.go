package postgresx

import (
	"context"
	"errors"

	"github.com/aarioai/aago/aa/ae"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PreparedStmt pins a pool connection for the lifetime of a named prepared
// statement. Close releases the connection back to the pool.
type PreparedStmt struct {
	conn        *pgxpool.Conn
	Name        string
	Description *pgconn.StatementDescription
}

func (s *PreparedStmt) Exec(ctx context.Context, args ...any) (pgconn.CommandTag, *ae.Error) {
	conn, e := s.connection()
	if e != nil {
		return pgconn.CommandTag{}, e
	}
	tag, err := conn.Exec(ctx, s.Name, args...)
	return tag, newPostgresError(err, s.Name)
}

func (s *PreparedStmt) Query(ctx context.Context, args ...any) (pgx.Rows, *ae.Error) {
	conn, e := s.connection()
	if e != nil {
		return nil, e
	}
	rows, err := conn.Query(ctx, s.Name, args...)
	if err != nil {
		if rows != nil {
			rows.Close()
		}
		return nil, newPostgresError(err, s.Name)
	}
	return rows, nil
}

func (s *PreparedStmt) QueryRow(ctx context.Context, args ...any) (pgx.Row, *ae.Error) {
	conn, e := s.connection()
	if e != nil {
		return nil, e
	}
	return conn.QueryRow(ctx, s.Name, args...), nil
}

func (s *PreparedStmt) Scan(ctx context.Context, args []any, dest ...any) *ae.Error {
	row, e := s.QueryRow(ctx, args...)
	if e != nil {
		return e
	}
	return newPostgresError(row.Scan(dest...), s.Name)
}

func (s *PreparedStmt) Close() {
	if s.conn != nil {
		s.conn.Release()
		s.conn = nil
	}
}

func (s *PreparedStmt) connection() (*pgx.Conn, *ae.Error) {
	if s.conn == nil {
		return nil, newPostgresError(errors.New("prepared statement is closed"))
	}
	return s.conn.Conn(), nil
}
