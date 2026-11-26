package asql

import (
	"strings"

	"github.com/aarioai/aago/pkg/types"
)

const (
	MaxUint8Decimal  = "255"
	MaxUint16Decimal = "65535"
	MaxUint24Decimal = "16777215"
	MaxUintDecimal   = "4294967295"
	MaxUint64Decimal = "18446744073709551615"

	MaxInt8Decimal  = "127"
	MaxInt16Decimal = "32767"
	MaxInt32Decimal = "2147483647"
	MaxInt64Decimal = "9223372036854775807"
)

// ClampIncrement builds "field=CASE WHEN field+delta>=max THEN max …" so an
// increment can never underflow below zero or overflow above maxValue.
func ClampIncrement(field string, delta int, maxValue string) string {
	if delta == 0 {
		return field + "=" + field
	}
	var expression string
	if delta > 0 {
		expression = field + "+" + types.FormatInt(delta)
		return field + "=CASE WHEN " + expression + ">=" + maxValue +
			" THEN " + maxValue + " ELSE " + expression + " END"
	}
	expression = field + "-" + types.FormatUint(uint64(-(delta+1))+1)
	return field + "=CASE WHEN " + expression + "<=0 THEN 0 ELSE " + expression + " END"
}

// Increment is one clamped upsert increment. Current and Incoming are the
// dialect specific references to the stored and the conflicting row.
type Increment struct {
	Field    string // assignment target
	Current  string // e.g. tb.count
	Incoming string // e.g. new_tb.count or "excluded"."count"
	Min      string
	Max      string
}

// ClampIncrementClause builds the comma separated "field=CASE …" list used by
// an upsert clause.
func ClampIncrementClause(increments []Increment) string {
	var b strings.Builder
	for i, increment := range increments {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(increment.Field)
		b.WriteString("=CASE WHEN ")
		b.WriteString(increment.Current)
		b.WriteByte('-')
		b.WriteString(increment.Min)
		b.WriteString("<=-")
		b.WriteString(increment.Incoming)
		b.WriteString(" THEN ")
		b.WriteString(increment.Min)
		b.WriteString(" WHEN ")
		b.WriteString(increment.Incoming)
		b.WriteString(">=")
		b.WriteString(increment.Max)
		b.WriteByte('-')
		b.WriteString(increment.Current)
		b.WriteString(" THEN ")
		b.WriteString(increment.Max)
		b.WriteString(" ELSE ")
		b.WriteString(increment.Current)
		b.WriteByte('+')
		b.WriteString(increment.Incoming)
		b.WriteString(" END")
	}
	return b.String()
}

// FormatNonNegativeInt formats an integer as a non-negative decimal string.
func FormatNonNegativeInt(value int) string {
	if value <= 0 {
		return "0"
	}
	return types.FormatInt(value)
}

// IsDecimal reports whether value is a plain non-negative decimal literal.
func IsDecimal(value string) bool {
	return isDecimal(value)
}
