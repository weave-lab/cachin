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
		ttl     time.Duration
		fn      func(context.Context) (T, error)
		options []Option
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
				options: []Option{},
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
				options: []Option{},
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
				options: []Option{},
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
				options: []Option{WithForceRefresh()},
			},
			time.Millisecond * 300,
			time.Millisecond * 350,
			"test",
			false,
		},
		{
			"reset ttl",
			args[string]{
				ttl: time.Millisecond * 15,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "test", nil
				},
				options: []Option{WithRefreshTTL()},
			},
			time.Millisecond * 100,
			time.Millisecond * 120,
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
				options: []Option{},
			},
			time.Millisecond * 300,
			time.Millisecond * 350,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			fn := InMemory(tt.args.ttl, tt.args.fn)

			ctx := context.Background()
			// call it once to warm up the cache
			_, _ = fn(ctx, tt.args.options...)
			time.Sleep(time.Millisecond * 10)

			// call it a second time to test the basic cache mechanisms
			got, err := fn(ctx, tt.args.options...)
			if got != tt.want {
				t.Errorf("InMemory() = %v, want = %v", got, tt.want)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("InMemory() err = %v, wantErr = %v", err, tt.wantErr)
			}
			time.Sleep(time.Microsecond * 10)

			// call it a third time to test the ttl reset mechanism
			got, err = fn(ctx, tt.args.options...)
			if got != tt.want {
				t.Errorf("InMemory() = %v, want = %v", got, tt.want)
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
		file    string
		ttl     time.Duration
		fn      func(context.Context) (T, error)
		options []Option
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
				options: []Option{},
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
				options: []Option{},
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
				options: []Option{},
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
				options: []Option{WithForceRefresh()},
			},
			time.Millisecond * 300,
			time.Millisecond * 350,
			"test",
			false,
			false,
			`"test"`,
		},
		{
			"reset ttl",
			args[string]{
				file: filepath.Join(t.TempDir(), "test"),
				ttl:  time.Millisecond * 15,
				fn: func(_ context.Context) (string, error) {
					time.Sleep(time.Millisecond * 100)
					return "test", nil
				},
				options: []Option{WithRefreshTTL()},
			},
			time.Millisecond * 100,
			time.Millisecond * 130,
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
				options: []Option{},
			},
			time.Millisecond * 300,
			time.Millisecond * 350,
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
			_, _, _ = fn(context.Background(), tt.args.options...)
			time.Sleep(time.Millisecond * 10)

			// call it a second time to test the basic cache mechanisms
			got, cacheErr, err := fn(context.Background(), tt.args.options...)
			if got != tt.want {
				t.Errorf("OnDisk() = %v, want = %v", got, tt.wantErr)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("OnDisk() err = %v, wantErr = %v", err, tt.wantErr)
			}
			if (cacheErr != nil) != tt.wantCacheErr {
				t.Errorf("OnDisk() err = %v, wantErr = %v", cacheErr, tt.wantCacheErr)
			}
			time.Sleep(time.Millisecond * 10)

			// call it a third time to test the ttl reset mechanism
			got, cacheErr, err = fn(context.Background(), tt.args.options...)
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

func TestSkipErr(t *testing.T) {
	type args struct {
		fn func(context.Context, ...Option) (string, error, error)
	}
	tests := []struct {
		name       string
		args       args
		wantString string
		wantError  bool
	}{
		{
			"skip cache error",
			args{
				fn: func(_ context.Context, _ ...Option) (string, error, error) {
					return "success", errors.New("there was a cache error"), nil
				},
			},
			"success",
			false,
		},
		{
			"return error",
			args{
				fn: func(_ context.Context, _ ...Option) (string, error, error) {
					return "", nil, errors.New("there was a value error")
				},
			},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := SkipErr(tt.args.fn)

			got, err := fn(context.Background())
			if (err != nil) != tt.wantError {
				t.Errorf("SkipErr() err = %v, got = %v", err, tt.wantError)
			}
			if got != tt.wantString {
				t.Errorf("SkipErr() = %s, got = %s", got, tt.wantString)
			}
		})
	}
}

func TestLogErr(t *testing.T) {
	type args struct {
		fn func(context.Context, ...Option) (string, error, error)
	}
	tests := []struct {
		name       string
		args       args
		wantString string
		wantLog    string
		wantError  bool
	}{
		{
			"skip cache error",
			args{
				fn: func(_ context.Context, _ ...Option) (string, error, error) {
					return "success", errors.New("there was a cache error"), nil
				},
			},
			"success",
			"there was a cache error",
			false,
		},
		{
			"return error",
			args{
				fn: func(_ context.Context, _ ...Option) (string, error, error) {
					return "", nil, errors.New("there was a value error")
				},
			},
			"",
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotLog string
			fn := LogErr(tt.args.fn, func(_ context.Context, err error) {
				gotLog = err.Error()
			})

			got, err := fn(context.Background())
			if (err != nil) != tt.wantError {
				t.Errorf("LogErr() err = %v, got = %v", err, tt.wantError)
			}
			if got != tt.wantString {
				t.Errorf("LogErr() = %s, got = %s", got, tt.wantString)
			}
			if gotLog != tt.wantLog {
				t.Errorf("LogErr() log = %s, got = %s", gotLog, tt.wantLog)
			}
		})
	}
}
