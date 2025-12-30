package argparse

// VarArgByName returns the variadic parameters for the argument named arg.
// It panics if arg is not defined or is not the last argument.
func (p *Parser) VarArgByName(arg string) []string {
	if len(p.ArgDefs) == 0 {
		panic("argument " + arg + " not defined")
	} else if len(p.Args) == 0 {
		return p.Args[:0]
	}
	// Variadic argument is guaranteed to be the last argument
	defsBefore := len(p.ArgDefs) - 1
	lastDef := p.ArgDefs[defsBefore]
	if lastDef.Name != arg {
		panic("argument " + arg + " not defined or is not the last argument")
	}
	return p.Args[defsBefore:]
}

// ArgAt returns the i'th argument in p.Args. The first argument is at index 1.
// If i is out of range, ArgAt returns "". ArgAt panics if i < 1.
func (p *Parser) ArgAt(i int) string {
	switch {
	case i == 0:
		panic("Parser.Arg(0): the first argument index is 1")
	case i < 0:
		panic("negative argument index")
	case i > len(p.Args):
		return ""
	}
	return p.Args[i-1]
}

// ArgByName returns the value of the argument named name. ArgByName panics
// if name is not defined. If name is a variadic parameter, ArgByName returns
// the first value; [Parser.VariadicArgByName] should be used instead.
func (p *Parser) ArgByName(name string) string {
	i, ok := p.ArgNames[name]
	if !ok {
		panic("argument " + name + " not defined")
	}
	if i >= len(p.Args) {
		return ""
	}
	return p.Args[i]
}

// Flag returns the value of the flag name.
func (p *Parser) Flag(name string) *Flag {
	name = p.resolve(name)
	f, ok := p.Flags[name]
	if !ok {
		return p.FlagDefs[name].Default
	}
	return f
}

func FormatFlag(flag string) string {
	if len(flag) == 1 {
		return "-" + flag
	}
	return "--" + flag
}

func (p *Parser) BoolFlag(name, desc string, def bool, aliases ...string) *Parser {
	p.FlagDefs[name] = FlagDef{
		Type:        TypeBool,
		Default:     newDefaultFlag(TypeBool, def),
		Description: desc,
	}
	p.makeAliases(name, aliases)
	return p
}

func (p *Parser) StringFlag(name, desc, param, def string, aliases ...string) *Parser {
	p.FlagDefs[name] = FlagDef{
		Type:        TypeString,
		Default:     newDefaultFlag(TypeString, def),
		Description: desc,
		ParamName:   param,
	}
	p.makeAliases(name, aliases)
	return p
}

func (p *Parser) EnumFlag(
	name, desc, param string,
	opts map[string]any, def string, aliases ...string,
) *Parser {
	p.FlagDefs[name] = FlagDef{
		Type:        TypeEnum,
		Default:     newDefaultFlag(TypeEnum, &Enum{def, opts[def]}),
		Description: desc,
		ParamName:   param,
	}
	p.SetOptions(name, opts)
	p.makeAliases(name, aliases)
	return p
}

func (p *Parser) Int(name, desc, param string, def int, aliases ...string) *Parser {
	p.FlagDefs[name] = FlagDef{
		Type:        TypeInt,
		Default:     newDefaultFlag(TypeInt, def),
		Description: desc,
		ParamName:   param,
	}
	p.makeAliases(name, aliases)
	return p
}

func (p *Parser) ListFlag(name, desc, param string, def []any, aliases ...string) *Parser {
	p.FlagDefs[name] = FlagDef{
		Type:        TypeList,
		Default:     newDefaultFlag(TypeList, def),
		Description: desc,
		ParamName:   param,
	}
	p.makeAliases(name, aliases)
	return p
}

func (p *Parser) ListFlagOf(
	name, desc, param string,
	itemType FlagType, opts map[string]any,
	def []any, aliases ...string,
) *Parser {
	p.FlagDefs[name] = FlagDef{
		Type:        TypeList,
		ItemType:    itemType,
		Default:     newDefaultFlag(TypeList, def),
		Description: desc,
		ParamName:   param,
	}
	p.makeAliases(name, aliases)
	if opts != nil {
		p.SetOptions(name, opts)
	}
	return p
}

// ResolveFlag finds the flag name from any alises. ResolveFlag returns name if not found.
func (p *Parser) ResolveFlag(name string) string {
	return p.resolve(name)
}

func (p *Parser) SetOptions(flag string, opts map[string]any) {
	if p.enumOpts == nil {
		p.enumOpts = make(map[string]map[string]any)
	}
	p.enumOpts[flag] = opts
}