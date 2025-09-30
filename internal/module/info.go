package module

type Submodule struct {
	Children map[string]*Submodule
	
}

type ModuleInfo struct {
	
}

func (m *Submodule) GetModule(paths []string) (*Submodule, bool) {
	// For resolving modules only when needed
	return nil, false
}