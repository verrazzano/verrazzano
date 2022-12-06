// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"context"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

type FakeArgoClientSecretProvider struct{}

func (f FakeArgoClientSecretProvider) GetClientSecret(_ spi.ComponentContext) (string, error) {
	return "blah", nil
}

// TestPatchArgoCDSecret should add the oidc configuration client secret for keycloak
// GIVEN a ArgoCD secret
//
//	WHEN TestPatchArgoCDSecret is called
//	THEN TestPatchArgoCDSecret should patch the secret with oidc client secret.
func TestPatchArgoCDSecret(t *testing.T) {
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
			Name:      "argocd-secret",
			Namespace: constants.ArgoCDNamespace,
		},
		Data: map[string][]byte{
			"oidc.keycloak.clientSecret": []byte("blah"),
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(secret).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false)

	component := NewComponent().(argoCDComponent)
	component.ArgoClientSecretProvider = FakeArgoClientSecretProvider{}
	assert.NoError(t, patchArgoCDSecret(component, ctx))
	patchedSecret := &corev1.Secret{}
	err := fakeClient.Get(context.TODO(), types.NamespacedName{
		Name:      "argocd-secret",
		Namespace: constants.ArgoCDNamespace,
	}, patchedSecret)

	assert.NoError(t, err)
	assert.Equal(t, []byte("blah"), patchedSecret.Data["oidc.keycloak.clientSecret"])
}

// TestPatchArgoCDConfigMap should add the oidc configuration to enable our keycloak authentication
// GIVEN a ArgoCD config map
//
//	WHEN TestPatchArgoCDConfigMap is called
//	THEN TestPatchArgoCDConfigMap should patch the cm with oidc config noted above.
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
