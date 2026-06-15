package version

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/lexer"
)

// parse parses a version literal. An error is returned if the version is invalid.
func Parse(s string) (*Version, error) {
	v, _, err := parse(s, true)
	return v, err
}

// parse parses a version literal. The length of the parsed version is
// returned. If full is true, an error is returned if there are trailing
// characters after the version.
func parse(s string, full bool) (v *Version, n int, err error) {
	if len(s) == 0 {
		return nil, 0, errors.New("version is empty")
	}
	// TODO: Special error if contains underscore
	v = &Version{}
	if s[0] == 'v' {
		n++
	}
	if len(s) == 1 {
		return nil, 1, errors.New("expected version number after 'v'")
	}
parseParts:
	for n < len(s) {
		if isDigit(s[n]) {
			var (
				number    int
				overflows bool
				start     = n
			)
			for n < len(s) && isDigit(s[n]) {
				oldNum := number
				number = (10 * number) + int(s[n]-'0')
				if number < oldNum {
					overflows = true
				}
				n++
			}
			if overflows {
				return nil, n, fmt.Errorf("version number %s is too large", s[start:n])
			}
			v.Parts = append(v.Parts, number)
			if n >= len(s) {
				break
			}
		}
		switch {
		case s[n] == '.':
			n++
			if n >= len(s) || !isDigit(s[n]) {
				return nil, n, errors.New("expected a number after dot")
			}
			continue
		case s[n] == ' ':
			n++
			if n >= len(s) && full {
				return v, n, errors.New("trailing space not allowed")
			}
			break parseParts

		case !full:
			return v, n, nil
		case s[n] == '_':
			return nil, n, errors.New("version numbers can't contain underscores")
		default:
			r, _ := utf8.DecodeRuneInString(s[n:])
			return nil, n, fmt.Errorf("invalid character in version: %q", r)
		}
	}
	switch {
	case len(v.Parts) == 0:
		return nil, n, errors.New("expected at least 1 number in version")
	case len(v.Parts) > 4:
		return nil, n, errors.New("expected at most 4 parts in version")
	case n >= len(s):
		return v, n, nil
	}
	// Read build
	buildLen := strings.IndexByte(s[n:], ' ')
	if buildLen < 0 {
		buildLen = len(s[n:])
	}
	buildStr := s[n : n+buildLen]
	var ok bool
	if v.Build, ok = BuildMap[buildStr]; !ok {
		return v, n, errors.New("invalid build: " + buildStr)
	}
	n += buildLen

	if n >= len(s) {
		return v, n, nil
	}
	// Build number
	if s[n] == ' ' && s[n+1:] != "" {
		n++
		buildNumLen := strings.IndexFunc(s[n:], func(r rune) bool { return !lexer.IsDigit(r) })
		if buildNumLen < 0 {
			buildNumLen = len(s[n:])
		}
		if v.BuildNum, err = strconv.Atoi(s[n : n+buildNumLen]); err != nil {
			return v, n, fmt.Errorf("invalid build number: %q", s[n:n+buildNumLen])
		}
		n += buildNumLen
	}
	if n < len(s) && full {
		r, _ := utf8.DecodeRuneInString(s[n:])
		return nil, n, fmt.Errorf("invalid character in version: %q", r)
	}
	return v, n, nil
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }
