package analysis

type TypeAlias struct {
	resolved Type
}

func (a *TypeAlias) Resolve() Type {
	if a.resolved != nil {
		return a.resolved
	}
	return nil
}

func (a *TypeAlias) Kind() Kind                        { return a.Resolve().Kind() }
func (a *TypeAlias) String() string                    { return a.Resolve().String() }
func (a *TypeAlias) StringWithName(name string) string { return name }

func (c *Checker) resolveFuncAlias(info funcAliasInfo) {
}

func Unalias(t Type) Type {
	if a, ok := t.(*TypeAlias); ok {
		return a.Resolve()
	}
	return t
}

func getDefined(t Type) *DefinedType {
	named, _ := t.(*DefinedType)
	return named
}
