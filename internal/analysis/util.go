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

func indexBuiltin(builtin, f string) (Type, *klarerrs.Error) {
	builtinObj := builtinModule.Context.Lookup(builtin)
	if builtinObj == nil {
		panic("invalid builtin: " + builtin)
	}
	if builtinObj.IsTypeName() {
		if bt, ok := builtinObj.TypeName().Type.(*bootstrapType); ok {
			return bt.IndexDot(f)
		}
	}
	// TODO: Is this reachable?
	indexer, ok := Underlying(builtinObj.typ).(Indexer)
	if !ok {
		panic("builtin " + builtin + " does not implement Indexer")
	}
	return indexer.IndexDot(f)
}

func indexError(code klarerrs.Code, t Type, label string) *klarerrs.Error {
	err := &klarerrs.Error{
		Code:  code,
		Label: label,
		Info:  klarerrs.TypeErrorInfo{GotType: t.String()},
	}
	return err
}

func indexTypeMismatchError(code klarerrs.Code, exp, got Type, label string) *klarerrs.Error {
	err := &klarerrs.Error{
		Code:  code,
		Label: label,
		Info:  klarerrs.TypeErrorInfo{ExpectedType: exp.String(), GotType: got.String()},
	}
	return err
}

func quoteAka(t Type) string {
	ts := t.String()
	if und := Underlying(t).String(); und != ts {
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
