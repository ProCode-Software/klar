package version

// parse parses a version literal. An error is returned if the version is invalid.
func Parse(s string) (*Version, error) {
	v, _, err := parse(s, true)
	return v, err
}

// parse parses a version literal. The length of the parsed version is
// returned. If full is true, an error is returned if there are trailing
// characters after the version.
func parse(s string, full bool) (v *Version, n int, err error) {
	// TODO: Special error if contains underscore
	v = &Version{}
	return
}
