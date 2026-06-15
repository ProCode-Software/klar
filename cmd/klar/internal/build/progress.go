package build

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/util"
)

var _ build.Progress = &BuildStatus{}

type BuildStatus struct {
	loading bool
	cwd     string
}

func NewBuildStatus(cwd string) *BuildStatus {
	return &BuildStatus{cwd: cwd}
}

func (s *BuildStatus) ResolvingInput(path string, curr, total int) {
	s.printf("📁 Locating input <c>%s</c> </><d>%d/%d</d>", s.rel(path), curr, total)
}

func (s *BuildStatus) DownloadingDeps() {
	s.printf("📦 Downloading dependencies")
}

func (s *BuildStatus) CompilingDep(pkgName string, curr, total int) {
	s.printf("🏗️ Compiling dependency <c>%s</c> </><d>%d/%d</d>", pkgName, curr, total)
}

func (s *BuildStatus) LocatingModules(input string, count int) {
	s.printf("🔍 Locating modules in <c>%s</c> </><d>%d</d>", s.rel(input), count)
}

func (s *BuildStatus) CheckingModule(path string, curr, total int) {
	s.printf("🧠 Typechecking module <c>%s</c> </><d>%d/%d</d>", s.rel(path), curr, total)
}

func (s *BuildStatus) printf(f string, args ...any) {
	s.loading = false
	fmt.Print(ansi.ClearLine)
	// TODO: should it be bold?
	fmt.Printf(ansi.Colorize("<m **>"+f+"</>")+"\r", args...)
	s.loading = true
}

func (s *BuildStatus) rel(path string) string { return util.RelPath(s.cwd, path) }
