package build

type Progress interface {
	ResolvingInput(path string, curr, total int)
	DownloadingDeps()
	CompilingDep(pkgName string, curr, total int)
	LocatingModules(input string, count int)
	ParsingModule(path string, curr, total int)
	CheckingModule(path string, curr, total int)
	// TODO: Codegen
}

var _ Progress = HiddenProgress{}

type HiddenProgress struct{}

func (HiddenProgress) ResolvingInput(string, int, int) {}
func (HiddenProgress) DownloadingDeps()                {}
func (HiddenProgress) CompilingDep(string, int, int)   {}
func (HiddenProgress) LocatingModules(string, int)     {}
func (HiddenProgress) ParsingModule(string, int, int)  {}
func (HiddenProgress) CheckingModule(string, int, int) {}
