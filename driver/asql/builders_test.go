package asql_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/aarioai/aago/aa/atype"
	"github.com/aarioai/aago/driver/asql"
)

func TestCondRendersPerDialect(t *testing.T) {
	mysql := asql.NewCond(&asql.MySQLDialect, atype.Paging{Offset: 5, Limit: 20})
	mysql.And("t.id", "100").Or("t.name", ":@Ada@").OrderBy("t.id DESC")
	query, args := mysql.StmtArgs()
	if want := " WHERE `t`.`id`=? OR `t`.`name` LIKE ? ESCAPE '!' ORDER BY `t`.`id` DESC LIMIT 5,20"; query != want {
		t.Errorf("mysql StmtArgs = %q, want %q", query, want)
	}
	if !reflect.DeepEqual(args, []any{"100", "%Ada%"}) {
		t.Errorf("mysql args = %#v", args)
	}
	if want := " WHERE `t`.`id`='100' OR `t`.`name` LIKE '%Ada%' ESCAPE '!' ORDER BY `t`.`id` DESC LIMIT 5,20"; mysql.Stmt() != want {
		t.Errorf("mysql Stmt = %q, want %q", mysql.Stmt(), want)
	}

	postgres := asql.NewCond(&asql.PostgresDialect, atype.Paging{Offset: 5, Limit: 20})
	postgres.And("t.id", "100").Or("t.name", ":@Ada@").OrderBy("t.id DESC")
	query, args = postgres.StmtArgs()
	if want := ` WHERE "t"."id"=$1 OR "t"."name" LIKE $2 ESCAPE '!' ORDER BY "t"."id" DESC LIMIT 20 OFFSET 5`; query != want {
		t.Errorf("postgres StmtArgs = %q, want %q", query, want)
	}
	if !reflect.DeepEqual(args, []any{"100", "%Ada%"}) {
		t.Errorf("postgres args = %#v", args)
	}
	if want := ` WHERE "t"."id"=E'100' OR "t"."name" LIKE E'%Ada%' ESCAPE '!' ORDER BY "t"."id" DESC LIMIT 20 OFFSET 5`; postgres.Stmt() != want {
		t.Errorf("postgres Stmt = %q, want %q", postgres.Stmt(), want)
	}
}

func TestCondFailsClosedWithoutDialect(t *testing.T) {
	var cond asql.Cond
	cond.And("a", "1")
	if cond.Err() == nil {
		t.Fatal("expected missing dialect error")
	}
	if query, args := cond.StmtArgs(); query != " WHERE 1!=1" || len(args) != 0 {
		t.Errorf("StmtArgs = %q %#v", query, args)
	}
}

func TestDialectIdentifierAndMarkers(t *testing.T) {
	for _, dialect := range []*asql.Dialect{&asql.MySQLDialect, &asql.PostgresDialect} {
		if _, e := dialect.Identifier("a.b"); e != nil {
			t.Errorf("Identifier(a.b) error: %v", e)
		}
		if _, e := dialect.Column("a.b"); e == nil {
			t.Error("Column accepted a qualified name")
		}
		if _, e := dialect.Identifier("a`=1"); e == nil {
			t.Error("Identifier accepted an injection attempt")
		}
	}
	if got := asql.MySQLDialect.Placeholders(2); got != "?,?" {
		t.Errorf("mysql placeholders = %q", got)
	}
	if got := asql.PostgresDialect.Placeholders(2); got != "$1,$2" {
		t.Errorf("postgres placeholders = %q", got)
	}
	// offset 1 means one marker was already consumed, so $2 is the first arg.
	if literal, ok := asql.PostgresDialect.InlineArgs("a=$2", []any{"x", "y"}, 1); !ok || literal != "a=E'x'" {
		t.Errorf("InlineArgs = %q %v", literal, ok)
	}
	if _, ok := asql.PostgresDialect.InlineArgs("a=$1", []any{"x\x00"}, 0); ok {
		t.Error("InlineArgs accepted a NUL byte")
	}
}

