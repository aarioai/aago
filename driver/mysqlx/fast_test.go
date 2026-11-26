package mysqlx_test

import (
	"testing"

	"github.com/aarioai/aago/driver/mysqlx"
)

func TestInsertMany(t *testing.T) {
	wants := map[int]string{
		1: "INSERT INTO `user` (`nickname`,`avatar`,`created_at`,`updated_at`) VALUES (?, ?, now(), now())",
		2: "INSERT INTO `user` (`nickname`,`avatar`,`created_at`,`updated_at`) VALUES (?, ?, now(), now()),(?, ?, now(), now())",
	}

	for n, want := range wants {
		qs, e := mysqlx.InsertMany("user", "(nickname, avatar, created_at, updated_at)", "(?, ?, now(), now())", n)
		if e != nil {
			t.Fatalf("InsertMany error: %v", e)
		}
		if qs != want {
			t.Errorf("num=%d, qs = %q, want %q", n, qs, want)
		}
	}

	for _, n := range []int{0, -1} {
		if qs, e := mysqlx.InsertMany("user", "(a)", "(?)", n); e != nil || qs != "" {
			t.Errorf("num=%d, qs = %q, want empty", n, qs)
		}
	}
}

func TestInsertManyRejectsSQLFragments(t *testing.T) {
	cases := [][3]string{
		{"users` VALUES ('x'); DROP TABLE users --", "(name)", "(?)"},
		{"users", "(name) VALUES (1); DROP TABLE users --", "(?)"},
		{"users", "(name)", "(?) ON DUPLICATE KEY UPDATE name=1"},
	}
	for _, tc := range cases {
		if _, e := mysqlx.InsertMany(tc[0], tc[1], tc[2], 1); e == nil {
			t.Errorf("InsertMany(%q, %q, %q) error = nil", tc[0], tc[1], tc[2])
		}
	}
}

func TestIn(t *testing.T) {
	if got := mysqlx.In("id", map[uint64]struct{}{}); got != "1!=1" {
		t.Errorf("In(empty) = %q", got)
	}
	if got := mysqlx.In("id", map[uint64]struct{}{7: {}}); got != "`id`=7" {
		t.Errorf("In(one) = %q", got)
	}
	if got := mysqlx.In("id", map[uint64]struct{}{7: {}, 9: {}}); got != "`id` IN (7,9)" && got != "`id` IN (9,7)" {
		t.Errorf("In(two) = %q", got)
	}
}

func TestInValuesUsesPlaceholders(t *testing.T) {
	injected := `x" OR 1=1 -- `
	qs, args, e := mysqlx.InValues("code", map[string]struct{}{injected: {}, "safe": {}})
	if e != nil {
		t.Fatalf("InValues error: %v", e)
	}
	if qs != "`code` IN (?,?)" {
		t.Errorf("InValues SQL = %q", qs)
	}
	if len(args) != 2 || (args[0] != injected && args[1] != injected) {
		t.Errorf("InValues args = %#v", args)
	}
}

func TestInValuesRejectsInvalidField(t *testing.T) {
	if _, _, e := mysqlx.InValues("code` OR 1=1 --", map[string]struct{}{"x": {}}); e == nil {
		t.Fatal("InValues accepted invalid field")
	}
}

func TestInRejectsInvalidField(t *testing.T) {
	if got := mysqlx.In("id) OR 1=1 --", map[uint64]struct{}{7: {}}); got != "1!=1" {
		t.Errorf("In(invalid field) = %q, want fail-closed", got)
	}
	if got := mysqlx.InUint("id) OR 1=1 --", map[uint]struct{}{7: {}}); got != "1!=1" {
		t.Errorf("InUint(invalid field) = %q, want fail-closed", got)
	}
}

func TestArgPairsRejectsInvalidField(t *testing.T) {
	sql, args, ok := mysqlx.ArgPairs(nil, []mysqlx.ArgStmt{
		{Field: "name`=1 --", Value: "x", Valid: true},
	})
	if ok || sql != "" || args != nil {
		t.Errorf("ArgPairs with invalid field = %q, %#v, %v; want empty, nil, false", sql, args, ok)
	}
}
