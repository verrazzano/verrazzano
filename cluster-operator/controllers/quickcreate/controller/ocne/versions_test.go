// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocne

import (
	ctx "context"
	_ "embed"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

var (
	//go:embed testdata/ocne-versions.yaml
	testConfigMapBytes []byte
	testOCNEVersion    = "1.6"
)

func TestGetVersionDefaults(t *testing.T) {
	cm := &corev1.ConfigMap{}
	_ = yaml.Unmarshal(testConfigMapBytes, cm)
	cli := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cm).Build()

	cmNoData := cm.DeepCopy()
	cmNoData.Data = nil
	cmNoMapping := cmNoData.DeepCopy()
	cmNoData.Data = map[string]string{
		"invalid": "x",
	}
	cmNoVersions := cmNoData.DeepCopy()
	cmNoVersions.Data = map[string]string{
		"mapping": "{}",
	}

	var tests = []struct {
		name        string
		ocneVersion string
		cli         clipkg.Client
		hasError    bool
	}{
		{
			"no error for valid OCNE Version",
			testOCNEVersion,
			cli,
			false,
		},
		{

			"error when invalid OCNE Version",
			"invalid version",
			cli,
			true,
		},
		{
			"error when no ocne metadata",
			testOCNEVersion,
			fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
			true,
		},
		{
			"error when no ocne data in configmap",
			testOCNEVersion,
			fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cmNoData).Build(),
			true,
		},
		{
			"error when no ocne mapping in configmap",
			testOCNEVersion,
			fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cmNoMapping).Build(),
			true,
		},
		{
			"error when no ocne versions in configmap",
			testOCNEVersion,
			fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cmNoVersions).Build(),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versions, err := GetVersionDefaults(ctx.TODO(), tt.cli, tt.ocneVersion)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.ocneVersion, versions.Release)
			}
		})
	}
}
