package postgresx

import (
	"github.com/aarioai/aago/driver/asql"
)

// ConflictIncrement describes a bounded increment for an
// ON CONFLICT DO UPDATE clause.
type ConflictIncrement struct {
	Field string
	Min   string
	Max   string
}

// ConflictInt8Increment creates an int8 conflict increment.
func ConflictInt8Increment(field string) ConflictIncrement {
	return ConflictIncrement{Field: field, Min: "0", Max: asql.MaxInt8Decimal}
}

// ConflictInt16Increment creates an int16 conflict increment.
func ConflictInt16Increment(field string) ConflictIncrement {
	return ConflictIncrement{Field: field, Min: "0", Max: asql.MaxInt16Decimal}
}

// ConflictInt32Increment creates an int32 conflict increment.
func ConflictInt32Increment(field string) ConflictIncrement {
	return ConflictIncrement{Field: field, Min: "0", Max: asql.MaxInt32Decimal}
}

// ConflictInt64Increment creates an int64 conflict increment.
func ConflictInt64Increment(field string) ConflictIncrement {
	return ConflictIncrement{Field: field, Min: "0", Max: asql.MaxInt64Decimal}
}

// SafeConflictIncrementClause renders the clamped assignments of an
// ON CONFLICT DO UPDATE clause. It returns an empty string for an invalid
// table, column or bound.
func SafeConflictIncrementClause(table string, increments []ConflictIncrement) string {
	quotedTable, e := asql.PostgresDialect.Identifier(table)
	if e != nil {
		return ""
	}
	converted := make([]asql.Increment, len(increments))
	for i, increment := range increments {
		field, e := asql.PostgresDialect.Column(increment.Field)
		if e != nil || !asql.IsDecimal(increment.Min) || !asql.IsDecimal(increment.Max) {
			return ""
		}
		converted[i] = asql.Increment{
			Field:    field,
			Current:  quotedTable + "." + field,
			Incoming: `"excluded".` + field,
			Min:      increment.Min,
			Max:      increment.Max,
		}
	}
	return asql.ClampIncrementClause(converted)
}

// SafeIncrementExpression returns a clamped increment expression. It returns an
// empty string for an invalid column or bound.
func SafeIncrementExpression(field string, delta int, maxValue string) string {
	quotedField, e := asql.PostgresDialect.Column(field)
	if e != nil || !asql.IsDecimal(maxValue) {
		return ""
	}
	return asql.ClampIncrement(quotedField, delta, maxValue)
}

// SafeIncrementInt8Expression returns a clamped int8 increment.
func SafeIncrementInt8Expression(field string, delta int) string {
	return SafeIncrementExpression(field, delta, asql.MaxInt8Decimal)
}

// SafeIncrementInt16Expression returns a clamped int16 increment.
func SafeIncrementInt16Expression(field string, delta int) string {
	return SafeIncrementExpression(field, delta, asql.MaxInt16Decimal)
}

// SafeIncrementInt32Expression returns a clamped int32 increment.
func SafeIncrementInt32Expression(field string, delta int) string {
	return SafeIncrementExpression(field, delta, asql.MaxInt32Decimal)
}

// SafeIncrementInt64Expression returns a clamped int64 increment.
func SafeIncrementInt64Expression(field string, delta int) string {
	return SafeIncrementExpression(field, delta, asql.MaxInt64Decimal)
}

// FormatNonNegativeInt formats an integer as a non-negative decimal string.
func FormatNonNegativeInt(value int) string {
	return asql.FormatNonNegativeInt(value)
}
