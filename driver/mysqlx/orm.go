package mysqlx

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

func ORM(db *DB, t index.Entity) *ORMS {
	return &ORMS{db, t}
}

func (d *DB) ORM(t index.Entity) *ORMS {
	return &ORMS{d, t}
}

func (d *ORMS) DeleteMany(ctx context.Context, field string, value any) *ae.Error {
	table, column, e := d.tableAndColumn(field)
	if e != nil {
		return e
	}
	return d.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s=?", table, column), value)
}

func (d *ORMS) DeleteOne(ctx context.Context, field string, value any) *ae.Error {
	table, column, e := d.tableAndColumn(field)
	if e != nil {
		return e
	}
	return d.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s=? LIMIT 1", table, column), value)
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
	qs := fmt.Sprintf("SELECT 1 FROM %s WHERE %s=? LIMIT 1", table, column)
	var newId uint8
	if e = d.db.ScanAny(ctx, qs, value, &newId); e != nil {
		return e
	}
	if newId == 1 {
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
	set, args, e := asql.AssignmentClause(&asql.MySQLDialect, data)
	if e != nil {
		return e
	}
	args = append(args, value)
	qs := fmt.Sprintf("UPDATE %s SET %s WHERE %s=?", table, set, column)
	if one {
		qs += " LIMIT 1"
	}
	return d.db.Exec(ctx, qs, args...)
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
	fields, dest, e := asql.SelectColumns(&asql.MySQLDialect, dst)
	if e != nil {
		return e
	}
	qs := fmt.Sprintf("SELECT %s FROM %s WHERE %s=?", fields, table, primaryField)
	return d.db.ScanAny(ctx, qs, id, dest...)
}

func (d *ORMS) tableAndColumn(field string) (string, string, *ae.Error) {
	table, e := quoteIdent(d.t.Table())
	if e != nil {
		return "", "", e
	}
	column, e := quoteColumnIdentifier(field)
	return table, column, e
}
