package version

type Range struct {
	From, To *Version
	Open     bool // If ..<
}

func (r Range) String() string {
	if r.Open {
		return r.From.String() + "..<" + r.To.String()
	}
	return r.From.String() + "..." + r.To.String()
}
