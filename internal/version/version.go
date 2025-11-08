package version

import (
	"strconv"
	"strings"
)

var KlarVersion = "0.0.1"

type Suffix int

// Higher is newer
const (
	Release Suffix = iota
	RC
	Beta
	Alpha
	Dev
)

var suffixString = map[Suffix]string{
	Release: "", RC: "rc", Beta: "beta",
	Alpha: "alpha", Dev: "dev",
}

var SuffixMap = map[string]Suffix{
	"":      Release,
	"rc":    RC,
	"beta":  Beta,
	"alpha": Alpha,
	"dev":   Dev, "main": Dev,
}

type Version struct {
	Parts     []int
	Suffix    Suffix
	SuffixNum int
	Commit    string
}

func (v Version) Major() int { return v.Parts[0] }
func (v Version) Minor() int { return v.Part(1) }
func (v Version) Patch() int { return v.Part(2) }

func (v Version) Part(n int) int {
	if len(v.Parts) < n+1 {
		return 0
	}
	return v.Parts[n]
}

func (v Version) String() string {
	var b strings.Builder
	for _, part := range v.Parts {
		b.WriteString("." + strconv.Itoa(part))
	}
	if v.Suffix != Release {
		b.WriteString("-" + suffixString[v.Suffix])
	}
	if v.SuffixNum > 0 {
		b.WriteString("-" + strconv.Itoa(v.SuffixNum))
	}
	if v.Commit != "" {
		b.WriteString("+" + v.Commit)
	}
	return "v" + b.String()[1:]
}
