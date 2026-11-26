package bson3_test

import (
	"testing"

	"github.com/aarioai/aago/driver/mongodb/bson3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestCCompareOperators(t *testing.T) {
	cases := []struct {
		operator string
		want     string
	}{
		{"=", "$eq"},
		{"!=", "$ne"},
		{">", "$gt"},
		{">=", "$gte"},
		{"<", "$lt"},
		{"<=", "$lte"},
		{"$gte", "$gte"},
	}
	for _, c := range cases {
		got := bson3.C("age", c.operator, 18)
		condition, ok := got["age"].(bson.M)
		if !ok {
			t.Fatalf("C(%q) = %#v, want a nested document", c.operator, got)
		}
		if _, ok := condition[c.want]; !ok {
			t.Errorf("C(%q) = %#v, want operator %q", c.operator, condition, c.want)
		}
		if v := condition[c.want]; v != 18 {
			t.Errorf("C(%q)[%q] = %v, want 18", c.operator, c.want, v)
		}
	}
}

func TestCBitOperatorWithNonIntValue(t *testing.T) {
	// A non integer operand must not panic; it falls back to the raw operator.
	got := bson3.C("type", "&", "10")
	if _, ok := got["type"].(bson.M); !ok {
		t.Fatalf("C(type, &, string) = %#v", got)
	}
}
