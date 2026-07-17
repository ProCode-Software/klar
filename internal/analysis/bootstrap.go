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
	for declaredName, ct := range compositeTypes {
		obj := c.module.Context.Lookup(declaredName)
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
	for name, kind := range primitives {
		obj := c.module.Context.Lookup(name)
		if obj == nil || !obj.IsTypeName() {
			continue
		}
		tn := obj.TypeName()
		tn.Type = &bootstrapType{
			asDeclared: tn.Type,
			kind:       kind,
			withKind:   kind,
		}
	}
}

func (c *Checker) wrapBootstrappedTypeName(tn *TypeName, recv *Object) *TypeName {
	if kind, ok := primitives[recv.Name]; ok {
		return &TypeName{Name: tn.Name, Type: &bootstrapType{
			asDeclared: recv.TypeName().Type,
			kind:       kind,
			withKind:   kind,
		}}
	}
	if ct, ok := compositeTypes[recv.Name]; ok {
		return &TypeName{Name: tn.Name, Type: &bootstrapType{
			asDeclared: recv.TypeName().Type,
			kind:       ct.kind,
			withKind:   ct.asKind(c.module.Context),
		}}
	}
	return tn
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
	indexer, ok := bt.withKind.(Indexer)
	if !ok {
		return nil
	}
	return indexer.Index(i, t)
	/* 	if indexer, ok := Underlying(bt.asDeclared).(Indexer); ok {
		return indexer.Index(i, t)
	}
	return nil */
}

func (bt *bootstrapType) CanIndex() bool {
	_, ok := Underlying(bt.asDeclared).(Indexer)
	return ok
}

func lookupBootstrap(name string) *Object {
	return builtinModule.Context.Lookup(name)
}
