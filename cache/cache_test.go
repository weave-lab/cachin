package cache

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/weave-lab/cachin/persist"
)

func TestInMemory(t *testing.T) {
	type args[T any] struct {
		ttl          time.Duration
		fn           func(context.Context) (T, error)
		forceRefresh bool
	}
	type testCase[T any] struct {
		name    string
		args    args[T]
		minTime time.Duration
		maxTime time.Duration
		want    T
		wantErr bool
	}
	tests := []testCase[string]{
		{
			"short expiration",
			args[string]{
				ttl: time.Millisecond,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "test", nil
				},
				forceRefresh: false,
			},
			time.Millisecond * 200,
			time.Second,
			"test",
			false,
		},
		{
			"long expiration",
			args[string]{
				ttl: time.Second,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "test", nil
				},
				forceRefresh: false,
			},
			time.Millisecond * 100,
			time.Second * 150,
			"test",
			false,
		},
		{
			"never expire",
			args[string]{
				ttl: persist.Forever,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "test", nil
				},
				forceRefresh: false,
			},
			time.Millisecond * 100,
			time.Millisecond * 150,
			"test",
			false,
		},
		{
			"force refresh",
			args[string]{
				ttl: persist.Forever,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "test", nil
				},
				forceRefresh: true,
			},
			time.Millisecond * 150,
			time.Millisecond * 250,
			"test",
			false,
		},
		{
			"with error",
			args[string]{
				ttl: persist.Forever,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "", errors.New("failed")
				},
				forceRefresh: false,
			},
			time.Millisecond * 100,
			time.Millisecond * 250,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			fn := InMemory(tt.args.ttl, tt.args.fn)

			// call it once to warm up the cache
			_, _ = fn(context.Background(), tt.args.forceRefresh)
			time.Sleep(time.Millisecond * 2)

			got, err := fn(context.Background(), tt.args.forceRefresh)
			if got != tt.want {
				t.Errorf("InMemory() = %v, want = %v", got, tt.wantErr)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("InMemory() err = %v, wantErr = %v", err, tt.wantErr)
			}

			// check timing to make sure the cache mechanism works as expected
			duration := time.Since(start)
			if duration > tt.maxTime {
				t.Errorf("InMemory() %v slower than max timeout %v", duration, tt.maxTime)
			}
			if duration < tt.minTime {
				t.Errorf("InMemory() %v faster than min timeout %v", duration, tt.minTime)
			}
		})
	}
}

func TestOnDisk(t *testing.T) {
	type args[T any] struct {
		file         string
		ttl          time.Duration
		fn           func(context.Context) (T, error)
		forceRefresh bool
	}
	type testCase[T any] struct {
		name          string
		args          args[T]
		minTime       time.Duration
		maxTime       time.Duration
		want          string
		wantErr       bool
		wantCacheErr  bool
		wantCacheFile string
	}
	tests := []testCase[string]{
		{
			"short expiration",
			args[string]{
				file: filepath.Join(t.TempDir(), "test"),
				ttl:  time.Millisecond,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "test", nil
				},
				forceRefresh: false,
			},
			time.Millisecond * 200,
			time.Second,
			"test",
			false,
			false,
			`"test"`,
		},
		{
			"long expiration",
			args[string]{
				file: filepath.Join(t.TempDir(), "test"),
				ttl:  time.Second,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "test", nil
				},
				forceRefresh: false,
			},
			time.Millisecond * 100,
			time.Second * 150,
			"test",
			false,
			false,
			`"test"`,
		},
		{
			"never expire",
			args[string]{
				file: filepath.Join(t.TempDir(), "test"),
				ttl:  persist.Forever,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "test", nil
				},
				forceRefresh: false,
			},
			time.Millisecond * 100,
			time.Millisecond * 150,
			"test",
			false,
			false,
			`"test"`,
		},
		{
			"force refresh",
			args[string]{
				file: filepath.Join(t.TempDir(), "test"),
				ttl:  persist.Forever,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "test", nil
				},
				forceRefresh: true,
			},
			time.Millisecond * 150,
			time.Millisecond * 250,
			"test",
			false,
			false,
			`"test"`,
		},
		{
			"with error",
			args[string]{
				file: filepath.Join(t.TempDir(), "test"),
				ttl:  persist.Forever,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "", errors.New("failed")
				},
				forceRefresh: false,
			},
			time.Millisecond * 100,
			time.Millisecond * 250,
			"",
			true,
			true,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			fn := OnDisk(tt.args.file, tt.args.ttl, tt.args.fn)

			// call it once to warm up the cache
			_, _, _ = fn(context.Background(), tt.args.forceRefresh)
			time.Sleep(time.Millisecond * 2)

			got, cacheErr, err := fn(context.Background(), tt.args.forceRefresh)
			if got != tt.want {
				t.Errorf("OnDisk() = %v, want = %v", got, tt.wantErr)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("OnDisk() err = %v, wantErr = %v", err, tt.wantErr)
			}
			if (cacheErr != nil) != tt.wantCacheErr {
				t.Errorf("OnDisk() err = %v, wantErr = %v", cacheErr, tt.wantCacheErr)
			}

			// check timing to make sure the cache mechanism works as expected
			duration := time.Since(start)
			if duration > tt.maxTime {
				t.Errorf("OnDisk() %v slower than max timeout %v", duration, tt.maxTime)
			}
			if duration < tt.minTime {
				t.Errorf("OnDisk() %v faster than min timeout %v", duration, tt.minTime)
			}

			if tt.wantCacheErr || tt.wantErr {
				return
			}
			// check to make sure the cache file exists
			cacheFile, err := os.ReadFile(tt.args.file)
			if err != nil {
				t.Errorf("OnDisk() failed to read cache file %s", err)
			}
			if string(cacheFile) != tt.wantCacheFile {
				t.Errorf("OnDisk() cacheFile = %v, wantCacheFile = %v", string(cacheFile), tt.wantCacheFile)
			}
		})
	}
}
