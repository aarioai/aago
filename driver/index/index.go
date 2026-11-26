package index

import (
	"slices"
	"strings"

	"github.com/aarioai/aago/aa/ae"
)

var (
	ErrMissingPKDefinition   = ae.New(ae.VariantAlsoNegotiates, "missing primary key definition").Lock()
	ErrMultiplePKDefinitions = ae.New(ae.VariantAlsoNegotiates, "can have only one primary key. found multiple definitions").Lock()
)

type Type uint8
type Column struct {
	Field     string
	Asc       bool // default is DESC
	Invisible bool
	Type      Type
	Language  string
}
type Indexes map[string][]Column
type Entity interface {
	Table() string
	Indexes() Indexes
}

const (
	KeyPrimaryName = "PRIMARY"
)

const (
	KeyPrimary Type = iota
	KeyUnique
	KeyIndex
	KeyText // MySQL fulltext, Mongodb text
	KeySpatial

	KeySpatial2D
	KeySpatial2DSphere
	KeyHashed
)

func index(t Type, fields ...string) []Column {
	if len(fields) == 1 {
		return []Column{{Type: t, Field: fields[0]}}
	}
	indexes := make([]Column, 0, len(fields))
	for _, field := range fields {
		indexes = append(indexes, Column{Type: t, Field: field})
	}
	return indexes
}

func parseFieldNames(columns []Column) []string {
	fields := make([]string, len(columns))
	for i, column := range columns {
		fields[i] = column.Field
	}
	return fields
}

func makeIndexName(t Type, columns []Column) string {
	var s strings.Builder
	switch t {
	case KeyPrimary:
		return KeyPrimaryName
	case KeyUnique:
		s.WriteString("u")
	case KeyIndex:
		s.WriteString("i")
	case KeyText:
		s.WriteString("t")
	case KeySpatial:
		s.WriteString("s")
	case KeySpatial2D:
		s.WriteString("d")
	case KeySpatial2DSphere:
		s.WriteString("p")
	case KeyHashed:
		s.WriteString("h")
	}
	for _, column := range columns {
		field := strings.ReplaceAll(column.Field, "_", "")
		s.WriteByte('_')
		s.WriteString(field)
	}
	return s.String()
}
func Primary(fields ...string) []Column {
	return index(KeyPrimary, fields...)
}
func Unique(fields ...string) []Column {
	return index(KeyUnique, fields...)
}
func Index(fields ...string) []Column {
	return index(KeyIndex, fields...)
}
func FullText(fields ...string) []Column {
	return index(KeyText, fields...)
}
func Spatial(fields ...string) []Column {
	return index(KeySpatial, fields...)
}
func Spatial2D(fields ...string) []Column {
	return index(KeySpatial2D, fields...)
}
func Spatial2DSphere(fields ...string) []Column {
	return index(KeySpatial2DSphere, fields...)
}
func Hashed(fields ...string) []Column {
	return index(KeyHashed, fields...)
}

func NewIndexes(indexes ...[]Column) Indexes {
	idx := make(Indexes, len(indexes))
	for _, columns := range indexes {
		if len(columns) == 0 {
			continue
		}
		idxName := makeIndexName(columns[0].Type, columns)
		idx[idxName] = columns
	}
	return idx
}

func (s Indexes) Primary(indexName ...string) []string {
	name := KeyPrimaryName
	if len(indexName) > 0 {
		name = indexName[0]
	}
	if columns, ok := s[name]; ok {
		return parseFieldNames(columns)
	}
	return nil
}

func (s Indexes) PrimaryKey(indexName ...string) (string, *ae.Error) {
	primaries := s.Primary(indexName...)
	if len(primaries) == 1 {
		return primaries[0], nil
	}
	if len(primaries) == 0 {
		return "", ErrMissingPKDefinition
	}
	return "", ErrMultiplePKDefinitions
}

// List all indexes of these types
// E.g. List(KeyUnique, KeyIndex)
func (s Indexes) List(types ...Type) map[string][]string {
	indexes := make(map[string][]string)
	for name, columns := range s {
		if len(columns) == 0 {
			continue
		}
		if slices.Contains(types, columns[0].Type) {
			indexes[name] = parseFieldNames(columns)
		}
	}
	if len(indexes) == 0 {
		return nil
	}
	return indexes
}
