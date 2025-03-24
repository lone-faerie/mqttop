package maputil

func ContainsKey[Map ~map[K]V, K comparable, V any](m Map, key K) (ok bool) {
	_, ok = m[key]
	return
}
