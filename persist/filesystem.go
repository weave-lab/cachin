package persist

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"
)

// FsStore is a Store that uses the filesystem to store cache data
type FsStore struct {
	afs        afero.Fs
	dir        string
	useSafeKey bool
}

// NewFsStore creates a new FsStore, dir is the rood directory where all cached files will be stored
func NewFsStore(dir string, useSafeKey bool) *FsStore {
	return &FsStore{
		afs:        afero.NewOsFs(),
		dir:        dir,
		useSafeKey: useSafeKey,
	}
}

// Get searches for a file that matches the provided key in the stores root directory. If the file is missing
// no error will be returned
func (c *FsStore) Get(_ context.Context, key string) ([]byte, time.Time, error) {
	if c.useSafeKey {
		key = SafeKey(key)
	}
	file := filepath.Join(c.dir, key)
	stat, err := os.Stat(file)
	switch {
	case os.IsNotExist(err):
		return nil, time.Time{}, nil
	case err != nil:
		return nil, time.Time{}, err
	}

	raw, err := afero.ReadFile(c.afs, file)
	if err != nil {
		return nil, time.Time{}, err
	}

	return raw, stat.ModTime(), nil
}

// Set writes or updates a file that matches the provided key in the stores root directory. The file will contain
// the raw bytes passed in by val
func (c *FsStore) Set(_ context.Context, key string, val []byte) error {
	if _, err := c.afs.Stat(c.dir); os.IsNotExist(err) {
		err := c.afs.MkdirAll(c.dir, 0750)
		if err != nil {
			return err
		}
	}

	if c.useSafeKey {
		key = SafeKey(key)
	}
	file := filepath.Join(c.dir, key)
	err := afero.WriteFile(c.afs, file, val, 0666)
	if err != nil {
		return err
	}

	return nil
}
