package module

type Submodule struct {
	Children map[string]*Submodule
	ModuleInfo
}

type ModuleInfo struct {
	Name        string
	Importable  bool
	Independent bool   // Cannot import non-std modules if individual file
	Path        string
	CachePath string // Path in cache
}

func (m *Submodule) GetModule(paths []string) (*Submodule, bool) {
	// For resolving modules only when needed
	return nil, false
}
