package memoize

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/weave-lab/cachin/persist"
)

// Options allow the caller to configure how the cache handles a call
type Options struct {
	// RefreshTTL refreshes the TTL on any resource when it's called. This keeps the cache alive as long as a value is being actively used
	RefreshTTL bool

	// ForceRefresh forces the cache to refresh, any time to option is passed the cache is forced to recaculate it's value
	ForceRefresh bool
}

// InMemory takes a function and wraps it in an in-memory cache. The function will not be run again if the timeout
// duration has not fully elapsed since its last run with the same input K. Instead, the previously calculated return
// value will be returned instead. The type K may implement the Keyer interface to provide custom type matching. If the
// Keyer interface is not provided, K will be JSON marshalled to determine matching inputs.
func InMemory[T, K any](ttl time.Duration, fn func(context.Context, K) (T, error)) func(context.Context, K, Options) (T, error) {
	return SkipErr(Func(nil, "", ttl, fn))
}

// OnDisk takes a function and wraps it in an on-disk cache. The function will not be run again if the timeout duration
// has not fully elapsed since its last run with the same input K. Instead the previously calculated return value will
// be returned instead. Additionally, since state is saved on disk, this timeout persists across multiple runs of a
// program. Because this requires writing to a backing file, the cache can fail. If this happens OnDisk will fall back
// on an in-memory cache. The type K may implement the Keyer interface to provide custom type matching. If the Keyer
// interface is not provided, K will be JSON marshalled to determine matching inputs.
func OnDisk[T, K any](dir string, ttl time.Duration, fn func(context.Context, K) (T, error)) func(context.Context, K, Options) (T, error, error) {
	key := filepath.Base(dir)
	cache := persist.NewFsStore(strings.TrimSuffix(dir, key), false)

	return Func(cache, key, ttl, fn)
}

// Func takes a function and wraps it in a cache. The returned function will use the provided store to cache the return
// value of the function. The function will not be run again if the timeout duration has not fully elapsed since it's
// last run. Instead, the previously calculated return value will be returned instead. The provided store allows this
// timeout to be respected even across multiple runs. However, because the store may fail this behavior is not guaranteed
// If the store cache does fail, Func will fall back on an in-memory cache.
func Func[T, K any](store persist.Store, key string, ttl time.Duration, fn func(context.Context, K) (T, error)) func(context.Context, K, Options) (T, error, error) {
	dataMap := persist.NewDataMap[T](store, key)

	return func(ctx context.Context, in K, options Options) (T, error, error) {
		data, loadErr := dataMap.Load(ctx, in)

		if options.RefreshTTL {
			(*data).RefreshTTL()
		}

		if options.ForceRefresh || data.IsUnset() || (*data).IsExpired(ttl) {
			got, err := fn(ctx, in)
			if err != nil {
				return data.Get(), loadErr, err
			}

			err = data.Set(ctx, got)
			if err != nil {
				return got, err, nil
			}
		}

		return data.Get(), nil, nil
	}
}

// SkipErr ignores cache errors in a cached function. It can be used to simplify a functions signature if you don't
// care about cache errors
func SkipErr[T, K any](fn func(context.Context, K, Options) (T, error, error)) func(context.Context, K, Options) (T, error) {
	return func(ctx context.Context, in K, o Options) (T, error) {
		t, _, err := fn(ctx, in, o)
		return t, err
	}
}
