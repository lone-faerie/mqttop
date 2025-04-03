package syncutil

import (
	"encoding/json"
	"iter"
	"sync"

	"github.com/lone-faerie/mqttop/log"
)

type Map[K comparable, V any] struct {
	m map[K]V
	sync.Mutex
}

func (m *Map[K, V]) Make() {
	log.Debug("syncutil.Map lock", "cmd", "Make")
	m.Lock()
	log.Debug("syscall.Map make", "cmd", "Make")
	m.m = make(map[K]V)
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "Make")
}

func (m *Map[K, V]) MakeSize(n int) {
	log.Debug("syncutil.Map lock", "cmd", "MakeSize", "n", n)
	m.Lock()
	log.Debug("syscall.Map make", "cmd", "MakeSize", "n", n)
	m.m = make(map[K]V, n)
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "MakeSize")
}

func (m *Map[K, V]) Clear() {
	log.Debug("syncutil.Map lock", "cmd", "Clear")
	m.Lock()
	clear(m.m)
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "Clear")

}

func (m *Map[K, V]) Store(k K, v V) {
	log.Debug("syncutil.Map lock", "cmd", "Store")
	m.Lock()
	m.m[k] = v
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "Store")
}

func (m *Map[K, V]) Load(k K) (v V, ok bool) {
	log.Debug("syncutil.Map lock", "cmd", "Load")
	m.Lock()
	v, ok = m.m[k]
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "Load")
	return
}

func (m *Map[K, V]) Swap(k K, v V) (old V, ok bool) {
	log.Debug("syncutil.Map lock", "cmd", "Swap")
	m.Lock()
	old, ok = m.m[k]
	m.m[k] = v
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "Swap")
	return
}

func (m *Map[K, V]) Delete(k K) {
	if m == nil {
		return
	}
	log.Debug("syncutil.Map lock", "cmd", "Delete")
	m.Lock()
	delete(m.m, k)
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "Delete")
}

func (m *Map[K, V]) Iter() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		log.Debug("syncutil.Map lock", "cmd", "Iter")
		m.Lock()
		defer log.Debug("syncutil.Map unlock", "cmd", "Iter")
		defer m.Unlock()
		for k, v := range m.m {
			if !yield(k, v) {
				return
			}
		}
	}
}

func (m *Map[K, V]) MarshalJSON() ([]byte, error) {
	log.Debug("syncutil.Map lock", "cmd", "MarshalJSON")
	m.Lock()
	defer log.Debug("syncutil.Map unlock", "cmd", "MarshalJSON")
	defer m.Unlock()
	return json.Marshal(m.m)
}

func (m *Map[K, V]) UnmarshalJSON(b []byte) error {
	log.Debug("syncutil.Map lock", "cmd", "UnmarshalJSON")
	m.Lock()
	defer log.Debug("syncutil.Map unlock", "cmd", "UnmarshalJSON")
	defer m.Unlock()
	return json.Unmarshal(b, &m.m)
}
