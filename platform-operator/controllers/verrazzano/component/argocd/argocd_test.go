// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	admv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	vzDefaultCA = vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "DefaultCA",
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{Certificate: vzapi.Certificate{CA: vzapi.CA{
					SecretName:               constants.DefaultVerrazzanoCASecretName,
					ClusterResourceNamespace: defaultSecretNamespace,
				}}},
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{Suffix: common.ArgoCDName},
				},
			},
		},
	}
)

func getScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = networking.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	_ = certv1.AddToScheme(scheme)
	_ = admv1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	_ = v12.AddToScheme(scheme)
	return scheme
}

// TestBuildArgoCDDNSNames asserts if the generated DNS name for Argo CD is correct.
func TestBuildArgoCDDNSNames(t *testing.T) {
	// GIVEN a Verrazzano CR with Argo CD Component enabled,
	// WHEN we call the buildArgoCDHostNameForDomain function,
	// THEN correct FQDN for ArgoCD is returned.
	argoCDDNSName := buildArgoCDHostNameForDomain("default.nip.io")
	assert.Equal(t, "argocd.default.nip.io", argoCDDNSName)
}

func TestRemoveArgoResources(t *testing.T) {
	appResource := &unstructured.Unstructured{}
	appResource.SetGroupVersionKind(common.GetArgoProjAPIGVRForResource(common.ArgoCDKindApplication))
	appResource.SetName("test app")
	appResource.SetNamespace("argocd")
	appResource.SetFinalizers([]string{"finalizer1", "finalizer2"})

	appSetResource := &unstructured.Unstructured{}
	appSetResource.SetGroupVersionKind(common.GetArgoProjAPIGVRForResource(common.ArgoCDKindApplicationSet))
	appSetResource.SetName("test app")
	appSetResource.SetNamespace("argocd")
	appSetResource.SetFinalizers([]string{"finalizer1"})

	projectResource := &unstructured.Unstructured{}
	projectResource.SetGroupVersionKind(common.GetArgoProjAPIGVRForResource(common.ArgoCDKindAppProject))
	projectResource.SetName("test app")
	projectResource.SetNamespace("argocd")
	projectResource.SetFinalizers([]string{"finalizer1"})

	fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects(appResource, appSetResource, projectResource).Build()

	assert.NoError(t, removeArgoResources(spi.NewFakeContext(fakeClient, nil, nil, false)))
}