func TestStructConditionBuilders(t *testing.T) {
	user := struct {
		Name string `db:"name"`
		Age  int    `db:"age"`
	}{Name: "O'Reilly", Age: 42}

	columns, e := asql.StructColumns(&asql.PostgresDialect, user, "name", "age")
	if e != nil || columns != `"name", "age"` {
		t.Errorf("StructColumns = %q %v", columns, e)
	}

	condition, args, e := asql.StructConditions(&asql.PostgresDialect, user, "AND", false, "name", "age")
	if e != nil || condition != `"name"=$1 AND "age"=$2` || !reflect.DeepEqual(args, []any{"O'Reilly", 42}) {
		t.Errorf("StructConditions = %q %#v %v", condition, args, e)
	}

	if got := asql.StructConditionsLiteral(&asql.MySQLDialect, user, "AND", false, "name", "age"); got != "`name`='O''Reilly' AND `age`=42" {
		t.Errorf("StructConditionsLiteral = %q", got)
	}
	if got := asql.StructConditionsLiteral(&asql.MySQLDialect, user, "OR", true, "name"); got != "`name` LIKE '%O''Reilly%' ESCAPE '!'" {
		t.Errorf("StructConditionsLiteral(like) = %q", got)
	}
	if got := asql.StructConditionsLiteral(&asql.MySQLDialect, user, "AND", false, "name`=1"); got != "1!=1" {
		t.Errorf("StructConditionsLiteral(invalid) = %q", got)
	}
	if got := asql.WhereClause("`a`=1"); got != " WHERE `a`=1 " {
		t.Errorf("WhereClause = %q", got)
	}
}

func TestInsertBuilders(t *testing.T) {
	if got, e := asql.InsertMany(&asql.MySQLDialect, "user", "(a, b)", "(?, now())", 2); e != nil || got != "INSERT INTO `user` (`a`,`b`) VALUES (?, now()),(?, now())" {
		t.Errorf("mysql InsertMany = %q %v", got, e)
	}
	want := `INSERT INTO "public"."users" ("name","created_at") VALUES ($1, CURRENT_TIMESTAMP),($2, CURRENT_TIMESTAMP)`
	if got, e := asql.InsertMany(&asql.PostgresDialect, "public.users", "(name, created_at)", "(?, now())", 2); e != nil || got != want {
		t.Errorf("postgres InsertMany = %q %v", got, e)
	}
	if got, e := asql.InsertMany(&asql.MySQLDialect, "user", "(a)", "(?)", 0); e != nil || got != "" {
		t.Errorf("InsertMany(0 rows) = %q %v", got, e)
	}
	for _, pattern := range []string{"(?) ON DUPLICATE KEY UPDATE a=1", "(values(1))", "?"} {
		if _, e := asql.InsertMany(&asql.MySQLDialect, "user", "(a)", pattern, 1); e == nil {
			t.Errorf("InsertMany accepted pattern %q", pattern)
		}
	}

	if got := asql.InUint64Set(&asql.MySQLDialect, "id", map[uint64]struct{}{7: {}}); got != "`id`=7" {
		t.Errorf("InUint64Set = %q", got)
	}
	if got := asql.InUintSet(&asql.PostgresDialect, "id", map[uint]struct{}{}); got != "1!=1" {
		t.Errorf("InUintSet(empty) = %q", got)
	}
	query, args, e := asql.InStringSet(&asql.PostgresDialect, "code", map[string]struct{}{"a": {}})
	if e != nil || query != `"code" IN ($1)` || !reflect.DeepEqual(args, []any{"a"}) {
		t.Errorf("InStringSet = %q %#v %v", query, args, e)
	}
}

func TestArgPairsAndORMBuilders(t *testing.T) {
	statements := []asql.ArgStmt{
		{Field: "name", Value: "Ada", Valid: true},
		{Field: "age", Value: 37, Valid: true, Ignores: []string{"age"}},
	}
	set, args, ok := asql.ArgPairs(&asql.PostgresDialect, 19, statements)
	if !ok || set != `"name"=$1` || !reflect.DeepEqual(args, []any{"Ada", 19}) {
		t.Errorf("ArgPairs = %q %#v %v", set, args, ok)
	}
	if set, args, ok := asql.ArgPairs(&asql.MySQLDialect, nil, []asql.ArgStmt{{Field: "name", Valid: false}}); ok || set != "" || args != nil {
		t.Errorf("ArgPairs(no valid) = %q %#v %v", set, args, ok)
	}

	assignment, values, e := asql.AssignmentClause(&asql.MySQLDialect, map[string]any{"name": "Ada"})
	if e != nil || assignment != "`name`=?" || !reflect.DeepEqual(values, []any{"Ada"}) {
		t.Errorf("AssignmentClause = %q %#v %v", assignment, values, e)
	}
	columns, dest, e := asql.SelectColumns(&asql.PostgresDialect, map[string]any{"name": new(string)})
	if e != nil || columns != `"name"` || len(dest) != 1 {
		t.Errorf("SelectColumns = %q %#v %v", columns, dest, e)
	}
}

