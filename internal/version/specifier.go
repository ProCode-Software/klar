package version

import (
	"errors"
	"fmt"
)

// A Specifier represents a version specifier that can be used to match specific versions.
type Specifier struct {
	specComponent
	// If the specifier specifies a latest version, MatchesLatest is used to match
	// v using b, the build specified. MatchesLatest should determine if v is latest
	// version with build b.
	MatchesLatest func(v *Version, b Build) bool
}

// ParseSpecifier parses the version specifier represented by s, returning
// an error if the specifier is invalid.
func ParseSpecifier(s string) (*Specifier, error) {
	return nil, nil
}

// GetMatches returns the versions in vs that match the specifier.
func (s *Specifier) GetMatches(vs []*Version) []*Version { return nil }

// CodableSpecifier wraps [Specifier] and implements [encoding.TextMarshaler]
// and [encoding.TextUnmarshaler] to allow it to be serialized and deserialized
// as a string.
type CodableSpecifier struct{ Specifier }

// Components
// ==========

// specComponent represents a component of a version specifier.
type specComponent interface {
	// String returns a string representation of the specifier.
	String() string
}

// Used in modifierComponent
const (
	exactly = iota // =
	from           // >=
	sameMajor
	sameMinor
	below // <
	above // >
	upTo  // <=
)

// TODO: update for consistency OR look at preferred formats for String()
type (
	// Examples:
	// 	from 1.0
	// 	exactly 3.1.4
	// 	>= 2.1
	// 	= 3.5
	modifierComponent struct {
		keyword int
		version *Version
	}
	// Example:
	// 	latest
	// 	latest beta
	latestComponent struct{ build Build }
	// Example:
	// 	2.1...3.2
	// 	1..<2.2
	rangeComponent struct {
		min, max *Version
		open     bool // true if ..< was used
	}
	anyComponent struct{} // *, any
)

func (c *modifierComponent) String() string {
	keywords := map[int]string{
		exactly:   "=",
		from:      ">=",
		sameMajor: "TODO",
		sameMinor: "TODO",
		below:     "<",
		above:     ">",
		upTo:      "<=",
	}
	return keywords[c.keyword] + c.version.String()
}

func (c *rangeComponent) String() string {
	min := c.min.String()
	max := c.max.String()
	if c.open {
		return min + "..<" + max
	}
	return min + "..." + max
}

func (c *anyComponent) String() string { return "*" }

func (c *latestComponent) String() string {
	if c.build == 0 { // Release
		return "latest"
	}
	return "latest " + c.build.String()
}

// Matching
// ========

// Matches reports whether v matches the versions specified by s.
func (s *Specifier) Matches(v *Version) bool {
	switch c := s.specComponent.(type) {
	case *modifierComponent:
		switch c.keyword {
		// TODO
		}
		return false
	case *anyComponent:
		return true // All versions match any
	case *latestComponent:
		if s.MatchesLatest == nil {
			// Can't match without knowing the latest version
			panic("s.Matches(v): s.MatchesLatest must be provided to match latest version specifier")
		}
		return s.MatchesLatest(v, c.build)
	case *rangeComponent:
		if c.open { // ..< Excluding the max version
			return Compare(v, c.min) >= 0 && Compare(v, c.max) < 0
		}
		return Compare(v, c.min) >= 0 && Compare(v, c.max) <= 0
	default:
		panic(fmt.Sprintf("unhandled specComponent: %T", c))
	}
}

// IsLatest reports whether s specifies a latest version.
func (s *Specifier) IsLatest() bool {
	_, ok := s.specComponent.(*latestComponent)
	return ok
}

func ParseSpecifierAndMatch(
	s string, v *Version, matchesLatest func(v *Version, b Build) bool,
) (bool, error) {
	spec, err := ParseSpecifier(s)
	if err != nil {
		return false, err
	}
	if spec.IsLatest() && matchesLatest == nil {
		return false, errors.New(
			"ParseSpecifierAndMatch(_, v, matchesLatest): " +
				"matchesLatest must be provided to match latest version specifier v",
		)
	}
	spec.MatchesLatest = matchesLatest
	return spec.Matches(v), nil
}
