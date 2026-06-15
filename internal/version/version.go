package version

import (
	"encoding"
	"strconv"
	"strings"
)

type Build int

// Higher is newer
const (
	Stable Build = iota
	RC
	Beta
	Alpha
	Dev
)

func (b Build) String() string {
	switch b {
	case Stable:
		return ""
	case RC:
		return "rc"
	case Beta:
		return "beta"
	case Alpha:
		return "alpha"
	case Dev:
		return "dev"
	default:
		panic("invalid build: " + strconv.Itoa(int(b)))
	}
}

var BuildMap = map[string]Build{
	"":      Stable,
	"rc":    RC,
	"beta":  Beta,
	"alpha": Alpha,
	"dev":   Dev, "main": Dev,
}

type Version struct {
	Parts    []int
	Build    Build
	BuildNum int
}

func (v *Version) Major() int { return v.Parts[0] }
func (v *Version) Minor() int { return v.Part(1) }
func (v *Version) Patch() int { return v.Part(2) }

func (v *Version) Part(n int) int {
	if len(v.Parts) < n+1 {
		return 0
	}
	return v.Parts[n]
}

var _ encoding.TextUnmarshaler = (*Version)(nil)

func (v *Version) UnmarshalText(text []byte) (err error) {
	v2, err := Parse(string(text))
	if err != nil {
		return err
	}
	*v = *v2
	return nil
}

func (v *Version) String() string {
	var b strings.Builder
	if len(v.Parts) == 0 {
		return "v0.0"
	}
	for _, part := range v.Parts {
		b.WriteByte('.')
		b.WriteString(strconv.Itoa(part))
	}
	if v.Build != Stable {
		b.WriteByte(' ')
		b.WriteString(v.Build.String())
	}
	if v.BuildNum > 0 {
		b.WriteByte(' ')
		b.WriteString(strconv.Itoa(v.BuildNum))
	}
	return "v" + b.String()[1:]
}

func (v *Version) Normalize() *Version {
	// v1.2.0 -> v1.2
	// Remove commit info (+...)
	return v
}

var Regex = `(\d+)(?:\.(?P<minor>\d+)){,3}`

func Compare(v1, v2 *Version) int {
	return 0
}
