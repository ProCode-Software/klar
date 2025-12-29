package analysis

type Flag uint64

const (
	SingleFileModule Flag = 1 << iota
	BootstrapModule       // Module used inside the Klar compiler
	REPLModule            // Module used for the REPL. Allow unused values
	OpaqueType
)

func parseFlags(flags []Flag) (flag Flag) {
	for _, fl := range flags {
		flag |= fl
	}
	return
}

func (f Flag) Has(flag Flag) bool {
	return (f & flag) != 0
}
