package framework

import "testing"

// TestIsBodyFunc - test function for introspecting an interface value
func TestIsBodyFunc(t *testing.T) {
	type args struct {
		body interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test using a function",
			args: args{body: func() {}},
			want: true,
		},
		{
			name: "Test using a struct",
			args: args{body: args{}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBodyFunc(tt.args.body); got != tt.want {
				t.Errorf("isBodyFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}
