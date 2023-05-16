// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type FakeArgoClientSecretProvider struct{}

func (f FakeArgoClientSecretProvider) GetClientSecret(_ spi.ComponentContext) (string, error) {
	return "blah", nil
}

const (
	policyWithVzadminTop = `g, verrazzano-admins, role:admin
p, role:staging-dev-admins, applications, create, dev-project/*, allow
p, role:staging-dev-admins, applications, get, dev-project/*, allow
p, role:staging-dev-admins, applications, override, dev-project/*, allow
p, role:staging-dev-admins, applications, sync, dev-project/*, allow
p, role:staging-dev-admins, applications, update, dev-project/*, allow
p, role:staging-dev-admins, exec, create, dev-project/*, allow
g, dev-project-key, role:staging-dev-admins`

	policyWithVzadminMiddle = `p, role:staging-dev-admins, applications, create, dev-project/*, allow
p, role:staging-dev-admins, applications, get, dev-project/*, allow
p, role:staging-dev-admins, applications, override, dev-project/*, allow
p, role:staging-dev-admins, applications, sync, dev-project/*, allow
g, verrazzano-admins, role:admin
p, role:staging-dev-admins, applications, update, dev-project/*, allow
p, role:staging-dev-admins, exec, create, dev-project/*, allow
g, dev-project-key, role:staging-dev-admins`

	policyWithVzadminBottom = `p, role:staging-dev-admins, applications, create, dev-project/*, allow
p, role:staging-dev-admins, applications, get, dev-project/*, allow
p, role:staging-dev-admins, applications, override, dev-project/*, allow
p, role:staging-dev-admins, applications, sync, dev-project/*, allow
p, role:staging-dev-admins, applications, update, dev-project/*, allow
p, role:staging-dev-admins, exec, create, dev-project/*, allow
g, dev-project-key, role:staging-dev-admins
g, verrazzano-admins, role:admin`

	policyWithVzadminMissing = `p, role:staging-dev-admins, applications, create, dev-project/*, allow
p, role:staging-dev-admins, applications, get, dev-project/*, allow
p, role:staging-dev-admins, applications, override, dev-project/*, allow
p, role:staging-dev-admins, applications, sync, dev-project/*, allow
p, role:staging-dev-admins, applications, update, dev-project/*, allow
p, role:staging-dev-admins, exec, create, dev-project/*, allow
g, dev-project-key, role:staging-dev-admins`

	policyWithVzadminOnly = `g, verrazzano-admins, role:admin`
)

// TestArgoCDRBAC should apply a policy that grants admin (role:admin) to verrazzano-admins group
// GIVEN a ArgoCD rbac config map
//
//	WHEN patchArgoCDRbacConfigMap is called for various cases
//	THEN the argocd-rbac-cm config map is patched correctly
func TestArgoCDRBAC(t *testing.T) {
	const cmName = "argocd-rbac-cm"

	tests := []struct {
		name         string
		initialRBAC  string
		expectedRBAC string
	}{
		{name: "test-empty", initialRBAC: "", expectedRBAC: policyWithVzadminOnly},
		{name: "test-missing", initialRBAC: policyWithVzadminMissing, expectedRBAC: policyWithVzadminBottom},
		{name: "test-top", initialRBAC: policyWithVzadminTop, expectedRBAC: policyWithVzadminTop},
		{name: "test-middle", initialRBAC: policyWithVzadminMiddle, expectedRBAC: policyWithVzadminMiddle},
		{name: "test-bottom", initialRBAC: policyWithVzadminBottom, expectedRBAC: policyWithVzadminBottom},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rbaccm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: constants.ArgoCDNamespace,
				},
				Data: map[string]string{policyCSVKey: test.initialRBAC},
			}
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(rbaccm).Build()
			ctx := spi.NewFakeContext(fakeClient, nil, nil, false)
			assert.NoError(t, patchArgoCDRbacConfigMap(ctx))

			cm := corev1.ConfigMap{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Immutable:  nil,
				Data:       nil,
				BinaryData: nil,
			}
			err := fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: constants.ArgoCDNamespace, Name: cmName}, &cm)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedRBAC, cm.Data[policyCSVKey])
		})
	}
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

	keycloakIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: constants.KeycloakNamespace,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "keycloak",
				},
			},
		},
	}

	argoCDIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.ArgoCDIngress,
			Namespace: constants.ArgoCDNamespace,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "keycloak",
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(configMap, secret, keycloakIngress, argoCDIngress).Build()
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
