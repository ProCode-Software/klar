package version

import (
	"errors"
	"fmt"
	"strings"
)

// A Specifier represents a version specifier that can be used to match specific versions.
type Specifier struct {
	specComponent
	// If the specifier specifies a latest version, MatchesLatest is used to match
	// v using b, the build specified. MatchesLatest should determine if v is latest
	// version with build b.
	MatchesLatest func(v Version, b Build) bool
}

// ParseSpecifier parses the version specifier represented by s, returning
// an error if the specifier is invalid.
func ParseSpecifier(s string) (spec Specifier, err error) {
	switch {
	case s == "latest":
		spec.specComponent = &latestComponent{build: Stable}
	case strings.HasPrefix(s, "latest "):
		_, build, _ := strings.Cut(s, "latest ")
		b, ok := BuildMap[build]
		if !ok {
			return spec, fmt.Errorf("invalid build: %s", build)
		}
		spec.specComponent = &latestComponent{build: b}
	case strings.HasPrefix(s, "="):
		_, version, _ := strings.Cut(s, "=")
		version = strings.TrimSpace(version)
		parsedVer, err := Parse(version)
		if err != nil {
			return spec, fmt.Errorf("invalid version: %s", version)
		}
		spec.specComponent = &modifierComponent{exactly, parsedVer}
	}
	return spec, nil
}

func (s Specifier) IsZero() bool { return s.specComponent == nil }

// GetMatches returns the versions in vs that match the specifier.
func (s Specifier) GetMatches(vs []Version) []Version { return nil }

func (s *Specifier) UnmarshalText(text []byte) (err error) {
	*s, err = ParseSpecifier(string(text))
	return err
}

func (s Specifier) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s Specifier) String() string {
	if s.IsZero() {
		return "<nil specifier>"
	}
	return s.specComponent.String()
}

// Components
// ==========

// specComponent represents a component of a version specifier.
type specComponent interface {
	// String returns a string representation of the specifier.
	String() string
}

// Used in modifierComponent
type modifier int

const (
	exactly modifier = iota // =
	from                    // +
	sameMajor
	sameMinor
)

// TODO: update for consistency OR look at preferred formats for String()
type (
	// Examples:
	// 	1.0+
	// 	=3.1.4
	// 	2.x
	// 	2.1.x
	modifierComponent struct {
		keyword modifier
		version Version
	}
	// Example:
	// 	latest
	// 	latest beta
	latestComponent struct{ build Build }
	// Example:
	// 	2.1...3.2
	// 	1..<2.2
	rangeComponent struct {
		min, max Version
		open     bool // true if ..< was used
	}
	anyComponent struct{} // *
)

func (c *modifierComponent) String() string {
	ver := c.version.String()
	switch c.keyword {
	case exactly:
		return "=" + ver
	case from:
		return ver + "+"
	case sameMajor, sameMinor:
		return ver + ".x"
	default:
		panic(fmt.Sprintf("unhandled keyword: %d", c.keyword))
	}
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
func (s *Specifier) Matches(v Version) bool {
	switch c := s.specComponent.(type) {
	case *modifierComponent:
		switch c.keyword {
		// TODO
		}
		return false
	case *anyComponent:
		return true // All versions match '*'
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
	s string, v Version, matchesLatest func(v Version, b Build) bool,
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
