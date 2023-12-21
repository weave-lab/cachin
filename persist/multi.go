package persist

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MultiStore can be used to store cache data in multiple stores
type MultiStore struct {
	stores []Store
	expire time.Duration
}

// Get attempts to get the document from the provided cache stores one at a time. The first store that
// successful returns the data
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

// Set attempts to set the document at all the provided stores one at a time. Any errors returned from
// a store will be aggregated and returned
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

// Delete attempts to delete the data associated with the key in eacy configured data stores. Ay errors returned
// for any of the stores will be aggregated and returned
func (s *MultiStore) Delete(ctx context.Context, key string) error {
	var errs []string
	for _, store := range s.stores {
		err := store.Delete(ctx, key)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	return fmt.Errorf("errs: %v", strings.Join(errs, "|"))
}
