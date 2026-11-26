package index_test

import (
	"testing"

	"github.com/aarioai/aago/driver/index"
)

type user struct{}

func (user) Table() string {
	return "user"
}

func (user) Indexes() index.Indexes {
	return index.NewIndexes(
		index.Primary("id"),
		index.Unique("email"),
		index.Index("created_at"),
	)
}

type composite struct{}

func (composite) Table() string {
	return "composite"
}

func (composite) Indexes() index.Indexes {
	return index.NewIndexes(index.Primary("a", "b"))
}

type emptyIndexes struct{}

func (emptyIndexes) Table() string {
	return "empty"
}

func (emptyIndexes) Indexes() index.Indexes {
	return index.Indexes{}
}

func TestIndexes(t *testing.T) {
	idx := user{}.Indexes()

	if pk, e := idx.PrimaryKey(); e != nil || pk != "id" {
		t.Errorf("PrimaryKey() = %q, %v", pk, e)
	}
	uniques := idx.List(index.KeyUnique)
	if len(uniques) != 1 || len(uniques["u_email"]) != 1 || uniques["u_email"][0] != "email" {
		t.Errorf("List(KeyUnique) = %#v", uniques)
	}
	if idx.List(index.KeyText) != nil {
		t.Error("List(KeyText) should be nil when no text index exists")
	}
}

func TestPrimaryKeyErrors(t *testing.T) {
	if _, e := (index.Indexes{}).PrimaryKey(); e == nil {
		t.Error("missing primary key should return an error")
	}
	if _, e := (composite{}).Indexes().PrimaryKey(); e == nil {
		t.Error("a composite primary key should return an error")
	}
}

func TestToMongoDBIndexModels(t *testing.T) {
	models := index.ToMongoDBIndexModels(user{})
	if len(models) != 3 {
		t.Fatalf("len(models) = %d, want 3", len(models))
	}
	if index.ToMongoDBIndexModels(emptyIndexes{}) != nil {
		t.Error("empty indexes should produce no models")
	}
}
