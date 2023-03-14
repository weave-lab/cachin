package persist

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
)

type testStore struct {
	data map[string]rawData
	err  error
}

func (t *testStore) Get(_ context.Context, key string) ([]byte, time.Time, error) {
	return t.data[key].Raw, t.data[key].LastSet, t.err
}

func (t *testStore) Set(_ context.Context, key string, data []byte) error {
	if t.err != nil {
		return t.err
	}
	t.data[key] = rawData{
		Raw:     data,
		LastSet: time.Date(2020, 05, 15, 10, 0, 0, 0, time.UTC),
	}

	return nil
}

func TestData_Load(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	type testCase[T any] struct {
		name    string
		d       Data[T]
		args    args
		want    Data[T]
		wantErr bool
	}
	tests := []testCase[string]{
		{
			"successful load",
			Data[string]{
				store: &testStore{
					data: map[string]rawData{
						"test": {
							Raw:     []byte(`"test value"`),
							LastSet: time.Date(2020, 05, 15, 10, 0, 0, 0, time.UTC),
						},
					},
				},
				key: "test",
			},
			args{
				ctx: context.Background(),
			},
			Data[string]{
				store: &testStore{
					data: map[string]rawData{
						"test": {
							Raw:     []byte(`"test value"`),
							LastSet: time.Date(2020, 05, 15, 10, 0, 0, 0, time.UTC),
						},
					},
				},
				key:     "test",
				value:   "test value",
				lastSet: time.Date(2020, 05, 15, 10, 0, 0, 0, time.UTC),
			},
			false,
		},
		{
			"failed load",
			Data[string]{
				store: &testStore{
					err: errors.New("failed to load"),
				},
			},
			args{
				ctx: context.Background(),
			},
			Data[string]{
				store: &testStore{
					err: errors.New("failed to load"),
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.d.Load(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.d, tt.want) {
				t.Errorf("Load() value = %v, wantValue %v", tt.d.value, tt.want)
			}
		})
	}
}

func TestData_Set(t *testing.T) {
	type args[T any] struct {
		ctx context.Context
		a   T
	}
	type testCase[T any] struct {
		name    string
		d       Data[T]
		args    args[T]
		want    Data[T]
		wantErr bool
	}
	tests := []testCase[string]{
		{
			"successful set",
			Data[string]{
				value: "test value",
				store: &testStore{
					data: map[string]rawData{},
				},
				key: "test",
			},
			args[string]{
				ctx: context.Background(),
				a:   "test value 2",
			},
			Data[string]{
				value: "test value 2",
				store: &testStore{
					data: map[string]rawData{
						"test": {
							Raw:     []byte(`"test value 2"`),
							LastSet: time.Now(),
						},
					},
				},
			},
			false,
		},
		{
			"failed to set",
			Data[string]{
				value: "test value",
				store: &testStore{
					err: errors.New("failed"),
				},
			},
			args[string]{
				ctx: context.Background(),
				a:   "test value 2",
			},
			Data[string]{
				value: "test value 2",
				store: &testStore{
					err: errors.New("failed"),
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.d.Set(tt.args.ctx, tt.args.a); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestData_IsExpired(t *testing.T) {
	type args struct {
		ttl time.Duration
	}
	type testCase[T any] struct {
		name string
		d    Data[T]
		args args
		want bool
	}
	tests := []testCase[string]{
		{
			"is expired",
			Data[string]{
				lastSet: time.Time{},
			},
			args{
				ttl: time.Second,
			},
			true,
		},
		{
			"is fresh",
			Data[string]{
				lastSet: time.Now(),
			},
			args{
				ttl: time.Hour,
			},
			false,
		},
		{
			"never expires",
			Data[string]{
				lastSet: time.Time{},
			},
			args{
				ttl: Forever,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.IsExpired(tt.args.ttl); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

type serializableType int

func (s *serializableType) Bytes() ([]byte, error) {
	return []byte(fmt.Sprintf("%X", *s)), nil
}

func (s *serializableType) FromBytes(b []byte) error {
	conv, err := strconv.ParseInt(string(b), 16, 64)
	if err != nil {
		return err
	}

	*s = serializableType(conv)
	return nil
}

func TestData_Bytes(t *testing.T) {
	type testCase[T any] struct {
		name    string
		d       Data[T]
		want    []byte
		wantErr bool
	}
	tests := []testCase[any]{
		{
			"serializable",
			Data[any]{
				value: func() *serializableType {
					i := serializableType(255)
					return &i
				}(),
			},
			[]byte("FF"),
			false,
		},
		{
			"not serializable",
			Data[any]{
				value: 10,
			},
			[]byte("10"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.d.Bytes()
			if (err != nil) != tt.wantErr {
				t.Errorf("Bytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Bytes() got = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}

func TestData_FromBytes(t *testing.T) {
	type args struct {
		bytes []byte
	}
	type testCase[T any] struct {
		name    string
		d       Data[T]
		args    args
		wantErr bool
	}
	tests := []testCase[any]{
		{
			"json unmarshal",
			Data[any]{
				value: 10,
			},
			args{
				[]byte("10"),
			},
			false,
		},
		{
			"deserialize",
			Data[any]{
				value: func() *serializableType {
					i := serializableType(255)
					return &i
				}(),
			},
			args{
				[]byte("FF"),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.d.FromBytes(tt.args.bytes); (err != nil) != tt.wantErr {
				t.Errorf("FromBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
