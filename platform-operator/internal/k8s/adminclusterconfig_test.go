// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testServerData = "https://testurl"

// TestGetApiServerURL tests the validation of a API server URL creation
// GIVEN a call validate GetAPIServerURL
// WHEN the verrazzano-admin-cluster configmap has the server data
// THEN the correct API URL shoule be returned
func TestGetApiServerURL(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.VerrazzanoMultiClusterNamespace,
				Name:      constants.AdminClusterConfigMapName,
			},
			Data: map[string]string{
				constants.ServerDataKey: testServerData,
			},
		},
	).Build()

	url, err := GetAPIServerURL(client)
	assert.NoError(t, err, "Error validating VerrazzanoMultiCluster resource")
	assert.Equal(t, testServerData, url, "expected URL")
}

// TestGetApiServerURLErr tests the validation of a API server URL creation
// GIVEN a call validate GetAPIServerURL
// WHEN the verrazzano-admin-cluster configmap has the server data
// THEN the correct API URL shoule be returned
func TestGetApiServerURLErr(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.VerrazzanoMultiClusterNamespace,
				Name:      constants.AdminClusterConfigMapName,
			},
			Data: map[string]string{
				constants.ServerDataKey: testServerData,
			},
		},
	).Build()

	url, err := GetAPIServerURL(client)
	assert.NoError(t, err, "Error validating VerrazzanoMultiCluster resource")
	assert.Equal(t, testServerData, url, "expected URL")
}
