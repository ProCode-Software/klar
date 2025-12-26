package analysis

type Flag uint64

const (
	SingleFileModule Flag = 1 << iota
)

func parseFlags(flags []Flag) (flag Flag) {
	for _, fl := range flags {
		flag |= fl
	}
	return
}
