package argparse

type baseFlag struct {
	idx int
}

func (f *baseFlag) Index() int { return f.idx }

type BoolFlag struct {
	baseFlag
	Val bool
}

func (f *BoolFlag) Value() any     { return f.Val }
func (f *BoolFlag) Type() FlagType { return TypeBoolFlag }

type StringFlag struct {
	baseFlag
	Val string
}

func (f *StringFlag) Value() any     { return f.Val }
func (f *StringFlag) Type() FlagType { return TypeStringFlag }

type NumberFlag struct {
	baseFlag
	Val float64
}

func (f *NumberFlag) Value() any     { return f.Val }
func (f *NumberFlag) Type() FlagType { return TypeNumberFlag }

type EnumFlag struct {
	baseFlag
	Val  any
	Name string
}

func (f *EnumFlag) Value() any     { return f.Val }
func (f *EnumFlag) Type() FlagType { return TypeEnumFlag }

type ListFlag struct {
	baseFlag
	Val []any
}

func (f *ListFlag) Value() any     { return f.Val }
func (f *ListFlag) Type() FlagType { return TypeListFlag }
