package syncx

import "sync"

type Map[K comparable, V any] struct {
	syncMap sync.Map
}

func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.syncMap.Load(key)
	if !ok {
		return value, ok
	}
	return v.(V), ok
}

func (m *Map[K, V]) Store(key K, value V) {
	m.syncMap.Store(key, value)
}

func (m *Map[K, V]) Delete(key K) {
	m.syncMap.Delete(key)
}

func (m *Map[K, V]) Clear() {
	m.syncMap.Clear()
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (V, bool) {
	a, loaded := m.syncMap.LoadOrStore(key, value)
	return a.(V), loaded
}

func (m *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	v, loaded := m.syncMap.LoadAndDelete(key)
	if !loaded {
		var zeroValue V
		return zeroValue, loaded
	}
	return v.(V), loaded
}

func (m *Map[K, V]) Swap(key K, value V) (V, bool) {
	previous, loaded := m.syncMap.Swap(key, value)
	return previous.(V), loaded
}

func (m *Map[K, V]) CompareAndSwap(key K, old, new V) bool {
	return m.syncMap.CompareAndSwap(key, old, new)
}

func (m *Map[K, V]) CompareAndDelete(key K, old V) bool {
	return m.syncMap.CompareAndDelete(key, old)
}

func (m *Map[K, V]) Range(callback func(key K, value V) bool) {
	m.syncMap.Range(func(key, value any) bool {
		return callback(key.(K), value.(V))
	})
}
