package syncutil

import (
	"iter"
	"sync"
)

type Map[K comparable, V any] struct {
	m  map[K]V
	mu sync.Mutex
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{m: make(map[K]V)}
}

func (m *Map[K, V]) Put(k K, v V) {
	m.mu.Lock()
	m.m[k] = v
	m.mu.Unlock()
}

func (m *Map[K, V]) Get(k K) (v V, ok bool) {
	m.mu.Lock()
	v, ok = m.m[k]
	m.mu.Unlock()
	return
}

func (m *Map[K, V]) Delete(k K) {
	if m == nil {
		return
	}
	m.mu.Lock()
	delete(m.m, k)
	m.mu.Unlock()
}

func (m *Map[K, V]) Iter() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.mu.Lock()
		defer m.mu.Unlock()
		for k, v := range m.m {
			if !yield(k, v) {
				return
			}
		}
	}
}
