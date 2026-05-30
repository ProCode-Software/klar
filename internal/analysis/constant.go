package analysis

type ConstValue interface {
	ConstValue() any
	Type() Type
}

type IntConst struct{ Value int64 }

func (c IntConst) ConstValue() any { return c.Value }
func (c IntConst) Type() Type      { return IntType }

type StringConst struct{ Value string }

func (c StringConst) ConstValue() any { return c.Value }
func (c StringConst) Type() Type      { return StringType }

type FloatConst struct{ Value float64 }

func (c FloatConst) ConstValue() any { return c.Value }
func (c FloatConst) Type() Type      { return FloatType }
