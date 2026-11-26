package mysqlx_test

import (
	"testing"

	"github.com/aarioai/aago/driver/mysqlx"
)

func TestSafeIncrementExpressions(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "negative uint16",
			got:  mysqlx.SafeIncrementUint16Expression("count", -3),
			want: "count=CASE WHEN count-3<=0 THEN 0 ELSE count-3 END",
		},
		{
			name: "positive uint16",
			got:  mysqlx.SafeIncrementUint16Expression("count", 3),
			want: "count=CASE WHEN count+3>=65535 THEN 65535 ELSE count+3 END",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestSafeDuplicateKeyIncrementClause(t *testing.T) {
	increment := mysqlx.DuplicateKeyUint16Increment("count")
	got := mysqlx.SafeDuplicateKeyIncrementClause("tb", []mysqlx.DuplicateKeyIncrement{increment})
	want := "count=CASE WHEN tb.count-0<=-new_tb.count THEN 0 WHEN new_tb.count>=65535-tb.count THEN 65535 ELSE tb.count+new_tb.count END"
	if got != want {
		t.Errorf("clause = %q, want %q", got, want)
	}
}

func TestFormatNonNegativeInt(t *testing.T) {
	for _, tt := range []struct {
		input int
		want  string
	}{
		{-1, "0"},
		{0, "0"},
		{12, "12"},
	} {
		if got := mysqlx.FormatNonNegativeInt(tt.input); got != tt.want {
			t.Errorf("FormatNonNegativeInt(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
