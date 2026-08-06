package utils

// InsertMap is a map that preserves initial insertion order.
type InsertMap[K comparable, V any] struct {
	keyValues map[K]V
	keys      []K
}

func NewInsertMap[K comparable, V any](entries ...MapEntry[K, V]) InsertMap[K, V] {
	m := InsertMap[K, V]{
		keys:      make([]K, 0, len(entries)),
		keyValues: make(map[K]V, len(entries)),
	}
	m.SetEntries(entries...)

	return m
}

func (m InsertMap[K, V]) Entries() []MapEntry[K, V] {
	values := make([]MapEntry[K, V], 0, len(m.keys))
	for _, k := range m.keys {
		values = append(values, MapEntry[K, V]{
			Key:   k,
			Value: m.keyValues[k],
		})
	}

	return values
}

func (m *InsertMap[K, V]) Set(key K, value V) {
	if _, ok := m.keyValues[key]; !ok {
		m.keys = append(m.keys, key)
	}

	m.keyValues[key] = value
}

func (m *InsertMap[K, V]) SetEntries(entries ...MapEntry[K, V]) {
	for _, e := range entries {
		m.Set(e.Key, e.Value)
	}
}
