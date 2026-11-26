package postgresx_test

import (
	"context"
	"testing"

	"github.com/aarioai/aago/driver/index"
	"github.com/aarioai/aago/driver/postgresx"
)

type pgEntity struct {
	table string
}

func (e pgEntity) Table() string { return e.table }
func (e pgEntity) Indexes() index.Indexes {
	return index.NewIndexes(index.Primary("id"))
}

func TestORMRejectsInvalidIdentifiers(t *testing.T) {
	orm := postgresx.ORM(nil, pgEntity{table: "users"})
	if e := orm.DeleteOne(context.Background(), `id" OR TRUE --`, 1); e == nil {
		t.Fatal("DeleteOne accepted invalid identifier")
	}
	if e := orm.AlterMany(context.Background(), "id", 1, map[string]any{`name"=NULL --`: "x"}); e == nil {
		t.Fatal("AlterMany accepted invalid identifier")
	}
	if e := orm.Find(context.Background(), 1, map[string]any{`name" FROM users --`: new(string)}); e == nil {
		t.Fatal("Find accepted invalid identifier")
	}
	badTable := postgresx.ORM(nil, pgEntity{table: `users" WHERE TRUE --`})
	if e := badTable.DeleteMany(context.Background(), "id", 1); e == nil {
		t.Fatal("DeleteMany accepted invalid table")
	}
}
