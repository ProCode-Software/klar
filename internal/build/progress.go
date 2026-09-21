package build

type Progress interface {
	ResolvingInput(path string, curr, total int)
	DownloadingDeps()
	CompilingDep(pkgName string, curr, total int)
	LocatingModules(input string, count int)
	CheckingModule(path string, curr, total int)
	GeneratingJS(curr, total int)
	OptimizingModule(module string, curr, total int)
	WritingModules()
}

var _ Progress = HiddenProgress{}

type HiddenProgress struct{}

func (HiddenProgress) ResolvingInput(string, int, int)   {}
func (HiddenProgress) DownloadingDeps()                  {}
func (HiddenProgress) CompilingDep(string, int, int)     {}
func (HiddenProgress) LocatingModules(string, int)       {}
func (HiddenProgress) CheckingModule(string, int, int)   {}
func (HiddenProgress) GeneratingJS(int, int)             {}
func (HiddenProgress) OptimizingModule(string, int, int) {}
func (HiddenProgress) WritingModules()                   {}
