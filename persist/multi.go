package persist

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type MultiStore struct {
	stores []Store
	expire time.Duration
}

func (s *MultiStore) Get(ctx context.Context, key string) ([]byte, time.Time, error) {
	// look in each store and return the first non-expired source
	var errs []string
	for _, store := range s.stores {
		got, lastUpdate, err := store.Get(ctx, key)
		if err != nil {
			errs = append(errs, err.Error())
		}

		if time.Since(lastUpdate) < s.expire {
			return got, lastUpdate, nil
		}
	}

	if len(errs) > 0 {
		return nil, time.Time{}, fmt.Errorf("errs: %s", strings.Join(errs, "|"))
	}

	return nil, time.Time{}, nil
}

func (s *MultiStore) Set(ctx context.Context, key string, val []byte) error {
	var errs []string
	for _, store := range s.stores {
		err := store.Set(ctx, key, val)
		if err != nil {
			errs = append(errs, err.Error())

		}
	}

	return fmt.Errorf("errs: %v", strings.Join(errs, "|"))
}
