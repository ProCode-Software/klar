package build

type (
	WarnLevel uint8
	InputKind int
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
