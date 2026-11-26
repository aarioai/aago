package postgresx

import (
	"context"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver/asql"
	"github.com/jackc/pgx/v5"
)

// InArgs marks the generated IN list inside a union query format.
const InArgs = asql.InArgs

// MaxUint64IDLen is the maximum decimal length of a uint64 id.
const MaxUint64IDLen = 20

// InUnionAllTablesQs queries every (partition, id) pair and unions the results.
// format and argsFor must only produce trusted SQL fragments; ids passed through
// InArgs are encoded as decimal integers. Do not forget to close rows.
func InUnionAllTablesQs(ctx context.Context, db *DB, format string, ids []uint64, partitions []string, argsFor func(string, uint64) []any) (pgx.Rows, *ae.Error) {
	if len(ids) == 0 {
		return nil, ae.ErrorEmptyInput
	}
	groups := asql.GroupPartitionUnionArgs(ids, partitions, argsFor)
	return db.Query(ctx, asql.UnionAllQuery(format, ids, groups))
}

// InUnionAllQs queries one partition per id and unions the results.
// Do not forget to close rows.
func InUnionAllQs(ctx context.Context, db *DB, format string, ids []uint64, argsFor func(uint64) []any) (pgx.Rows, *ae.Error) {
	if len(ids) == 0 {
		return nil, ae.ErrorEmptyInput
	}
	return db.Query(ctx, asql.UnionAllQuery(format, ids, asql.GroupUnionArgs(ids, argsFor)))
}

// InUnionAllTablesQs is the DB method form of InUnionAllTablesQs.
func (d *DB) InUnionAllTablesQs(ctx context.Context, format string, ids []uint64, partitions []string, argsFor func(string, uint64) []any) (pgx.Rows, *ae.Error) {
	return InUnionAllTablesQs(ctx, d, format, ids, partitions, argsFor)
}

// InUnionAllQs is the DB method form of InUnionAllQs.
func (d *DB) InUnionAllQs(ctx context.Context, format string, ids []uint64, argsFor func(uint64) []any) (pgx.Rows, *ae.Error) {
	return InUnionAllQs(ctx, d, format, ids, argsFor)
}

// UnionAllQs joins one query per partition with UNION ALL.
func UnionAllQs(format string, partitions []string, argsFor func(string) []any) string {
	return asql.UnionAllQs(format, partitions, argsFor)
}

// UnionInUints returns the distinct partition tables and the id list.
func UnionInUints(ids []uint, tableForID func(uint) string) ([]string, string) {
	return asql.UnionInUints(ids, tableForID)
}

// UnionInUint64s returns the distinct partition tables and the id list.
func UnionInUint64s(ids []uint64, tableForID func(uint64) string) ([]string, string) {
	return asql.UnionInUint64s(ids, tableForID)
}
