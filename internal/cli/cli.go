package cli

import "github.com/ProCode-Software/klar/internal/version"

var KlarVersion, KlarCommit string

var ParsedKlarVersion version.Version

func init() {
	var err error
	v := KlarVersion
	if KlarCommit != "" {
		v += "+" + KlarCommit
	}
	ParsedKlarVersion, err = version.Parse(v)
	if err != nil {
		panic("failed to parse Klar version: " + err.Error())
	}
}
