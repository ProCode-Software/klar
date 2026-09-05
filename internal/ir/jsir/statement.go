package jsir

type Statement interface{ _stmt() }

type Block struct {
	Statements []Statement
}

// Control flow
// ========

// TypeScript types are omitted from these because these are never
// used in .dts declarations

type IfStatement struct {
	Condition Expression
	Then      *Block
	Else      *Block // Can be nil
}

type (
	DebuggerStatement struct{}
	ContinueStatement struct {
		Label string // Can be empty
	}
	BreakStatement struct {
		Label string // Can be empty
	}
	ThrowStatement  struct{ Expression Expression }
	ReturnStatement struct {
		Expression Expression // Can be nil
	}
)

type SwitchStatement struct {
	Subject Expression
	Cases   []SwitchCase
	Default *Block // Can be nil
}

type SwitchCase struct {
	Expression Expression
	Body       *Block
}

type ForLoopKind int

const (
	ForOfLoop ForLoopKind = iota
	ForInLoop
	ForCLoop // let i = 0; i < ...; ...
)

type ForStatement struct {
	Kind     ForLoopKind
	CForLoop *[3]Statement

	// Specific to for-of/for-in loops
	BindingKind BindingKind
	Variable    Destructure
	Iterator    Expression

	Body *Block
}

type WhileStatement struct {
	Condition Expression
	Body      *Block
}

type LabelledStatement struct {
	Name      string
	Statement Statement
}

type TryStatement struct {
	Try       *Block
	CatchExpr Expression // If Catch != nil, but can still be nil
	Catch     *Block     // Can be nil if Finally != nil
	Finally   *Block     // Can be nil
}

// 'with' statements are intentionally not supported as they are deprecated by ECMAScript
