// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

// TestGetInstallOverridesYAML tests GetInstallOverridesYAML
// GIVEN an override list
//
//	WHEN I call GetInstallOverridesYAML
//	THEN I get a list of key value pairs of files from the override sources
func TestGetInstallOverridesYAML(t *testing.T) {
	trueval := true
	dataKey := "testKey"
	wrongKey := "wrongKey"
	testName := "testName"
	dataVal := "dataVal"
	goodJSON := "{\"foo\": {\"foo\": \"bar\"}}"
	goodYAML := "foo:\n  foo: bar\n"
	badJSON := "{\"foo\": {\"foo\": \"bar\"}"

	tests := []struct {
		name          string
		overrides     []v1alpha1.Overrides
		expectError   bool
		expectCMGet   bool
		expectSecGet  bool
		expectValGet  bool
		expectCMData  map[string]string
		expectSecData map[string][]byte
	}{
		{
			name:        "test no overrides",
			overrides:   []v1alpha1.Overrides{},
			expectError: false,
		},
		{
			name: "test nil refs",
			overrides: []v1alpha1.Overrides{
				{
					ConfigMapRef: nil,
					SecretRef:    nil,
				},
			},
			expectError: false,
		},
		{
			name: "test nil selectors",
			overrides: []v1alpha1.Overrides{
				{
					ConfigMapRef: nil,
					SecretRef:    nil,
				},
			},
			expectError: false,
		},
		{
			name: "test configMap selectors",
			overrides: []v1alpha1.Overrides{
				{
					ConfigMapRef: &v1.ConfigMapKeySelector{
						Key: dataKey,
						LocalObjectReference: v1.LocalObjectReference{
							Name: testName,
						},
					},
				},
			},
			expectError: false,
			expectCMGet: true,
			expectCMData: map[string]string{
				dataKey: dataVal,
			},
		},
		{
			name: "test Secret selectors",
			overrides: []v1alpha1.Overrides{
				{
					SecretRef: &v1.SecretKeySelector{
						Key: dataKey,
						LocalObjectReference: v1.LocalObjectReference{
							Name: testName,
						},
					},
				},
			},
			expectError:  false,
			expectSecGet: true,
			expectSecData: map[string][]byte{
				dataKey: []byte(dataVal),
			},
		},
		{
			name: "test invalid data selectors",
			overrides: []v1alpha1.Overrides{
				{
					SecretRef: &v1.SecretKeySelector{
						Key: dataKey,
						LocalObjectReference: v1.LocalObjectReference{
							Name: testName,
						},
					},
				},
			},
			expectError:  true,
			expectSecGet: true,
			expectSecData: map[string][]byte{
				wrongKey: []byte(dataVal),
			},
		},
		{
			name: "test invalid data selectors optional",
			overrides: []v1alpha1.Overrides{
				{
					ConfigMapRef: &v1.ConfigMapKeySelector{
						Key: dataKey,
						LocalObjectReference: v1.LocalObjectReference{
							Name: testName,
						},
						Optional: &trueval,
					},
				},
			},
			expectError: false,
			expectCMGet: true,
			expectSecData: map[string][]byte{
				wrongKey: []byte(dataVal),
			},
		},
		{
			name: "test valid data selectors optional",
			overrides: []v1alpha1.Overrides{
				{
					ConfigMapRef: &v1.ConfigMapKeySelector{
						Key: dataKey,
						LocalObjectReference: v1.LocalObjectReference{
							Name: testName,
						},
						Optional: &trueval,
					},
				},
			},
			expectError: false,
			expectCMGet: true,
			expectSecData: map[string][]byte{
				dataKey: []byte(dataVal),
			},
		},
		{
			name: "test overrideValue valid YAML",
			overrides: []v1alpha1.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: []byte(goodJSON),
					},
				},
			},
			expectError:  false,
			expectValGet: true,
		},
		{
			name: "test overrideValue valid YAML",
			overrides: []v1alpha1.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: []byte(badJSON),
					},
				},
			},
			expectError:  true,
			expectValGet: true,
		},
	}

	a := assert.New(t)
	mock := gomock.NewController(t)
	client := mocks.NewMockClient(mock)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectCMGet {
				client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(
					func(ctx context.Context, nsn types.NamespacedName, configmap *v1.ConfigMap) error {
						configmap.Data = tt.expectCMData
						return nil
					})
			}
			if tt.expectSecGet {
				client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(
					func(ctx context.Context, nsn types.NamespacedName, sec *v1.Secret) error {
						sec.Data = tt.expectSecData
						return nil
					})
			}

			ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v12.ObjectMeta{Namespace: "foo"}}, nil, false)

			data, err := GetInstallOverridesYAML(ctx, tt.overrides)
			if tt.expectError {
				a.Error(err)
			} else {
				for _, d := range data {
					if tt.expectCMGet {
						a.Equal(tt.expectCMData[dataKey], d)
					}
					if tt.expectSecGet {
						a.Equal(tt.expectSecData[dataKey], []byte(d))
					}
					if tt.expectValGet {
						a.Equal(goodYAML, d)
					}
				}
				a.NoError(err)
			}
		})
	}
}

// TestExtractValueFromOverrideString tests ExtractValueFromOverrideString
// GIVEN an override string
//
//	WHEN I call ExtractValueFromOverrideString
//	THEN I get a value of the specified json path from the override string
func TestExtractValueFromOverrideString(t *testing.T) {
	goodYAML := "foo:\n  foo: bar\n"
	badYAML := "foo:\n  {foo: bar\n"

	tests := []struct {
		name         string
		overrideStr  string
		field        string
		expectError  bool
		expectValGet interface{}
	}{
		{
			name:         "test invalid yaml string",
			overrideStr:  badYAML,
			expectError:  true,
			expectValGet: nil,
		},
		{
			name:         "test valid field",
			overrideStr:  goodYAML,
			field:        "foo",
			expectError:  false,
			expectValGet: map[string]interface{}{"foo": "bar"},
		},
		{
			name:         "test valid nested field",
			overrideStr:  goodYAML,
			field:        "foo.foo",
			expectError:  false,
			expectValGet: "bar",
		},
		{
			name:         "test valid field",
			overrideStr:  goodYAML,
			field:        "test",
			expectError:  false,
			expectValGet: nil,
		},
	}

	a := assert.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ExtractValueFromOverrideString(tt.overrideStr, tt.field)
			if tt.expectError {
				a.Error(err)
			} else {
				a.NoError(err)
				a.Equal(tt.expectValGet, data)
			}
		})
	}
}
