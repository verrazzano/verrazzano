// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocne

import (
	ctx "context"
	_ "embed"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
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
	var tests = []struct {
		ocneVersion string
		hasError    bool
	}{
		{
			testOCNEVersion,
			false,
		},
		{
			"invalid version",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.ocneVersion, func(t *testing.T) {
			versions, err := GetVersionDefaults(ctx.TODO(), cli, tt.ocneVersion)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.ocneVersion, versions.Release)
			}
		})
	}
}
