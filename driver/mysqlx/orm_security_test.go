package mysqlx_test

import (
	"context"
	"testing"

	"github.com/aarioai/aago/driver/index"
	"github.com/aarioai/aago/driver/mysqlx"
)

type securityEntity struct {
	table   string
	indexes index.Indexes
}

func (e securityEntity) Table() string {
	return e.table
}

func (e securityEntity) Indexes() index.Indexes {
	return e.indexes
}

func TestORMRejectsInvalidIdentifiers(t *testing.T) {
	ctx := context.Background()
	validEntity := securityEntity{
		table:   "users",
		indexes: index.NewIndexes(index.Primary("id")),
	}
	orm := mysqlx.ORM(nil, validEntity)

	if e := orm.DeleteMany(ctx, "id` OR 1=1 --", 1); e == nil {
		t.Fatal("DeleteMany accepted invalid field")
	}
	if e := orm.ExistsOne(ctx, "id; DROP TABLE users", 1); e == nil {
		t.Fatal("ExistsOne accepted invalid field")
	}
	if e := orm.AlterMany(ctx, "id", 1, map[string]any{"name`=1 --": "x"}); e == nil {
		t.Fatal("AlterMany accepted invalid data field")
	}
	if e := orm.Find(ctx, 1, map[string]any{"name` FROM users --": new(string)}); e == nil {
		t.Fatal("Find accepted invalid destination field")
	}

	badTable := mysqlx.ORM(nil, securityEntity{
		table:   "users` WHERE 1=1 --",
		indexes: index.NewIndexes(index.Primary("id")),
	})
	if e := badTable.DeleteOne(ctx, "id", 1); e == nil {
		t.Fatal("DeleteOne accepted invalid table")
	}
}
