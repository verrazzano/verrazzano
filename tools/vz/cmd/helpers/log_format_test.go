// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"github.com/spf13/pflag"
	"testing"
)

func TestLogFormat_AsFlag(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    LogFormat
		wantErr bool
	}{
		{name: "simple", args: []string{"--v=simple"}, want: LogFormatSimple, wantErr: false},
		{name: "json", args: []string{"--v=json"}, want: LogFormatJSON, wantErr: false},
		{name: "invalid", args: []string{"--v=invalid"}, want: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var flags pflag.FlagSet
			flags.Init("test", pflag.ContinueOnError)
			var v LogFormat
			flags.VarP(&v, "v", "v", "usage")
			if err := flags.Parse(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("flags.Parse(LogFormat) error = %v, wantErr %v", err, tt.wantErr)
			}
			if v.String() != tt.want.String() {
				t.Errorf("expected value %q got %q", tt.want.String(), v.String())
			}
			if v.Type() != "format" {
				t.Errorf("expected type to be %q got %q", "format", v.Type())
			}
		})
	}
}
