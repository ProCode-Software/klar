package analysis

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/lexer"
)

type ConstExpr struct {
	Type
	Value ConstValue
}

func (c *ConstExpr) Underlying() Type { return c.Type }
func (c *ConstExpr) String() string   { return c.Type.String() }
func (c *ConstExpr) ConstString() string {
	if v, ok := c.Value.(ObjectConst); ok {
		// Print the type with the fields
		// TODO: The fields should probably be in the same order
		// as the struct/interface's
		return c.Type.String() + v.String()
	}
	return c.String()
}

// Only simple ConstValues are supported ([FloatConst], [IntConst], [StringConst]).
// ConstValueToConstExpr panics if cv is any other type.
func ConstValueToConstExpr(cv ConstValue) *ConstExpr {
	switch cv := cv.(type) {
	case IntConst:
		return &ConstExpr{IntType, cv}
	case StringConst:
		return &ConstExpr{StringType, cv}
	case FloatConst:
		return &ConstExpr{FloatType, cv}
	default:
		panic(fmt.Sprintf("ConstValueToConstExpr: invalid ConstValueType: %T", cv))
	}
}

type ConstValue interface {
	String() string
	ConstValue() any
}

// Helper initializers
// ===========

func NewIntConstExpr(val int64) *ConstExpr {
	return &ConstExpr{IntType, IntConst(val)}
}

func NewBoolConstExpr(val bool) *ConstExpr {
	return &ConstExpr{BoolType, BoolConst(val)}
}

func NewFloatConstExpr(val float64) *ConstExpr {
	return &ConstExpr{FloatType, FloatConst(val)}
}

func NewStringConstExpr(val string) *ConstExpr {
	return &ConstExpr{StringType, StringConst{Value: val}}
}

// ConstValues
// =======

type UnknownConst struct{}

func (u UnknownConst) ConstValue() any { return nil }
func (u UnknownConst) String() string  { return "invalid" }

type IntConst int64

func (i IntConst) ConstValue() any { return int64(i) }
func (i IntConst) String() string  { return strconv.FormatInt(int64(i), 10) }

// [IntConst.Int64] should be preferred to avoid overflows.
func (i IntConst) Int() int { return int(i) }

type StringConst struct {
	Value string
	len   int // Lazily evaluated
}

func (s StringConst) ConstValue() any { return s.Value }

// TODO: Should single quotes be used?
func (s StringConst) String() string { return strconv.Quote(s.Value) }

func (s StringConst) Len() int {
	if s.len <= 0 {
		// TODO: Count grapheme clusters
		s.len = utf8.RuneCountInString(s.Value)
	}
	return s.len
}

// TODO: Should float constants be represented as [math/big.Float]/[math/big.Rat] instead
// of float64 like the Go type checker does?

type FloatConst float64

func (f FloatConst) ConstValue() any { return float64(f) }
func (f FloatConst) String() string {
	return strconv.FormatFloat(float64(f), 'g', -1, 64)
}

type BoolConst bool

func (b BoolConst) ConstValue() any { return bool(b) }
func (b BoolConst) String() string  { return strconv.FormatBool(bool(b)) }

// An ObjectConst can represent a struct or interface. The type is based
// on [ConstExpr.Type].
type ObjectConst struct {
	Fields map[string]ConstValue
}

func (o ObjectConst) ConstValue() any { return o.Fields }
func (o ObjectConst) String() string {
	var b strings.Builder
	b.WriteByte('(')
	for i, key := range slices.Sorted(maps.Keys(o.Fields)) {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(o.Fields[key].String())
	}
	b.WriteByte(')')
	return b.String()
}

type TupleConst struct {
	Items []ConstValue
}

func (t TupleConst) ConstValue() any { return t.Items }
func (t TupleConst) String() string {
	var b strings.Builder
	b.WriteByte('(')
	for i, item := range t.Items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(item.String())
	}
	b.WriteByte(')')
	return b.String()
}

type ConstHash int64

type ListConst struct {
	Items []ConstValue
}

func (l ListConst) ConstValue() any { return l.Items }
func (l ListConst) String() string {
	var b strings.Builder
	b.WriteByte('[')
	for i, item := range l.Items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(item.String())
	}
	b.WriteByte(']')
	return b.String()
}

/* type MapConst struct {
	Keys    []ConstValue
	Entries map[ConstHash]ConstValue
}

func (m MapConst) ConstValue() any { return m.Entries }
func (m MapConst) String() string {
	var b strings.Builder
	b.WriteString("#{")
	for i, key := range m.Keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(key.String())
		b.WriteString(": ")
		b.WriteString(m.Entries[ConstHash(i)].String())
	}
	b.WriteByte('}')
	return b.String()
}
*/

type RegexConst struct {
	Pattern, Flags string
}

func (r RegexConst) ConstValue() any { return r }
func (r RegexConst) String() string {
	return "#/" + r.Pattern + "/" + r.Flags
}

func ConstBinaryOp(lhs, rhs *ConstExpr, op lexer.TokenType, typ Type) *ConstExpr {
	res := &ConstExpr{Type: typ}
	switch op {
	// Arithmetic
	case lexer.Plus:
	case lexer.Minus:
	case lexer.Asterisk:
	case lexer.Slash:
	case lexer.Percent:
	case lexer.Caret:
	// Logical
	case lexer.AndAnd:
	case lexer.OrOr:
	// Relational
	case lexer.EqualEqual:
	case lexer.NotEqual:
	case lexer.GreaterThan:
	case lexer.LessThan:
	case lexer.GreaterEqualTo:
	case lexer.LessEqualTo:
	default:
		panic("unknown operator: " + op.String())
	}
	if res.Value == nil {
		panic(fmt.Sprintf("invalid operation: %s %s %s", lhs, op, rhs))
	}
	return res
}
