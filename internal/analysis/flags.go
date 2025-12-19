package analysis

type Flag uint64

const (
	SingleFileModule Flag = 1 << iota
)
