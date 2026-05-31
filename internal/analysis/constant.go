package analysis

import "unicode/utf8"

type ConstValue interface {
	ConstValue() any
	Type() Type
}

type IntConst struct{ Value int64 }

func (c IntConst) ConstValue() any { return c.Value }
func (c IntConst) Type() Type      { return IntType }

type StringConst struct {
	Value string
	Len   int
}

func (c StringConst) ConstValue() any { return c.Value }
func (c StringConst) Type() Type      { return StringType }

func NewStringConst(s string) StringConst {
	// TODO: Count grapheme clusters instead of runes
	return StringConst{Value: s, Len: utf8.RuneCountInString(s)}
}

type FloatConst struct{ Value float64 }

func (c FloatConst) ConstValue() any { return c.Value }
func (c FloatConst) Type() Type      { return FloatType }
