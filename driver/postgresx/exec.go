package postgresx

import (
	"context"

	"github.com/aarioai/aago/aa/ae"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Schema string
	Pool   *pgxpool.Pool
	err    *ae.Error
}

func NewDriver(schema string, pool *pgxpool.Pool, deferredErr *ae.Error) *DB {
	return &DB{Schema: schema, Pool: pool, err: deferredErr}
}

func (d *DB) check() *ae.Error {
	if d.err != nil {
		return d.err
	}
	if d.Pool == nil {
		return ae.NewError("postgres pool is nil")
	}
	return nil
}

func (d *DB) Prepare(ctx context.Context, name, query string) (*PreparedStmt, *ae.Error) {
	if e := d.check(); e != nil {
		return nil, e
	}
	conn, err := d.Pool.Acquire(ctx)
	if err != nil {
		return nil, newPostgresError(err, query)
	}
	description, err := conn.Conn().Prepare(ctx, name, query)
	if err != nil {
		conn.Release()
		return nil, newPostgresError(err, query)
	}
	return &PreparedStmt{conn: conn, Name: description.Name, Description: description}, nil
}

func (d *DB) Execute(ctx context.Context, query string, args ...any) (pgconn.CommandTag, *ae.Error) {
	if e := d.check(); e != nil {
		return pgconn.CommandTag{}, e
	}
	tag, err := d.Pool.Exec(ctx, query, args...)
	return tag, newPostgresError(err, query)
}

func (d *DB) Exec(ctx context.Context, query string, args ...any) *ae.Error {
	_, e := d.Execute(ctx, query, args...)
	return e
}

// Insert expects a PostgreSQL INSERT statement with a RETURNING clause.
func (d *DB) Insert(ctx context.Context, query string, args ...any) (uint64, *ae.Error) {
	var id uint64
	e := d.ScanArgs(ctx, query, args, &id)
	return id, e
}

func (d *DB) Update(ctx context.Context, query string, args ...any) (int64, *ae.Error) {
	tag, e := d.Execute(ctx, query, args...)
	if e != nil {
		return 0, e
	}
	return tag.RowsAffected(), nil
}

func (d *DB) QueryRow(ctx context.Context, query string, args ...any) (pgx.Row, *ae.Error) {
	if e := d.check(); e != nil {
		return nil, e
	}
	return d.Pool.QueryRow(ctx, query, args...), nil
}

func (d *DB) ScanArgs(ctx context.Context, query string, args []any, dest ...any) *ae.Error {
	if e := d.check(); e != nil {
		return e
	}
	row, e := d.QueryRow(ctx, query, args...)
	if e != nil {
		return e
	}
	return newPostgresError(row.Scan(dest...), query)
}

func (d *DB) ScanRow(ctx context.Context, query string, dest ...any) *ae.Error {
	return d.ScanArgs(ctx, query, nil, dest...)
}

func (d *DB) Scan(ctx context.Context, query string, id uint64, dest ...any) *ae.Error {
	return d.ScanArgs(ctx, query, []any{id}, dest...)
}

func (d *DB) ScanX(ctx context.Context, query string, id string, dest ...any) *ae.Error {
	return d.ScanArgs(ctx, query, []any{id}, dest...)
}

func (d *DB) ScanAny(ctx context.Context, query string, id any, dest ...any) *ae.Error {
	return d.ScanArgs(ctx, query, []any{id}, dest...)
}

func (d *DB) Query(ctx context.Context, query string, args ...any) (pgx.Rows, *ae.Error) {
	if e := d.check(); e != nil {
		return nil, e
	}
	rows, err := d.Pool.Query(ctx, query, args...)
	if err != nil {
		if rows != nil {
			rows.Close()
		}
		return nil, newPostgresError(err, query)
	}
	return rows, nil
}
