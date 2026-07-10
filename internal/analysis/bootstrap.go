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
	kind       Kind // The kind the type represents
	withKind   Type // Type if it actually had the kind
	MethodSet       // TODO: Is this needed?
}

func (bt *bootstrapType) Kind() Kind       { return bt.kind }
func (bt *bootstrapType) Underlying() Type { return bt.withKind }
func (bt *bootstrapType) String() string   { return bt.kind.String() }

var _ interface {
	SupportsMethods
	Indexer
} = &bootstrapType{}

func (c *Checker) wrapBootstrapTypes() {
	for _, ct := range compositeTypes {
		obj := c.module.Context.Lookup(ct.declaredName)
		if obj == nil || !obj.IsTypeName() {
			continue
		}
		tn := obj.TypeName()
		tn.Type = &bootstrapType{
			asDeclared: tn.Type,
			kind:       ct.kind,
			withKind:   ct.asKind(c.module.Context), // Root context to lookup generics
		}
	}
	// Int, String, etc.
	for _, prim := range primitives {
		obj := c.module.Context.Lookup(prim.name)
		if obj == nil || !obj.IsTypeName() {
			continue
		}
		tn := obj.TypeName()
		tn.Type = &bootstrapType{
			asDeclared: tn.Type,
			kind:       prim.typ,
			withKind:   prim.typ,
		}
	}
}

func (c *Checker) wrapBootstrappedTypeName(tn *TypeName, recv *Object) *TypeName {
	old := tn
	tn = new(*tn)
	tn.Type = nil
	for _, prim := range primitives {
		if prim.name == recv.name {
			tn.Type = &bootstrapType{
				asDeclared: recv.TypeName().Type,
				kind:       prim.typ,
				withKind:   prim.typ,
			}
			return tn
		}
	}
	for _, comp := range compositeTypes {
		if comp.declaredName == recv.name {
			tn.Type = &bootstrapType{
				asDeclared: recv.TypeName().Type,
				kind:       comp.kind,
				withKind:   comp.asKind(c.module.Context),
			}
			return tn
		}
	}
	return old
}

func (bt *bootstrapType) IndexComputed(i Type, t *Expr) *klarerrs.Error {
	if indexer, ok := Underlying(bt.withKind).(ComputedIndexer); ok {
		return indexer.IndexComputed(i, t)
	}
	if indexer, ok := Underlying(bt.asDeclared).(ComputedIndexer); ok {
		return indexer.IndexComputed(i, t)
	}
	return nil
}

func (bt *bootstrapType) Index(i string, t *Expr) *klarerrs.Error {
	if indexer, ok := Underlying(bt.asDeclared).(Indexer); ok {
		return indexer.Index(i, t)
	}
	return nil
}

func (bt *bootstrapType) CanIndex() bool {
	_, ok := Underlying(bt.asDeclared).(Indexer)
	return ok
}
