package asql

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/pkg/types"
)

// MarkerKind selects how bind markers are spelled in rendered SQL.
type MarkerKind uint8

const (
	// QuestionMark renders "?" markers (MySQL).
	QuestionMark MarkerKind = iota
	// DollarMark renders numbered "$1" markers (PostgreSQL).
	DollarMark
)

// Dialect captures the SQL spelling differences between supported databases.
// The builders in this package (Cond, InsertMany, In, ArgPairs, union helpers
// and the struct helpers) keep their logic here so drivers only supply a
// Dialect.
type Dialect struct {
	// Quote is the identifier quote character.
	Quote byte
	// MaxIdentPart is the maximum length of a single identifier part.
	MaxIdentPart int
	// Marker selects the bind marker style.
	Marker MarkerKind
	// Bind renders the 1-based marker of an argument.
	Bind func(index int) string
	// Inline renders a value as an escaped SQL literal and reports whether the
	// value can be inlined at all.
	Inline func(value any) (string, bool)
	// Limit renders the pagination clause.
	Limit func(offset uint, limit uint16) string
	// Now is the current-timestamp expression accepted in value patterns.
	Now string
}

var (
	// MySQLDialect targets MySQL 8.0+.
	MySQLDialect = Dialect{
		Quote:        '`',
		MaxIdentPart: 64,
		Marker:       QuestionMark,
		Bind:         func(int) string { return "?" },
		Inline:       inlineMySQLValue,
		Limit: func(offset uint, limit uint16) string {
			return "LIMIT " + types.FormatUint(offset) + "," + types.FormatUint(uint(limit))
		},
		Now: "now()",
	}

	// PostgresDialect targets PostgreSQL 12+.
	PostgresDialect = Dialect{
		Quote:        '"',
		MaxIdentPart: 63,
		Marker:       DollarMark,
		Bind:         func(index int) string { return "$" + strconv.Itoa(index) },
		Inline:       inlinePostgresValue,
		Limit: func(offset uint, limit uint16) string {
			return "LIMIT " + types.FormatUint(uint(limit)) + " OFFSET " + types.FormatUint(offset)
		},
		Now: "CURRENT_TIMESTAMP",
	}
)

// Identifier validates and quotes a possibly qualified identifier.
func (d *Dialect) Identifier(name string) (string, *ae.Error) {
	if d == nil {
		return "", badSqlError("missing SQL dialect")
	}
	return QuoteIdentifier(name, d.Quote, d.MaxIdentPart)
}

// Column validates and quotes a single, unqualified column identifier.
func (d *Dialect) Column(name string) (string, *ae.Error) {
	if strings.Contains(name, ".") {
		return "", badSqlError("invalid column name: " + name)
	}
	return d.Identifier(name)
}

// Placeholders renders n bind markers, e.g. "?" or "$1,$2".
func (d *Dialect) Placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	if d.Marker == QuestionMark {
		return "?" + strings.Repeat(",?", n-1)
	}
	var b strings.Builder
	b.Grow(n * 4)
	for i := 1; i <= n; i++ {
		if i > 1 {
			b.WriteByte(',')
		}
		b.WriteString(d.Bind(i))
	}
	return b.String()
}

// InlineArgs rewrites the bind markers of a rendered fragment into escaped
// literals. offset is the number of markers rendered before the fragment, so a
// numbered marker "$n" consumes args[n-offset-1]. It reports false when a value
// cannot be inlined safely.
func (d *Dialect) InlineArgs(fragment string, args []any, offset int) (string, bool) {
	if len(args) == 0 {
		return fragment, true
	}
	if d == nil || d.Inline == nil {
		return "", false
	}
	var b strings.Builder
	b.Grow(len(fragment) + 2*len(args))
	next := 0
	for i := 0; i < len(fragment); {
		if d.Marker == DollarMark && fragment[i] == '$' {
			j := i + 1
			for j < len(fragment) && fragment[j] >= '0' && fragment[j] <= '9' {
				j++
			}
			if index, ok := parseIndex(fragment[i+1 : j]); ok && index > offset && index-offset <= len(args) {
				literal, ok := d.Inline(args[index-offset-1])
				if !ok {
					return "", false
				}
				b.WriteString(literal)
				i = j
				continue
			}
		}
		if d.Marker == QuestionMark && fragment[i] == '?' && next < len(args) {
			literal, ok := d.Inline(args[next])
			if !ok {
				return "", false
			}
			next++
			b.WriteString(literal)
			i++
			continue
		}
		b.WriteByte(fragment[i])
		i++
	}
	return b.String(), true
}

func parseIndex(value string) (int, bool) {
	index := 0
	for _, r := range value {
		index = index*10 + int(r-'0')
		if index > 1<<30 {
			return 0, false
		}
	}
	return index, true
}

var mysqlLiteralEscaper = strings.NewReplacer(`\`, `\\`, `'`, `''`)

func inlineMySQLValue(value any) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "NULL", true
	case string:
		return "'" + mysqlLiteralEscaper.Replace(v) + "'", true
	case []byte:
		return "'" + mysqlLiteralEscaper.Replace(string(v)) + "'", true
	case bool:
		if v {
			return "TRUE", true
		}
		return "FALSE", true
	}
	return inlineNumber(value)
}

func inlinePostgresValue(value any) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "NULL", true
	case string:
		return escapePostgresString(v)
	case []byte:
		return escapePostgresString(string(v))
	case bool:
		if v {
			return "TRUE", true
		}
		return "FALSE", true
	}
	return inlineNumber(value)
}

func inlineNumber(value any) (string, bool) {
	switch reflect.ValueOf(value).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return fmt.Sprint(value), true
	}
	return "", false
}

func escapePostgresString(value string) (string, bool) {
	if strings.ContainsRune(value, 0) {
		return "", false
	}
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "'", `\'`)
	return "E'" + value + "'", true
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