func TestUnionBuilders(t *testing.T) {
	ids := []uint64{1, 2, 3}
	groups := asql.GroupUnionArgs(ids, func(uint64) []any { return []any{"tb", asql.InArgs} })
	if len(groups) != 1 || !reflect.DeepEqual(groups[0].IDs, ids) {
		t.Fatalf("GroupUnionArgs = %#v", groups)
	}
	query := asql.UnionAllQuery("SELECT * FROM %s WHERE id IN %s", ids, groups)
	if want := "SELECT * FROM tb WHERE id IN (1,2,3) LIMIT 3"; query != want {
		t.Errorf("UnionAllQuery = %q, want %q", query, want)
	}
	if !strings.Contains(asql.InArgs, "(") {
		t.Errorf("InArgs = %q", asql.InArgs)
	}

	partitioned := asql.GroupPartitionUnionArgs(ids[:2], []string{"a", "b"}, func(pte string, id uint64) []any {
		return []any{pte, id}
	})
	if len(partitioned) != 4 {
		t.Errorf("GroupPartitionUnionArgs = %#v", partitioned)
	}

	if got := asql.UnionAllQs("SELECT %s", []string{"1", "2"}, func(p string) []any { return []any{p} }); got != "SELECT 1 UNION ALL SELECT 2" {
		t.Errorf("UnionAllQs = %q", got)
	}
	tables, values := asql.UnionInUint64s([]uint64{1, 2, 3}, func(id uint64) string {
		if id == 1 {
			return "t1"
		}
		return "t2"
	})
	if !reflect.DeepEqual(tables, []string{"t1", "t2"}) || values != "1,2,3" {
		t.Errorf("UnionInUint64s = %#v %q", tables, values)
	}
}

func TestClampIncrementBuilders(t *testing.T) {
	if got := asql.ClampIncrement("count", -3, asql.MaxUint16Decimal); got != "count=CASE WHEN count-3<=0 THEN 0 ELSE count-3 END" {
		t.Errorf("ClampIncrement(-3) = %q", got)
	}
	if got := asql.ClampIncrement("count", 3, asql.MaxUint16Decimal); got != "count=CASE WHEN count+3>=65535 THEN 65535 ELSE count+3 END" {
		t.Errorf("ClampIncrement(3) = %q", got)
	}
	if got := asql.ClampIncrement("count", 0, asql.MaxUint16Decimal); got != "count=count" {
		t.Errorf("ClampIncrement(0) = %q", got)
	}
	if got := asql.ClampIncrement("count", math.MinInt, asql.MaxUint64Decimal); !strings.Contains(got, "count-9223372036854775808<=0") {
		t.Errorf("ClampIncrement(math.MinInt) = %q", got)
	}

	increment := asql.Increment{Field: "count", Current: "tb.count", Incoming: "new_tb.count", Min: "0", Max: asql.MaxUint16Decimal}
	want := "count=CASE WHEN tb.count-0<=-new_tb.count THEN 0 WHEN new_tb.count>=65535-tb.count THEN 65535 ELSE tb.count+new_tb.count END"
	if got := asql.ClampIncrementClause([]asql.Increment{increment}); got != want {
		t.Errorf("ClampIncrementClause = %q, want %q", got, want)
	}
	if got := asql.FormatNonNegativeInt(-2); got != "0" {
		t.Errorf("FormatNonNegativeInt(-2) = %q", got)
	}
	if asql.IsDecimal("65a") || !asql.IsDecimal("65535") {
		t.Error("IsDecimal mismatch")
	}
}
