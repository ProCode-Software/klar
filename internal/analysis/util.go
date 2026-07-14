package analysis

import (
	"cmp"
	"fmt"

	"github.com/ProCode-Software/klar/internal/klarerrs"
)

func sortByOrder(a, b *Object) int { return cmp.Compare(a.order, b.order) }

type mapObject map[string]*Object

func (m *mapObject) Insert(obj *Object) (existing *Object) {
	if *m == nil {
		*m = make(mapObject)
	}
	if existing = (*m)[obj.name]; existing != nil {
		return
	}
	(*m)[obj.name] = obj
	return nil
}

func (m *mapObject) Set(name string, obj *Object) {
	if *m == nil {
		*m = make(mapObject)
	}
	(*m)[name] = obj
}

func (m *mapObject) ContainsName(name string) bool {
	if *m == nil {
		return false
	}
	_, ok := (*m)[name]
	return ok
}

func quote(s string) string { return klarerrs.Quote(s) }

func repeatWithItem[T any](item T, count int) []T {
	result := make([]T, count)
	for i := range result {
		result[i] = item
	}
	return result
}

func indexBuiltin(builtin, f string, t *Expr) *klarerrs.Error {
	builtinObj := builtinModule.Context.Lookup(builtin)
	if builtinObj == nil {
		panic("invalid builtin: " + builtin)
	}
	if builtinObj.IsTypeName() {
		if bt, ok := builtinObj.TypeName().Type.(*bootstrapType); ok {
			return bt.Index(f, t)
		}
	}
	// TODO: Is this reachable?
	indexer, ok := Underlying(builtinObj.typ).(Indexer)
	if !ok {
		panic("builtin " + builtin + " does not implement Indexer")
	}
	return indexer.Index(f, t)
}

func quoteAka(t Type) string {
	ts := t.String()
	if und := UnderlyingTypeName(t).String(); und != ts {
		return fmt.Sprintf("%s (aka %s)", quote(ts), quote(und))
	}
	return quote(ts)
}

func isTODO(t Type) bool {
	if t.Kind() != KindFunction {
		return false
	}
	builtinTODO := BuiltInContext.Lookup("TODO")
	if builtinTODO != nil {
		return t == builtinTODO.typ
	}
	// If currently bootstrapping
	return t == builtinModule.Context.Lookup("TODO").typ
}

func isCloneBuiltin(t Type) bool {
	if t.Kind() != KindFunction {
		return false
	}
	builtinClone := BuiltInContext.Lookup("clone")
	if builtinClone != nil {
		return t == builtinClone.typ
	}
	// If currently bootstrapping
	return t == builtinModule.Context.Lookup("clone").typ
}

func hintWithDiff(err *klarerrs.Error, hint string, edits ...klarerrs.DiffEdit) {
	err.HintWithDiff(hint, klarerrs.NewDiff("", edits...))
}
