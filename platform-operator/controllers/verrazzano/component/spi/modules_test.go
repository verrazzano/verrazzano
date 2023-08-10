// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewModuleConfigHelmValuesWrapper(t *testing.T) {
	type emptyConfig struct{}
	type nestedConfig struct {
		Name string
	}
	type testConfig struct {
		Name    string        `json:"name,omitempty"`
		BoolVal bool          `json:"myBoolVal"`
		Nested  *nestedConfig `json:"nested,omitempty"`
	}
	tests := []struct {
		name         string
		configObject interface{}
		want         string
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name: "basicMergeTest",
			configObject: &testConfig{
				Name:    "myconfig",
				BoolVal: true,
				Nested:  &nestedConfig{Name: "nested"},
			},
			want: `{
			  "verrazzano": {
				"module": {
				  "spec": {
					"name": "myconfig",
					"myBoolVal": true,
					"nested": {
					  "Name": "nested"
					}
				  }
				}
			  }
			}`,
		},
		{
			name: "EmptyConfigObject",
		},
		{
			name:         "NilConfigObject",
			configObject: &emptyConfig{},
			want: `{
			  "verrazzano": {
				"module": {
				  "spec": {
				  }
				}
			  }
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := tt.wantErr
			if wantErr == nil {
				wantErr = assert.NoError
			}

			got, err := NewModuleConfigHelmValuesWrapper(tt.configObject)
			if !wantErr(t, err, fmt.Sprintf("NewModuleConfigHelmValuesWrapper(%v)", tt.configObject)) {
				return
			}
			if len(tt.want) > 0 {
				assert.JSONEq(t, tt.want, string(got.Raw))
			} else {
				assert.Nil(t, got.Raw)
			}
		})
	}
}
