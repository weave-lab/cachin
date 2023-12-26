package persist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Forever tells cache data to persist forever and never expire
var Forever = time.Duration(0)

var (
	// ErrExternalCache indicates there was an error reading or writing to an external cache
	// this error may mean the external cache has become out of date. However, even if this error is returned
	// the cache will be safe to use as it will fall back on and in-memory cache.
	ErrExternalCache = errors.New("could not read/write from the external cache")

	// ErrNotSerializable indicates there was an error serializing the underlying data into a storable format
	// this error may mean the external cache has become out of date. However, even if this error is returned
	// the cache will be safe to use as it will fall back on an in-memory cache.
	ErrNotSerializable = errors.New("return type could not be serialized/deserialized")

	// ErrFailedKey indicates there was an error converting an incoming parameter into a valid key
	// this error may mean the external cache has become out of date. However, even if this error is returned
	// the cache will be safe to use as it will fall back on an in-memory cache.
	ErrFailedKey = errors.New("failed to convert input into valid key")
)

// Store is an interface that can be used by a Data struct to back up it's internal value to any
// external persistent storage mechanism
type Store interface {
	Get(context.Context, string) ([]byte, time.Time, error)
	Set(context.Context, string, []byte) error
}

// Serializable is an optional interface that can be used to customize the way a Data struct serializes its data
// if this interface is not provided, jsonMarshall and jsonUnmarshal will be used instead.
type Serializable interface {
	Bytes() ([]byte, error)
	FromBytes([]byte) error
}

// Data wraps a value in a persistent data type. Once created, Load can be called to restore the value from a persistent
// data store. the Get() and Set() methods can be used to read and update the value and will attempt to keep the external
// data store in sync. Even if the external data store goes out of sync, Data is safe to use, however, future calls to
// Load may retrieve old data.
type Data[T any] struct {
	value   T
	lastSet time.Time
	store   Store
	key     string
}

// NewData wraps the initial in a Data type. If the provided store is non-nil, Data will sync it's internal value
// to the external store
func NewData[T any](store Store, key string) Data[T] {
	return Data[T]{
		store: store,
		key:   key,
	}
}

// Load will load the initial data from the external store. If the store is nil or the Data has already been set
// Load is a no-op. Load can safely be called multiple times.
func (d *Data[T]) Load(ctx context.Context) error {
	if d.IsUnset() && d.store != nil {
		// try to populate the initial value from the cache
		raw, lastUpdate, err := d.store.Get(ctx, d.key)

		// if lastUpdate is missing that's considered a cache failure since we can't then know how old the data is
		if err != nil {
			return fmt.Errorf("%w | %s", ErrExternalCache, err)
		}
		if lastUpdate.IsZero() {
			return fmt.Errorf("%w | last update was not set", ErrExternalCache)
		}

		tmp := Data[T]{}
		err = tmp.FromBytes(raw)
		if err != nil {
			return fmt.Errorf("%w | %s", ErrNotSerializable, err)
		}

		d.value = tmp.value
		d.lastSet = lastUpdate
	}

	return nil
}

// Get returns the underlying value of the data value
func (d *Data[T]) Get() T {
	return d.value
}

// Age returns how long it has been since the Data was last Set
func (d *Data[T]) Age() time.Duration {
	return time.Since(d.lastSet)
}

// Set will set the Data's internal value, it will always succeed at setting the in memory value. However, setting the
// store value may fail. If this happens, Data is still safe to use, and it's value will still reflect the update.
// however, the data in the external store will not be updated and may be out of date the next time the backed value is created.
func (d *Data[T]) Set(ctx context.Context, a T) error {
	d.value = a
	d.lastSet = time.Now()

	if d.store != nil {
		raw, err := d.Bytes()
		if err != nil {
			return fmt.Errorf("%w | %s", ErrNotSerializable, err)
		}

		err = d.store.Set(ctx, d.key, raw)
		if err != nil {
			return fmt.Errorf("%w | %s", ErrExternalCache, err)
		}
	}

	return nil
}

// IsUnset returns true if the value has never been set
func (d *Data[T]) IsUnset() bool {
	return d.lastSet.IsZero()
}

func (d *Data[T]) ResetTTL() {
	d.lastSet = time.Now()
}

// Bytes converts the value int a slice of bytes, so it can be stored. If the underlying type implements the
// Serializable interface that will be used. Otherwise, the type is JSON marshalled
func (d *Data[T]) Bytes() ([]byte, error) {
	if s, ok := any(d.value).(Serializable); ok {
		return s.Bytes()
	}

	return json.Marshal(d.value)
}

// FromBytes takes a slice of bytes and hydrates Data. It can fail if the by format is incorrect. If the underlying
// type implements the Serializable interface that will be used. Otherwise, the type is JSON marshalled
func (d *Data[T]) FromBytes(bytes []byte) error {
	if s, ok := any(d.value).(Serializable); ok {
		return s.FromBytes(bytes)
	}

	tmp := *new(T)
	err := json.Unmarshal(bytes, &tmp)
	if err != nil {
		return err
	}

	d.value = tmp
	return nil
}

// IsExpired can be used to determine if a Data value is expired in relation to the provided expiration
func (d *Data[T]) IsExpired(ttl time.Duration) bool {
	if ttl == Forever {
		return false
	}
	return d.Age() > ttl
}

// Keyer allows the underlying type to be converted to a Key that can be used for memoization
type Keyer interface {
	Key() string
}

// DataMap is a map of Data values, the underlying data values are keyed based of the Keyer interface
type DataMap[T any] struct {
	values    map[string]*Data[T]
	store     Store
	keyPrefix string
}

// NewDataMap creates a new DataMap type that shares the store. keyPrefix will be used as the prefix for all keys
// belonging to the underlying values.
func NewDataMap[T any](store Store, keyPrefix string) DataMap[T] {
	return DataMap[T]{
		values:    make(map[string]*Data[T]),
		store:     store,
		keyPrefix: keyPrefix,
	}
}

// Load calls load on all the underlying Data values. This is safe to call multiple times as new underlying Data values
// are added to the map
func (d *DataMap[T]) Load(ctx context.Context, in any) (*Data[T], error) {
	var key string
	if k, ok := in.(Keyer); ok {
		key = k.Key()
	} else {
		rawKey, err := json.Marshal(in)
		if err != nil {
			return nil, ErrFailedKey
		}

		key = string(rawKey)
	}

	if _, ok := d.values[key]; !ok {
		tmp := NewData[T](d.store, d.keyPrefix+key)
		d.values[key] = &tmp
	}

	return d.values[key], d.values[key].Load(ctx)
}
