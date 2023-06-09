// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestGetCapiOverrides tests getting the override values for the Cluster API component
// GIVEN a call to getCapiOverrides
//
//	WHEN all env variables are set to the correct values
//	THEN true is returned
func TestGetCapiOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	config.TestHelmConfigDir = "../../../../helm_config"

	overrides, err := getCapiOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, overrides)

	// Check that expected values are loaded into the struct
	assert.Equal(t, "ghcr.io", overrides.Global.Registry)
	assert.Equal(t, corev1.PullIfNotPresent, overrides.Global.PullPolicy)

	bootstrapImage := overrides.DefaultProviders.OCNE.Bootstrap.Image
	assert.Equal(t, "verrazzano", bootstrapImage.Repository)
	assert.Equal(t, "v0.1.0-20230427222244-4ef1141", bootstrapImage.Tag)

	controlPlaneImage := overrides.DefaultProviders.OCNE.ControlPlane.Image
	assert.Equal(t, "verrazzano", controlPlaneImage.Repository)
	assert.Equal(t, "v0.1.0-20230427222244-4ef1141", controlPlaneImage.Tag)

	coreImage := overrides.DefaultProviders.Core.Image
	assert.Equal(t, "verrazzano", coreImage.Repository)
	assert.Equal(t, "v1.3.3-20230427222746-876fe3dc9", coreImage.Tag)

	ociImage := overrides.DefaultProviders.OCI.Image
	assert.Equal(t, "oracle", ociImage.Repository)
	assert.Equal(t, "v0.8.1", ociImage.Tag)
}
