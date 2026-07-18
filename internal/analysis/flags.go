package analysis

type Flag uint64

const (
	SingleFileModule Flag = 1 << iota
	BootstrapModule       // Module used inside the Klar compiler
	REPLModule            // Module used for the REPL. Allow unused values
	VariadicParam         // *Variable is a variadic function param
	ModuleWithErrors      // Module has errors and cannot be imported
	// [*Expr] refers to a constant or is supposed to be constant
	ConstExpr
	// When parsing types, only types declared in this module are allowed
	LocalOnly
	// Passed to [Checker.parseType] to avoid errors for a generic without params
	genericLHS
)

func parseFlags[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](flags []T) (flag T) {
	for _, fl := range flags {
		flag |= fl
	}
	return
}

func (f Flag) Has(flag Flag) bool {
	return (f & flag) != 0
}
