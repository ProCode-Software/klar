package analysis

import (
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

// bootstrapType is a type that is used to bootstrap the typechecker.
// It wraps a type and a kind, so the typechecker allows operations
// for the provided kind on the wrapped type.
//
// Example: If the builtin module declares `type List`, allow list
// operations on the `List` type (such as iteration).
type bootstrapType struct {
	asDeclared Type // Most likely a struct
	kind       Kind
	withKind   Type // Type if it actually had the kind
	MethodSet
}

func (bt *bootstrapType) Kind() Kind       { return bt.kind }
func (bt *bootstrapType) Underlying() Type { return bt.withKind }
func (bt *bootstrapType) String() string   { return bt.kind.String() }

var _ interface {
	SupportsMethods
	Indexer
} = &bootstrapType{}

func (c *Checker) wrapCompositeBootstrapTypes() {
	for _, ct := range compositeTypes {
		// Queue it so we can store tn.Type, but still run this before functions are checked
		c.queue(func() {
			obj := c.rootContext.Lookup(ct.declaredName)
			if obj == nil || !obj.IsTypeName() {
				return
			}
			tn := obj.TypeName()
			tn.Type = &bootstrapType{
				asDeclared: tn.Type,
				kind:       ct.kind,
				withKind:   ct.asKind(c.rootContext),
			}
		}, false)
	}
}

func (bt *bootstrapType) Index(i Type) (Type, *klarerrs.Error) {
	if indexer, ok := Underlying(bt.withKind).(Indexer); ok {
		return indexer.Index(i)
	}
	if indexer, ok := Underlying(bt.asDeclared).(Indexer); ok {
		return indexer.Index(i)
	}
	return nil, nil
}

func (bt *bootstrapType) IndexDot(i string) (Type, *klarerrs.Error) {
	if indexer, ok := Underlying(bt.asDeclared).(Indexer); ok {
		return indexer.IndexDot(i)
	}
	return nil, nil
}

func (bt *bootstrapType) CanIndex() bool {
	_, ok := Underlying(bt.asDeclared).(Indexer)
	return ok
}
