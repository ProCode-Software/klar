package cli

import "github.com/ProCode-Software/klar/internal/version"

var KlarVersion, KlarCommit string

var KlarVersionAndCommit = KlarVersion + "+" + KlarCommit

var ParsedKlarVersion *version.Version

func init() {
	var err error
	ParsedKlarVersion, err = version.Parse(KlarVersion)
	if err != nil {
		panic("failed to parse Klar version: " + err.Error())
	}
}
