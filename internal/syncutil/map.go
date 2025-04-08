package syncutil

import (
	"fmt"
	"strconv"
	"sync"
)

type Map[K ~string, V any] struct {
	sync.Map
}

func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.Map.Load(key)
	if ok {
		value = v.(V)
	}
	return
}

func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := m.Map.LoadAndDelete(key)
	if loaded {
		value = v.(V)
	}
	return
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	v, loaded := m.Map.LoadOrStore(key, value)
	if loaded {
		actual = v.(V)
	} else {
		actual = value
	}
	return
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.Map.Range(f)
}

func (m *Map[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	v, loaded := m.Map.Swap(key, value)
	if loaded {
		previous = v.(V)
	}
	return
}

func (m *Map[K, V]) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	b := []byte{'{'}
	first := true
	m.Range(func(key K, value any) bool {
		if !first {
			b = append(b, ',')
		}
		b = strconv.AppendQuote(b, string(key))
		b = append(b, ':')
		switch v := value.(type) {
		case string:
			b = strconv.AppendQuote(b, v)
		case bool:
			b = strconv.AppendBool(b, v)
		case uint, uint8, uint16, uint32, uint64:
			b = strconv.AppendUint(b, v, 10)
		case int, int8, int16, int32, int64:
			b = strconv.AppendInt(b, v, 10)
		case float32:
			b = strconv.AppendFloat(b, v, 'f', -1, 32)
		case float64:
			b = strconv.AppendFloat(b, v, 'f', -1, 64)
		default:
			fmt.Append(b, value)
		}
		return true
	})
	b = append(b, '}')
	return b, nil
}
