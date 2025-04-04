package syncutil

import (
	"encoding/json"
	"iter"
	"sync"

	"github.com/lone-faerie/mqttop/log"
)

// Map is a wrapper around a map[K]V that is safe for concurrent use by multiple goroutines.
type Map[K comparable, V any] struct {
	m map[K]V
	sync.Mutex
}

// Make is the concurrency-safe equivalent of make(map[K]V)
func (m *Map[K, V]) Make() {
	log.Debug("syncutil.Map lock", "cmd", "Make")
	m.Lock()
	log.Debug("syscall.Map make", "cmd", "Make")
	m.m = make(map[K]V)
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "Make")
}

// MakeSize is the concurrency-safe equivalent of make(map[K]V, n)
func (m *Map[K, V]) MakeSize(n int) {
	log.Debug("syncutil.Map lock", "cmd", "MakeSize", "n", n)
	m.Lock()
	log.Debug("syscall.Map make", "cmd", "MakeSize", "n", n)
	m.m = make(map[K]V, n)
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "MakeSize")
}

// Clear deletes all the entries, resulting in an empty Map.
func (m *Map[K, V]) Clear() {
	log.Debug("syncutil.Map lock", "cmd", "Clear")
	m.Lock()
	clear(m.m)
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "Clear")

}

// Store sets the value for a key.
func (m *Map[K, V]) Store(k K, v V) {
	log.Debug("syncutil.Map lock", "cmd", "Store")
	m.Lock()
	m.m[k] = v
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "Store")
}

// Load returns the value stored in the map for a key, or the zero value of V if no
// value is present. The ok result indicates whether value was found in the map.
func (m *Map[K, V]) Load(k K) (v V, ok bool) {
	log.Debug("syncutil.Map lock", "cmd", "Load")
	m.Lock()
	v, ok = m.m[k]
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "Load")
	return
}

// Swap swaps the value for a key and returns the previous value if any. The loaded
// result reports whether the key was present.
func (m *Map[K, V]) Swap(k K, v V) (old V, ok bool) {
	log.Debug("syncutil.Map lock", "cmd", "Swap")
	m.Lock()
	old, ok = m.m[k]
	m.m[k] = v
	m.Unlock()
	log.Debug("syncutil.Map unlock", "cmd", "Swap")
	return
}

// Delete deletes the value for a key.
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

// Iter locks m and returns an iterator over entries of m.
// Once iteration is complete, m will be unlocked.
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
