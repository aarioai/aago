package mysqlx

import (
	"context"
	"database/sql"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver/asql"
)

// InArgs marks the generated IN list inside a union query format.
const InArgs = asql.InArgs

// MaxUint64IdLen is the maximum decimal length of a uint64 id.
const MaxUint64IdLen = 20

// InUnionAllTablesQs queries every (partition table, id) pair and unions the
// results. format and xargs must only produce trusted SQL fragments; ids passed
// through InArgs are encoded as decimal integers. Do not forget to close rows.
func InUnionAllTablesQs(ctx context.Context, db *DB, format string, ids []uint64, ptbs []string, xargs func(string, uint64) []any) (*sql.Rows, *ae.Error) {
	if len(ids) == 0 {
		return nil, ae.ErrorEmptyInput
	}
	groups := asql.GroupPartitionUnionArgs(ids, ptbs, xargs)
	return db.Query(ctx, asql.UnionAllQuery(format, ids, groups))
}

// InUnionAllQs queries one sharded table per id and unions the results.
// Do not forget to close rows.
func InUnionAllQs(ctx context.Context, db *DB, format string, ids []uint64, xargs func(uint64) []any) (*sql.Rows, *ae.Error) {
	if len(ids) == 0 {
		return nil, ae.ErrorEmptyInput
	}
	return db.Query(ctx, asql.UnionAllQuery(format, ids, asql.GroupUnionArgs(ids, xargs)))
}

// InUnionAllTablesQs is the DB method form of InUnionAllTablesQs.
func (d *DB) InUnionAllTablesQs(ctx context.Context, format string, ids []uint64, ptbs []string, xargs func(string, uint64) []any) (*sql.Rows, *ae.Error) {
	return InUnionAllTablesQs(ctx, d, format, ids, ptbs, xargs)
}

// InUnionAllQs is the DB method form of InUnionAllQs.
func (d *DB) InUnionAllQs(ctx context.Context, format string, ids []uint64, xargs func(uint64) []any) (*sql.Rows, *ae.Error) {
	return InUnionAllQs(ctx, d, format, ids, xargs)
}

// UnionAllQs joins one query per partition with UNION ALL.
func UnionAllQs(format string, ptbs []string, xargs func(string) []any) string {
	return asql.UnionAllQs(format, ptbs, xargs)
}

// UnionInUints returns the distinct partition tables and the id list.
func UnionInUints(ids []uint, tableForID func(uint) string) ([]string, string) {
	return asql.UnionInUints(ids, tableForID)
}

// UnionInUint64s returns the distinct partition tables and the id list.
func UnionInUint64s(ids []uint64, tableForID func(uint64) string) ([]string, string) {
	return asql.UnionInUint64s(ids, tableForID)
}
