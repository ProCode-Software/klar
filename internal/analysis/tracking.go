package analysis

type TrackedInfo interface {
	tracked()
}

type TrackedFunc struct {
	// Variables declared outside the function that are captured
	Captures []*Object
}

func (TrackedFunc) tracked() {}
