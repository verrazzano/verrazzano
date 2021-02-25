// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const kubeAdminData = `
apiEndpoints:
  oke-xyz:
    advertiseAddress: 1.2.3.4
    bindPort: 6443
`

// TestGetApiServerURL tests the validation of a API server URL creation
// GIVEN a call validate GetApiServerURL
// WHEN the kubeadmin configmap has the API server URL information
// THEN the correct API URL shoule be returned
func TestGetApiServerURL(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeAdminConfig,
			Namespace: KubeSystem,
		},
		Data: map[string]string{
			ClusterStatusKey: kubeAdminData,
		},
	})

	url, err := GetApiServerURL(client)
	assert.NoError(t, err, "Error validating VerrazzanoMultiCluster resource")
	assert.Equal(t, url, "https://1.2.3.4:6443")
}
