package files

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSanitizeFile(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, SanitizeFile(tt.args.path), fmt.Sprintf("SanitizeFile(%v)", tt.args.path))
		})
	}
}

func Test_check(t *testing.T) {
	type args struct {
		e error
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, check(tt.args.e), fmt.Sprintf("check(%v)", tt.args.e))
		})
	}
}

func Test_sanitizeEachLine(t *testing.T) {
	type args struct {
		l string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, sanitizeEachLine(tt.args.l), "sanitizeEachLine(%v)", tt.args.l)
		})
	}
}
