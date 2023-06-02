// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	adminv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	dynfake "k8s.io/client-go/dynamic/fake"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	dnsSuffix = "DNS"
	name      = "NAME"
)

var GVKNodeDriver = common.GetRancherMgmtAPIGVKForKind("NodeDriver")
var GVKDynamicSchema = common.GetRancherMgmtAPIGVKForKind("DynamicSchema")
var GVKNodeDriverList = common.GetRancherMgmtAPIGVKForKind(GVKNodeDriver.Kind + "List")

// TestAddAcmeIngressAnnotations verifies if LetsEncrypt Annotations are added to the Ingress
// GIVEN a Rancher Ingress
//
//	WHEN addAcmeIngressAnnotations is called
//	THEN addAcmeIngressAnnotations should annotate the ingress
func TestAddAcmeIngressAnnotations(t *testing.T) {
	in := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}
	out := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/auth-realm":  fmt.Sprintf("%s auth", dnsSuffix),
				"external-dns.alpha.kubernetes.io/target": fmt.Sprintf("verrazzano-ingress.%s.%s", name, dnsSuffix),
				"external-dns.alpha.kubernetes.io/ttl":    "60",
			},
		},
	}

	addAcmeIngressAnnotations(name, dnsSuffix, &in)
	assert.Equal(t, out, in)
}

// TestAddCAIngressAnnotations verifies if CA Annotations are added to the Ingress
// GIVEN a Rancher Ingress
//
//	WHEN addCAIngressAnnotations is called
//	THEN addCAIngressAnnotations should annotate the ingress
func TestAddCAIngressAnnotations(t *testing.T) {
	in := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}
	out := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/auth-realm": fmt.Sprintf("%s.%s auth", name, dnsSuffix),
			},
		},
	}

	addCAIngressAnnotations(name, dnsSuffix, &in)
	assert.Equal(t, out, in)
}

// TestPatchRancherIngress should annotate the Rancher ingress with Acme/Private CA values
// GIVEN a Rancher Ingress and a Verrazzano CR
//
//	WHEN patchRancherIngress is called
//	THEN patchRancherIngress should annotate the ingress according to the Verrazzano CR
func TestPatchRancherIngress(t *testing.T) {
	ingress := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   common.CattleSystem,
			Name:        common.RancherName,
			Annotations: map[string]string{"test": "data"},
		},
	}
	var tests = []struct {
		in    networking.Ingress
		vzapi vzapi.Verrazzano
	}{
		{ingress, vzAcmeDev},
		{ingress, vzDefaultCA},
	}

	for _, tt := range tests {
		c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&tt.in).Build()
		t.Run(tt.vzapi.Spec.EnvironmentName, func(t *testing.T) {
			// Create a fake ComponentContext with the profiles dir to create an EffectiveCR; this is required to
			// convert the legacy CertManager config to the ClusterIssuer config
			ctx := spi.NewFakeContext(c, &tt.vzapi, nil, false, profilesRelativePath)

			assert.Nil(t, patchRancherIngress(c, ctx.EffectiveCR()))
		})
	}
}

// TestPatchRancherIngressNotFound should fail to find the ingress
// GIVEN no Rancher Ingress and a Verrazzano CR
//
//	WHEN patchRancherIngress is called
//	THEN patchRancherIngress should fail to annotate the Ingress
func TestPatchRancherIngressNotFound(t *testing.T) {
	// Create a fake ComponentContext with the profiles dir to create an EffectiveCR; this is required to
	// convert the legacy CertManager config to the ClusterIssuer config
	c := fake.NewClientBuilder().WithScheme(getScheme()).Build()
	ctx := spi.NewFakeContext(c, &vzAcmeDev, nil, false, profilesRelativePath)

	err := patchRancherIngress(c, ctx.EffectiveCR())
	assert.NotNil(t, err)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestCleanupRancherResources(t *testing.T) {
	const (
		nd1                  = "nd1"
		nd2                  = "nd2"
		dynamicSchemaND2Name = "ds2"
	)

	nodeDriver1 := &unstructured.Unstructured{}
	nodeDriver1.SetGroupVersionKind(GVKNodeDriver)
	nodeDriver1.SetName(nd1)
	nodeDriver2 := nodeDriver1.DeepCopy()
	nodeDriver2.SetName(nd2)

	// dynamic schema with owner reference that should be removed
	dynamicSchemaND1 := &unstructured.Unstructured{}
	dynamicSchemaND1.SetGroupVersionKind(GVKDynamicSchema)
	dynamicSchemaND1.SetName(ociSchemaName)
	dynamicSchemaND1.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion: nodeDriver1.GetAPIVersion(),
			Kind:       nodeDriver1.GetKind(),
			Name:       nodeDriver1.GetName(),
			UID:        "xyz",
		},
	})

	// dynamic schema with owner reference that should be preserved, and the schema deleted
	dynamicSchemaND2 := dynamicSchemaND1.DeepCopy()
	dynamicSchemaND1.SetName(dynamicSchemaND2Name)
	dynamicSchemaND1.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion: nodeDriver2.GetAPIVersion(),
			Kind:       nodeDriver2.GetKind(),
			Name:       nodeDriver2.GetName(),
			UID:        "abc",
		},
	})

	scheme := getScheme()
	scheme.AddKnownTypeWithName(GVKNodeDriverList, &unstructured.UnstructuredList{})
	fakeDynamicClient := dynfake.NewSimpleDynamicClient(scheme, nodeDriver1, nodeDriver2, dynamicSchemaND1, dynamicSchemaND2)
	prevGetDynamicClientFunc := getDynamicClientFunc
	getDynamicClientFunc = func() (dynamic.Interface, error) { return fakeDynamicClient, nil }
	defer func() {
		getDynamicClientFunc = prevGetDynamicClientFunc
	}()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&adminv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: CAPIValidatingWebhook,
		},
	}, &adminv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: CAPIMutatingWebhook,
		},
	}).Build()
	ctx := context.TODO()
	err := cleanupRancherResources(ctx, fakeClient)
	assert.NoError(t, err)

	// Check node drivers are no longer found
	_, err = fakeDynamicClient.Resource(nodeDriverGVR).Get(ctx, nd1, metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err))
	_, err = fakeDynamicClient.Resource(nodeDriverGVR).Get(ctx, nd2, metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err))

	// Check Rancher CAPI webhooks are no longer found
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name: CAPIValidatingWebhook,
	}, &adminv1.ValidatingWebhookConfiguration{})
	assert.True(t, apierrors.IsNotFound(err))
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name: CAPIMutatingWebhook,
	}, &adminv1.MutatingWebhookConfiguration{})
	assert.True(t, apierrors.IsNotFound(err))

	ds1, err := fakeDynamicClient.Resource(dynamicSchemaGVR).Get(ctx, ociSchemaName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Len(t, ds1.GetOwnerReferences(), 0)
	// Check schemas are deleted/preserved according to their owner references
	ds2, err := fakeDynamicClient.Resource(dynamicSchemaGVR).Get(ctx, dynamicSchemaND2Name, metav1.GetOptions{})
	assert.NoError(t, err)
	// Cascading delete does not happen with fake client, so we check if owner reference is still present
	assert.NotNil(t, ds2.GetOwnerReferences())
}
