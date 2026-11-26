package asql

import (
	"fmt"
	"strings"

	"github.com/aarioai/aago/pkg/types"
)

// InArgs marks the generated IN list inside a union query format.
const InArgs = "(...)"

// IsInArgs reports whether an argument is the InArgs placeholder.
func IsInArgs(value any) bool {
	tag, ok := value.(string)
	return ok && tag == InArgs
}

// ArgsKey identifies the argument list of one union branch. Branch arguments
// that differ only in ID are merged into a single branch with a longer IN list.
func ArgsKey(args []any) string {
	var b strings.Builder
	for _, arg := range args {
		if IsInArgs(arg) {
			continue
		}
		fmt.Fprintf(&b, "%T:%v;", arg, arg)
	}
	return b.String()
}

// UnionGroup is one UNION ALL branch: its arguments and the IDs it covers.
type UnionGroup struct {
	Args []any
	IDs  []uint64
}

// GroupUnionArgs groups ids by their arguments while preserving input order.
func GroupUnionArgs(ids []uint64, argsFor func(uint64) []any) []UnionGroup {
	groups := make([]UnionGroup, 0, len(ids))
	index := make(map[string]int, len(ids))
	for _, id := range ids {
		args := argsFor(id)
		key := ArgsKey(args)
		if position, ok := index[key]; ok {
			groups[position].IDs = append(groups[position].IDs, id)
			continue
		}
		index[key] = len(groups)
		groups = append(groups, UnionGroup{Args: args, IDs: []uint64{id}})
	}
	return groups
}

// GroupPartitionUnionArgs groups every (partition, id) pair by its arguments.
func GroupPartitionUnionArgs(ids []uint64, partitions []string, argsFor func(string, uint64) []any) []UnionGroup {
	groups := make([]UnionGroup, 0, len(ids)*len(partitions))
	index := make(map[string]int, len(ids)*len(partitions))
	for _, id := range ids {
		for _, partition := range partitions {
			args := argsFor(partition, id)
			key := ArgsKey(args)
			if position, ok := index[key]; ok {
				groups[position].IDs = append(groups[position].IDs, id)
				continue
			}
			index[key] = len(groups)
			groups = append(groups, UnionGroup{Args: args, IDs: []uint64{id}})
		}
	}
	return groups
}

// UnionAllQuery joins the formatted branches with UNION ALL and appends
// "LIMIT <number of ids>". InArgs is replaced by the branch IN list.
func UnionAllQuery(format string, ids []uint64, groups []UnionGroup) string {
	queries := make([]string, 0, len(groups))
	for _, group := range groups {
		args := make([]any, len(group.Args))
		copy(args, group.Args)
		idList := "(" + joinUint64s(group.IDs) + ")"
		for i, arg := range args {
			if IsInArgs(arg) {
				args[i] = idList
			}
		}
		queries = append(queries, fmt.Sprintf(format, args...))
	}
	return strings.Join(queries, " UNION ALL ") + " LIMIT " + types.FormatUint(uint(len(ids)))
}

// UnionAllQs joins one query per partition with UNION ALL.
func UnionAllQs(format string, partitions []string, argsFor func(string) []any) string {
	var b strings.Builder
	for i, partition := range partitions {
		if i > 0 {
			b.WriteString(" UNION ALL ")
		}
		b.WriteString(fmt.Sprintf(format, argsFor(partition)...))
	}
	return b.String()
}

// UnionInUints returns the distinct tables and the comma separated id list.
func UnionInUints(ids []uint, tableForID func(uint) string) ([]string, string) {
	return unionInIDs(ids, tableForID)
}

// UnionInUint64s returns the distinct tables and the comma separated id list.
func UnionInUint64s(ids []uint64, tableForID func(uint64) string) ([]string, string) {
	return unionInIDs(ids, tableForID)
}

func unionInIDs[T uint | uint64](ids []T, tableForID func(T) string) ([]string, string) {
	seen := make(map[string]struct{}, len(ids))
	tables := make([]string, 0, len(ids))
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, types.FormatUint(uint64(id)))
		table := tableForID(id)
		if _, ok := seen[table]; ok {
			continue
		}
		seen[table] = struct{}{}
		tables = append(tables, table)
	}
	return tables, strings.Join(values, ",")
}

func joinUint64s(ids []uint64) string {
	var b strings.Builder
	b.Grow(len(ids) * (types.MaxUint64Len + 1))
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(types.FormatUint(id))
	}
	return b.String()
}
