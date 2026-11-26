package mysqlx

import (
	"github.com/aarioai/aago/driver/asql"
)

// DuplicateKeyIncrement describes a bounded increment for an
// ON DUPLICATE KEY UPDATE clause.
type DuplicateKeyIncrement struct {
	Field string
	Min   string
	Max   string
}

// DuplicateKeyUint8Increment creates an unsigned 8-bit duplicate-key increment.
func DuplicateKeyUint8Increment(field string) DuplicateKeyIncrement {
	return DuplicateKeyIncrement{field, "0", asql.MaxUint8Decimal}
}

// DuplicateKeyUint16Increment creates an unsigned 16-bit duplicate-key increment.
func DuplicateKeyUint16Increment(field string) DuplicateKeyIncrement {
	return DuplicateKeyIncrement{field, "0", asql.MaxUint16Decimal}
}

// DuplicateKeyUint24Increment creates an unsigned 24-bit duplicate-key increment.
func DuplicateKeyUint24Increment(field string) DuplicateKeyIncrement {
	return DuplicateKeyIncrement{field, "0", asql.MaxUint24Decimal}
}

// DuplicateKeyUintIncrement creates an unsigned 32-bit duplicate-key increment.
func DuplicateKeyUintIncrement(field string) DuplicateKeyIncrement {
	return DuplicateKeyIncrement{field, "0", asql.MaxUintDecimal}
}

// DuplicateKeyUint64Increment creates an unsigned 64-bit duplicate-key increment.
func DuplicateKeyUint64Increment(field string) DuplicateKeyIncrement {
	return DuplicateKeyIncrement{field, "0", asql.MaxUint64Decimal}
}

// SafeDuplicateKeyIncrementClause renders the clamped assignments of an
// ON DUPLICATE KEY UPDATE clause.
//
// @warn 如果类型是 unsigned ，一定要使用 INSERT IGNORE ，否则若值为负数，则直接报错，走不到 ON DUPLICATE KEY UPDATE 部分
func SafeDuplicateKeyIncrementClause(table string, increments []DuplicateKeyIncrement) string {
	converted := make([]asql.Increment, len(increments))
	for i, increment := range increments {
		converted[i] = asql.Increment{
			Field:    increment.Field,
			Current:  table + "." + increment.Field,
			Incoming: "new_" + table + "." + increment.Field,
			Min:      increment.Min,
			Max:      increment.Max,
		}
	}
	return asql.ClampIncrementClause(converted)
}

// SafeIncrementExpression returns a SQL expression that clamps an increment
// to zero or maxValue.
func SafeIncrementExpression(field string, delta int, maxValue string) string {
	return asql.ClampIncrement(field, delta, maxValue)
}

// FormatNonNegativeInt formats an integer as a non-negative decimal string.
func FormatNonNegativeInt(value int) string {
	return asql.FormatNonNegativeInt(value)
}

// SafeIncrementUint8Expression returns a clamped unsigned 8-bit increment.
func SafeIncrementUint8Expression(field string, delta int) string {
	return SafeIncrementExpression(field, delta, asql.MaxUint8Decimal)
}

// SafeIncrementUint16Expression returns a clamped unsigned 16-bit increment.
func SafeIncrementUint16Expression(field string, delta int) string {
	return SafeIncrementExpression(field, delta, asql.MaxUint16Decimal)
}

// SafeIncrementUint24Expression returns a clamped unsigned 24-bit increment.
func SafeIncrementUint24Expression(field string, delta int) string {
	return SafeIncrementExpression(field, delta, asql.MaxUint24Decimal)
}

// SafeIncrementUintExpression returns a clamped unsigned 32-bit increment.
func SafeIncrementUintExpression(field string, delta int) string {
	return SafeIncrementExpression(field, delta, asql.MaxUintDecimal)
}

// SafeIncrementUint64Expression returns a clamped unsigned 64-bit increment.
func SafeIncrementUint64Expression(field string, delta int) string {
	return SafeIncrementExpression(field, delta, asql.MaxUint64Decimal)
}
