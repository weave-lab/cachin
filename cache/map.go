package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/weave-lab/cachin/persist"
)

// Map is a cached map that can be used to cache data using a persist.Store
type Map[K fmt.Stringer, V any] struct {
	data          map[string]persist.Data[V]
	store         persist.Store
	evictionTimer time.Duration
	ttl           time.Duration
}

func NewMap[K fmt.Stringer, V any](ctx context.Context, store persist.Store, ttl, evictionTimer time.Duration) Map[K, V] {
	m := Map[K, V]{
		data:          make(map[string]persist.Data[V]),
		evictionTimer: evictionTimer,
		ttl:           ttl,
	}

	go m.runCleanup(ctx)

	return m
}

func (m *Map[K, V]) Set(ctx context.Context, k K, v V) error {
	d := persist.NewData[V](m.store, k.String())
	err := d.Set(ctx, v)
	if err != nil {
		return err
	}

	m.data[k.String()] = d
	return nil
}

func (m *Map[K, V]) Get(k K) (V, bool) {
	data, ok := m.data[k.String()]
	if !ok || data.IsExpired(m.ttl) {
		return *new(V), false
	}

	return data.Get(), true
}

func (m *Map[K, V]) runCleanup(ctx context.Context) {
	ttlWait := time.NewTicker(m.evictionTimer)
	for {
		for k, v := range m.data {
			if v.IsExpired(m.ttl) {
				// TODO: we need to handle this error somehow
				_ = v.Delete(ctx)
				delete(m.data, k)
			}
		}

		select {
		case <-ttlWait.C:
		case <-ctx.Done():
			return
		}
	}
}
