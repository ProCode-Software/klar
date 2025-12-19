package version

import (
	"strconv"
	"strings"
)

type Build int

// Higher is newer
const (
	Release Build = iota
	RC
	Beta
	Alpha
	Dev
)

var suffixString = map[Build]string{
	Release: "", RC: "rc", Beta: "beta",
	Alpha: "alpha", Dev: "dev",
}

var BuildMap = map[string]Build{
	"":      Release,
	"rc":    RC,
	"beta":  Beta,
	"alpha": Alpha,
	"dev":   Dev, "main": Dev,
}

type Version struct {
	Parts    []int
	Build    Build
	BuildNum int
	Commit   string
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
	if v.Build != Release {
		b.WriteString("-" + suffixString[v.Build])
	}
	if v.BuildNum > 0 {
		b.WriteString("-" + strconv.Itoa(v.BuildNum))
	}
	if v.Commit != "" {
		b.WriteString("+" + v.Commit)
	}
	return "v" + b.String()[1:]
}

var Regex = `(\d+)(?:\.(?P<minor>\d+)){,3}`

func Compare(v1, v2 Version) int {
	return 0
}
