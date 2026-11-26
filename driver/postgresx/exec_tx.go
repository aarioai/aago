package postgresx

import (
	"context"
	"log"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/alog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Tx struct {
	Tx   pgx.Tx
	done bool
}

func (d *DB) Begin(ctx context.Context, opts pgx.TxOptions) (*Tx, *ae.Error) {
	if e := d.check(); e != nil {
		return nil, e
	}
	tx, err := d.Pool.BeginTx(ctx, opts)
	if err != nil {
		return nil, newPostgresError(err)
	}
	return &Tx{Tx: tx}, nil
}

func (t *Tx) Commit(ctx context.Context) *ae.Error {
	t.done = true
	return newPostgresError(t.Tx.Commit(ctx))
}

func (t *Tx) Rollback(ctx context.Context) *ae.Error {
	t.done = true
	return newPostgresError(t.Tx.Rollback(ctx))
}

// Recover must be deferred: defer tx.Recover(ctx)().
// Any transaction left open is rolled back; a panic is rethrown after rollback.
func (t *Tx) Recover(ctx context.Context) func() {
	return func() {
		if p := recover(); p != nil {
			if !t.done {
				t.done = true
				alog.OnError(t.Tx.Rollback(ctx))
			}
			panic(p)
		}
		if t.done {
			return
		}
		t.done = true
		log.Println("[warning] tx not committed, rolling back")
		alog.OnError(t.Tx.Rollback(ctx))
	}
}

func (t *Tx) Prepare(ctx context.Context, name, query string) (*pgconn.StatementDescription, *ae.Error) {
	stmt, err := t.Tx.Prepare(ctx, name, query)
	return stmt, newPostgresError(err, query)
}

func (t *Tx) Execute(ctx context.Context, query string, args ...any) (pgconn.CommandTag, *ae.Error) {
	tag, err := t.Tx.Exec(ctx, query, args...)
	return tag, newPostgresError(err, query)
}

func (t *Tx) Exec(ctx context.Context, query string, args ...any) *ae.Error {
	_, e := t.Execute(ctx, query, args...)
	return e
}

// Insert expects a PostgreSQL INSERT statement with a RETURNING clause.
func (t *Tx) Insert(ctx context.Context, query string, args ...any) (uint64, *ae.Error) {
	var id uint64
	e := t.ScanArgs(ctx, query, args, &id)
	return id, e
}

func (t *Tx) Update(ctx context.Context, query string, args ...any) (int64, *ae.Error) {
	tag, e := t.Execute(ctx, query, args...)
	if e != nil {
		return 0, e
	}
	return tag.RowsAffected(), nil
}

func (t *Tx) QueryRow(ctx context.Context, query string, args ...any) (pgx.Row, *ae.Error) {
	return t.Tx.QueryRow(ctx, query, args...), nil
}

func (t *Tx) ScanArgs(ctx context.Context, query string, args []any, dest ...any) *ae.Error {
	row, e := t.QueryRow(ctx, query, args...)
	if e != nil {
		return e
	}
	return newPostgresError(row.Scan(dest...), query)
}

func (t *Tx) ScanRow(ctx context.Context, query string, dest ...any) *ae.Error {
	return t.ScanArgs(ctx, query, nil, dest...)
}

func (t *Tx) Scan(ctx context.Context, query string, id uint64, dest ...any) *ae.Error {
	return t.ScanArgs(ctx, query, []any{id}, dest...)
}

func (t *Tx) ScanX(ctx context.Context, query string, id string, dest ...any) *ae.Error {
	return t.ScanArgs(ctx, query, []any{id}, dest...)
}

func (t *Tx) ScanAny(ctx context.Context, query string, id any, dest ...any) *ae.Error {
	return t.ScanArgs(ctx, query, []any{id}, dest...)
}

func (t *Tx) Query(ctx context.Context, query string, args ...any) (pgx.Rows, *ae.Error) {
	rows, err := t.Tx.Query(ctx, query, args...)
	if err != nil {
		if rows != nil {
			rows.Close()
		}
		return nil, newPostgresError(err, query)
	}
	return rows, nil
}
