// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package osenv

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type fakeEnv struct {
	data map[string]string
}

// TestEnv tests the Environment interface
// GIVEN an EnvVarDesc that is used to load data from the os Env vars
//
//	WHEN the env var is defined or missing
//	THEN ensure that LoadFromEnv and GetEnv work correctly
func TestEnv(t *testing.T) {
	const (
		fooKey     = "key"
		fooVal     = "val"
		missingKey = "missingKey"
		missingVal = "missingDefault"
	)
	f := fakeEnv{data: map[string]string{fooKey: fooVal}}
	gf := genEnvFunc
	genEnvFunc = f.getEnv
	defer func() {
		genEnvFunc = gf
	}()

	var tests = []struct {
		name        string
		expectedVal string
		expectErr   bool
		desc        EnvVarDesc
	}{
		{name: "valExists", expectedVal: fooVal, expectErr: false,
			desc: EnvVarDesc{
				Key:        fooKey,
				DefaultVal: "",
				Required:   false,
			},
		},
		{name: "valDefault", expectedVal: missingVal, expectErr: false,
			desc: EnvVarDesc{
				Key:        missingKey,
				DefaultVal: missingVal,
				Required:   false,
			},
		},
		{name: "valRequiredButMissing", expectedVal: "", expectErr: true,
			desc: EnvVarDesc{
				Key:        missingKey,
				DefaultVal: "",
				Required:   true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := NewEnv()
			err := e.LoadFromEnv([]EnvVarDesc{test.desc})
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			v := e.GetEnv(test.desc.Key)
			assert.Equal(t, test.expectedVal, v)
		})
	}
}

func (e fakeEnv) getEnv(key string) string {
	return e.data[key]
}
