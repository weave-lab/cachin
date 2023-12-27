package persist

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestFsStore_Get(t *testing.T) {
	type fields struct {
		dir        string
		useSafeKey bool
	}
	type args struct {
		ctx context.Context
		key string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantBytes []byte
		wantTS    time.Time
		wantErr   bool
	}{
		{
			"use safe key",
			fields{
				dir: func() string {
					dir := t.TempDir()
					err := os.WriteFile(filepath.Join(dir, SafeKey("safe_key")), []byte(`test`), 0o0644)
					if err != nil {
						t.Error("failed to write test file", err)
					}

					return dir
				}(),
				useSafeKey: true,
			},
			args{
				ctx: context.Background(),
				key: "safe_key",
			},
			[]byte(`test`),
			time.Now(),
			false,
		},
		{
			"file does not exist",
			fields{
				dir:        t.TempDir(),
				useSafeKey: false,
			},
			args{
				ctx: context.Background(),
				key: "test_key",
			},
			nil,
			time.Time{},
			false,
		},
		{
			"permission error",
			fields{
				dir: func() string {
					dir := t.TempDir()
					err := os.WriteFile(filepath.Join(dir, "test_key"), []byte(`test`), 0o0222)
					if err != nil {
						t.Error("failed to write test file", err)
					}
					return dir
				}(),
			},
			args{
				ctx: context.Background(),
				key: "test_key",
			},
			nil,
			time.Time{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FsStore{
				dir:        tt.fields.dir,
				useSafeKey: tt.fields.useSafeKey,
			}
			gotBytes, gotTS, err := c.Get(tt.args.ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("FsStore.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotBytes, tt.wantBytes) {
				t.Errorf("FsStore.Get() gotBytes = %v, want %v", gotBytes, tt.wantBytes)
			}

			// use a range so time stamps dont have to match exactly
			if tt.wantTS.Before(gotTS.Add(-time.Millisecond*10)) ||
				tt.wantTS.After(gotTS.Add(time.Millisecond*10)) {
				t.Errorf("FsStore.Get() gotTS = %v, want %v", gotTS, tt.wantTS)
			}
		})
	}
}

func TestFsStore_Set(t *testing.T) {
	type fields struct {
		dir        string
		useSafeKey bool
	}
	type args struct {
		ctx context.Context
		key string
		val []byte
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantFile []byte
		wantErr  bool
	}{
		{
			"dir does not exist",
			fields{
				dir:        filepath.Join(t.TempDir(), "nested"),
				useSafeKey: false,
			},
			args{
				ctx: context.Background(),
				key: "test_key",
				val: []byte(`test`),
			},
			[]byte(`test`),
			false,
		},
		{
			"use safe key",
			fields{
				dir:        t.TempDir(),
				useSafeKey: true,
			},
			args{
				ctx: context.Background(),
				key: "safe_key",
				val: []byte(`test`),
			},
			[]byte(`test`),
			false,
		},
		{
			"permission denyed",
			fields{
				dir: func() string {
					dir := t.TempDir()
					err := os.Chmod(dir, 0o0666)
					if err != nil {
						t.Error("FsStore.Set() failed to set up dir", err)
					}

					return dir
				}(),
				useSafeKey: false,
			},
			args{
				ctx: context.Background(),
				key: "test_key",
				val: []byte(`test`),
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FsStore{
				dir:        tt.fields.dir,
				useSafeKey: tt.fields.useSafeKey,
			}
			if err := c.Set(tt.args.ctx, tt.args.key, tt.args.val); (err != nil) != tt.wantErr {
				t.Errorf("FsStore.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// check the file was written
			key := tt.args.key
			if c.useSafeKey {
				key = SafeKey(key)
			}

			got, err := os.ReadFile(filepath.Join(c.dir, key))
			if err != nil {
				t.Error("FsStore.Set() failed to read cache file", err)
			}
			if !reflect.DeepEqual(got, tt.wantFile) {
				t.Errorf("FsStore.Set() cache file = %s, wanted = %s", string(got), string(tt.wantFile))
			}
		})
	}
}
