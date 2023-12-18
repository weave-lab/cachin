package cache

import (
	"context"
	"path/filepath"
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

// InMemory takes a function and wraps it in an in-memory cache. The function will not be run again if the timeout duration
// has not fully elapsed since it's last run. Instead, the previously calculated return value will be returned instead
func InMemory[T any](ttl time.Duration, fn func(context.Context) (T, error)) func(context.Context, Options) (T, error) {
	data := persist.Data[T]{}

	return func(ctx context.Context, options Options) (T, error) {

		if options.RefreshTTL {
			data.RefreshTTL()
		}

		if options.ForceRefresh || data.IsUnset() || data.IsExpired(ttl) {
			got, err := fn(ctx)
			if err != nil {
				return data.Get(), err
			}

			// Set can not fail if it's just in memory
			_ = data.Set(ctx, got)
		}

		return data.Get(), nil
	}
}

// OnDisk takes a function and wraps it in an on-disk cache. The function will not be run again if the timeout duration
// has not fully elapsed since it's last run. Instead, the previously calculated return value will be returned instead.
// Additionally, since state is saved on disk, this timeout persists across multiple runs of a program. Because this
// requires writing to a backing file, the cache can fail. If this happens OnDisk will fall back on an in-memory cache.
func OnDisk[T any](file string, ttl time.Duration, fn func(context.Context) (T, error)) func(context.Context, Options) (T, error, error) {
	store := persist.NewFsStore(filepath.Dir(file), false)
	key := filepath.Base(file)

	return Func(store, key, ttl, fn)
}

// Func takes a function and wraps it in a cache. The returned function will use the provided store to cache the return
// value of the function. The function will not be run again if the timeout duration has not fully elapsed since it's
// last run. Instead, the previously calculated return value will be returned instead. The provided store allows this
// timeout to be respected even across multiple runs. However, because the store may fail this behavior is not guaranteed
// If the store cache does fail, Func will fall back on an in-memory cache.
func Func[T any](store persist.Store, key string, ttl time.Duration, fn func(context.Context) (T, error)) func(context.Context, Options) (T, error, error) {
	data := persist.NewData[T](store, key)

	return func(ctx context.Context, options Options) (T, error, error) {
		loadErr := data.Load(ctx)

		if options.RefreshTTL {
			data.RefreshTTL()
		}

		if options.ForceRefresh || data.IsUnset() || data.IsExpired(ttl) {
			got, err := fn(ctx)
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
// need to explicitly handle cache errors
func SkipErr[T any](fn func(context.Context, bool) (T, error, error)) func(context.Context, bool) (T, error) {
	return func(ctx context.Context, b bool) (T, error) {
		t, _, err := fn(ctx, b)
		return t, err
	}
}

// LogErr logs cache errors in the cached function. It can be used to simplify a function's signature if you can
// just log cache errors and not explicitly handle them
func LogErr[T any](fn func(context.Context, bool) (T, error, error), log func(context.Context, error)) func(context.Context, bool) (T, error) {
	return func(ctx context.Context, b bool) (T, error) {
		t, cacheErr, err := fn(ctx, b)
		if cacheErr != nil {
			log(ctx, cacheErr)
		}

		return t, err
	}
}
