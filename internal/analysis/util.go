package analysis

import "cmp"

func sortByOrder(a, b *Object) int { return cmp.Compare(a.order, b.order) }

type objectMap map[string]*Object

func (m *objectMap) Insert(obj *Object) (existing *Object) {
	if *m == nil {
		*m = make(objectMap)
	}
	if existing = (*m)[obj.name]; existing != nil {
		return
	}
	(*m)[obj.name] = obj
	return nil
}
