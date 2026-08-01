package analysis

type TrackedInfo interface {
	tracked()
}

type TrackedFunc struct {
	// Variables declared outside the function that are captured
	Captures   []*Object
	SideEffect bool // Whether the function has side effects
}

func (TrackedFunc) tracked() {}
