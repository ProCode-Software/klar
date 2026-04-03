package ast


func equalSlice[T any](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	/* if _, ok := any(a).([]Node); ok {
		for i := range a {
			if !any(a[i]).(Node).Equal(any(b[i]).(Node)) {
				return false
			}
		}
		return true
	} */
	for i := range a {
		if any(a[i]) != any(b[i]) {
			return false
		}
	}
	return true
}
