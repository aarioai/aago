package bson2_test

import (
	"testing"

	"github.com/aarioai/aago/driver/mongodb/bson2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestUnset(t *testing.T) {
	got := bson2.Unset("a", "b")
	if len(got) != 1 || got[0].Key != "$unset" {
		t.Fatalf("Unset() = %#v", got)
	}
	fields, ok := got[0].Value.(bson.D)
	if !ok {
		t.Fatalf("$unset value = %#v, want bson.D", got[0].Value)
	}
	if len(fields) != 2 || fields[0].Key != "a" || fields[1].Key != "b" {
		t.Errorf("$unset fields = %#v", fields)
	}
	for _, f := range fields {
		if v, ok := f.Value.(string); !ok || v != "" {
			t.Errorf("$unset %q = %#v, want empty string", f.Key, f.Value)
		}
	}
}

func TestSetOnInsert(t *testing.T) {
	got := bson2.SetOnInsert(bson.M{"a": 1})
	if _, ok := got["$setOnInsert"]; !ok {
		t.Errorf("SetOnInsert() = %#v, want $setOnInsert", got)
	}
}
