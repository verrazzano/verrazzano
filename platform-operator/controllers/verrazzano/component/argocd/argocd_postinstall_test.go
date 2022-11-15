// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestPatchArgoCDConfigMap should add the oidc configuration to enable our keycloak authentication
// GIVEN a ArgoCD config map
//
//	WHEN TestPatchArgoCDConfigMap is called
//	THEN TestPatchArgoCDRbacConfigMap should patch the cm with oidc config noted above.
func TestPatchArgoCDConfigMap(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tls-argocd-ingress",
			Namespace: constants.ArgoCDNamespace,
		},
		Data: map[string][]byte{
			"ca.crt": []byte("foobar"),
		},
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-cm",
			Namespace: constants.ArgoCDNamespace,
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(configMap, secret).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false)
	assert.NoError(t, patchArgoCDConfigMap(ctx))
}

// TestPatchArgoCDRbacConfigMap should apply a policy that grants admin (role:admin) to verrazzano-admins group
// GIVEN a ArgoCD rbac config map
//
//	WHEN TestPatchArgoCDRbacConfigMap is called
//	THEN TestPatchArgoCDRbacConfigMap should patch the cm with a policy noted above
func TestPatchArgoCDRbacConfigMap(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	rbaccm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-rbac-cm",
			Namespace: constants.ArgoCDNamespace,
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(rbaccm).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false)
	assert.NoError(t, patchArgoCDRbacConfigMap(ctx))
}
