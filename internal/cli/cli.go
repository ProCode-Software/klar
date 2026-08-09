package cli

import (
	"os"

	"github.com/ProCode-Software/klar/internal/version"
)

var (
	KlarVersion = "0.0.0"
	KlarCommit  string
)

const (
	KlarGitHub = "https://github.com/ProCode-Software/klar"
	KlarIssues = KlarGitHub + "/issues"
)

var KlarVersionAndCommit = KlarVersion + "+" + KlarCommit

var ParsedKlarVersion version.Version

func init() {
	var err error
	ParsedKlarVersion, err = version.Parse(KlarVersion)
	if err != nil {
		panic("failed to parse Klar version: " + err.Error())
	}
}

var Debug = os.Getenv("KLAR_DEBUG") == "1"
