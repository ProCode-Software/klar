package build

type (
	BuildMode int
	WarnLevel uint8
)

const (
	KindFile InputKind = iota
	KindPackage
	KindModule
	KindStdin
)

const (
	_ WarnLevel = iota
	SuppressWarning
	WarningAsError
)
