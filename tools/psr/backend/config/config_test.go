// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"testing"
)

type envVarTestDesc struct {
	val         string
	expectedVal string
}

type envMap map[string]envVarTestDesc

func Test(t *testing.T) {
	e := getEnvFunc
	defer func() {
		getEnvFunc = e
	}()
	var tests = []struct {
		name    string
		env     envMap
		envVars []envVarTestDesc
	}{
		{name: "1",
			env: envMap{
				"PSR_WORKER_TYPE": envVarTestDesc{val: "WT_EXAMPLE", expectedVal: "WT_EXAMPLE"},
			},
		}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			getEnvFunc = func(key string) string {
				desc, ok := test.env[key]
				if !ok {
					return ""
				}
				return desc.val
			}
			c, err := GetCommonConfig(vzlog.DefaultLogger())
			assert.NoError(t, err)
			assert.Len(t, c., )
		})
	}
}

//
//func TestGetCommonConfig(t *testing.T) {
//	e := getEnvFunc
//	defer func() {
//		getEnvFunc = e
//	}()
//	getEnvFunc = func(string) string {
//		return ""
//	}
//
//
//	c, err := GetCommonConfig(vzlog.DefaultLogger())
//	assert.NoError(t, err)
//	assert.Len(t, c, 3)
//
//}
