package analysis

import (
	"cmp"

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
