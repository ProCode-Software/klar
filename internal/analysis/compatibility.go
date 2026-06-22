package analysis

// Compatible returns whether type a is compatible with b.
func Compatible(a, b Type) bool {
	if b, ok := b.(Kind); ok && !b.IsPrimitive() {
		return Underlying(a).Kind() == b
	}
	// TODO
	return Underlying(a) == Underlying(b)
}
