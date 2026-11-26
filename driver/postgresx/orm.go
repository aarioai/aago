package postgresx

import (
	"context"
	"fmt"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver/asql"
	"github.com/aarioai/aago/driver/index"
)

type ORMS struct {
	db *DB
	t  index.Entity
}

func ORM(db *DB, entity index.Entity) *ORMS { return &ORMS{db: db, t: entity} }
func (d *DB) ORM(entity index.Entity) *ORMS { return ORM(d, entity) }

func (d *ORMS) DeleteMany(ctx context.Context, field string, value any) *ae.Error {
	table, column, e := d.tableAndColumn(field)
	if e != nil {
		return e
	}
	return d.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s=$1", table, column), value)
}

func (d *ORMS) DeleteOne(ctx context.Context, field string, value any) *ae.Error {
	table, column, e := d.tableAndColumn(field)
	if e != nil {
		return e
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE (tableoid,ctid) IN (SELECT tableoid,ctid FROM %s WHERE %s=$1 LIMIT 1)", table, table, column)
	return d.db.Exec(ctx, query, value)
}

func (d *ORMS) DeletePK(ctx context.Context, id any) *ae.Error {
	primary, e := d.t.Indexes().PrimaryKey()
	if e != nil {
		return e
	}
	return d.DeleteOne(ctx, primary, id)
}

func (d *ORMS) ExistsOne(ctx context.Context, field string, value any) *ae.Error {
	table, column, e := d.tableAndColumn(field)
	if e != nil {
		return e
	}
	var exists bool
	e = d.db.ScanAny(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s=$1)", table, column), value, &exists)
	if e != nil {
		return e
	}
	if exists {
		return nil
	}
	return ae.ErrorNotFound
}

func (d *ORMS) ExistsPK(ctx context.Context, id any) *ae.Error {
	primary, e := d.t.Indexes().PrimaryKey()
	if e != nil {
		return e
	}
	return d.ExistsOne(ctx, primary, id)
}

func (d *ORMS) AlterMany(ctx context.Context, field string, value any, data map[string]any) *ae.Error {
	return d.alter(ctx, field, value, data, false)
}

func (d *ORMS) AlterOne(ctx context.Context, field string, value any, data map[string]any) *ae.Error {
	return d.alter(ctx, field, value, data, true)
}

func (d *ORMS) Alter(ctx context.Context, id any, data map[string]any) *ae.Error {
	primary, e := d.t.Indexes().PrimaryKey()
	if e != nil {
		return e
	}
	return d.AlterOne(ctx, primary, id, data)
}

func (d *ORMS) alter(ctx context.Context, field string, value any, data map[string]any, one bool) *ae.Error {
	if len(data) == 0 {
		return ae.ErrorInputTooShort
	}
	table, column, e := d.tableAndColumn(field)
	if e != nil {
		return e
	}
	set, args, e := asql.AssignmentClause(&asql.PostgresDialect, data)
	if e != nil {
		return e
	}
	args = append(args, value)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s=$%d", table, set, column, len(args))
	if one {
		query = fmt.Sprintf("UPDATE %s SET %s WHERE (tableoid,ctid) IN (SELECT tableoid,ctid FROM %s WHERE %s=$%d LIMIT 1)", table, set, table, column, len(args))
	}
	return d.db.Exec(ctx, query, args...)
}

func (d *ORMS) Find(ctx context.Context, id any, dst map[string]any) *ae.Error {
	if len(dst) == 0 {
		return ae.ErrorInputTooShort
	}
	primary, e := d.t.Indexes().PrimaryKey()
	if e != nil {
		return e
	}
	table, e := quoteIdent(d.t.Table())
	if e != nil {
		return e
	}
	primaryField, e := quoteColumnIdentifier(primary)
	if e != nil {
		return e
	}
	fields, dest, e := asql.SelectColumns(&asql.PostgresDialect, dst)
	if e != nil {
		return e
	}
	return d.db.ScanArgs(ctx, fmt.Sprintf("SELECT %s FROM %s WHERE %s=$1", fields, table, primaryField), []any{id}, dest...)
}

func (d *ORMS) tableAndColumn(field string) (string, string, *ae.Error) {
	table, e := quoteIdent(d.t.Table())
	if e != nil {
		return "", "", e
	}
	column, e := quoteColumnIdentifier(field)
	return table, column, e
}
