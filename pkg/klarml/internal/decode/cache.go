package decode

import (
	"sync/atomic"
)

type cache[K comparable, V any] struct {
	Pointer *atomic.Pointer[map[K]V]
}

func MakeCache[K comparable, V any]() *cache[K, V] {
	var a atomic.Pointer[map[K]V]
	mapper := make(map[K]V)
	a.Store(&mapper)
	return &cache[K, V]{Pointer: &a}
}

func (c *cache[K, V]) Get(key K) (V, bool) {
	mapper := *c.Pointer.Load()
	value, ok := mapper[key]
	return value, ok
}

func (c *cache[K, V]) Set(key K, value V) {
	mapper := *c.Pointer.Load()
	r := make(map[K]V, len(mapper)+1)
		for key, elm := range mapper {
		r[key] = elm
	}
	r[key] = value
	c.Pointer.Store(&r)
}
