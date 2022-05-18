// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	objects  = "./testdata/objects"
	testdata = "./testdata"
)

func TestApplyD(t *testing.T) {
	var tests = []struct {
		name    string
		dir     string
		count   int
		isError bool
	}{
		{
			"should apply YAML files",
			objects,
			3,
			false,
		},
		{
			"should fail to apply non-existent directories",
			"blahblah",
			0,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewFakeClientWithScheme(k8scheme.Scheme)
			y := k8sutil.NewYAMLApplier(c, "")
			err := y.ApplyD(tt.dir)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.count, len(y.Objects()))
			}
		})
	}
}

func TestApplyF(t *testing.T) {
	var tests = []struct {
		name    string
		file    string
		count   int
		isError bool
	}{
		{
			"should apply file",
			objects + "/service.yaml",
			1,
			false,
		},
		{
			"should apply file with two objects",
			testdata + "/two_objects.yaml",
			2,
			false,
		},
		{
			"should fail to apply files that are not YAML",
			"blahblah",
			0,
			true,
		},
		{
			"should fail when file is not YAML",
			objects + "/not-yaml.txt",
			0,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewFakeClientWithScheme(k8scheme.Scheme)
			y := k8sutil.NewYAMLApplier(c, "test")
			err := y.ApplyF(tt.file)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.count, len(y.Objects()))
		})
	}
}

func TestApplyFT(t *testing.T) {
	var tests = []struct {
		name    string
		file    string
		args    map[string]interface{}
		count   int
		isError bool
	}{
		{
			"should apply a template file",
			testdata + "/templated_service.yaml",
			map[string]interface{}{"namespace": "default"},
			1,
			false,
		},
		{
			"should fail to apply when template is incomplete",
			testdata + "/templated_service.yaml",
			map[string]interface{}{},
			0,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewFakeClientWithScheme(k8scheme.Scheme)
			y := k8sutil.NewYAMLApplier(c, "")
			err := y.ApplyFT(tt.file, tt.args)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.count, len(y.Objects()))
		})
	}
}

func TestDeleteF(t *testing.T) {
	var tests = []struct {
		name    string
		file    string
		isError bool
	}{
		{
			"should delete valid file",
			testdata + "/two_objects.yaml",
			false,
		},
		{
			"should fail to delete invalid file",
			"blahblah",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewFakeClientWithScheme(k8scheme.Scheme)
			y := k8sutil.NewYAMLApplier(c, "")
			err := y.DeleteF(tt.file)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteFD(t *testing.T) {
	var tests = []struct {
		name    string
		file    string
		args    map[string]interface{}
		isError bool
	}{
		{
			"should apply a template file",
			testdata + "/templated_service.yaml",
			map[string]interface{}{"namespace": "default"},
			false,
		},
		{
			"should fail to apply when template is incomplete",
			testdata + "/templated_service.yaml",
			map[string]interface{}{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewFakeClientWithScheme(k8scheme.Scheme)
			y := k8sutil.NewYAMLApplier(c, "")
			err := y.DeleteFT(tt.file, tt.args)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteAll(t *testing.T) {
	c := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	y := k8sutil.NewYAMLApplier(c, "")
	err := y.ApplyD(objects)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(y.Objects()))
	err = y.DeleteAll()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(y.Objects()))
}
