package klon

import (
	"maps"
	"sync/atomic"
)

type cache[K comparable, V any] struct {
	Pointer *atomic.Pointer[map[K]V]
}

func makeCache[K comparable, V any]() *cache[K, V] {
	var a atomic.Pointer[map[K]V]
	a.Store(new(make(map[K]V)))
	return &cache[K, V]{Pointer: &a}
}

func (c *cache[K, V]) get(key K) (V, bool) {
	mapper := *c.Pointer.Load()
	value, ok := mapper[key]
	return value, ok
}

func (c *cache[K, V]) set(key K, value V) {
	mapper := *c.Pointer.Load()
	r := make(map[K]V, len(mapper)+1)
	maps.Copy(r, mapper)
	r[key] = value
	c.Pointer.Store(&r)
}
