package mongodb

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name    string
		args    []any
		want    any
		wantErr bool
	}{
		{name: "empty", want: nil},
		{name: "document", args: []any{bson.M{"age": 20}}, want: bson.M{"age": 20}},
		{name: "field value", args: []any{"age", 20}, want: bson.M{"age": 20}},
		{name: "comparison", args: []any{"age", ">=", 20}, want: bson.M{"age": bson.M{"$gte": 20}}},
		{name: "invalid field", args: []any{100, 20}, wantErr: true},
		{name: "invalid operator", args: []any{"age", 20, 30}, wantErr: true},
		{name: "too many arguments", args: []any{"a", "=", 1, "extra"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFilter(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFilter() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseFilter() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
