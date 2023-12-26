package persist

import "testing"

func TestSafeKey(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"empty key",
			args{
				key: "",
			},
			"",
		},
		{
			"replace key",
			args{
				key: "key123",
			},
			"a2V5MTIz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeKey(tt.args.key); got != tt.want {
				t.Errorf("SafeKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
