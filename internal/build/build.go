package build

import (
	"strings"
	"time"
)

type (
	WarnLevel uint8
	InputKind int
)

const (
	KindFile InputKind = iota
	KindPackage
	KindModule
	KindStdin
)

const (
	_ WarnLevel = iota
	SuppressWarning
	WarningAsError
)

func CompileString(s, fileName string) (pc *ProjectCompiler, res *Result, err error) {
	cwd, err := Cwd()
	if err != nil {
		return
	}
	pc = NewProjectCompiler(NewCompiler(ModeBuild, cwd))
	pc.Inputs = append(pc.Inputs, &Input{Path: fileName, Kind: KindFile})
	pc.Parser = NewStaticParser(fileName, &StaticParserFile{Reader: strings.NewReader(s)})
	pc.StartTime = time.Now()
	res, err = pc.Compile()
	return
}
