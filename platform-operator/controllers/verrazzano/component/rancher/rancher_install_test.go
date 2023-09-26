// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	adminv1 "k8s.io/api/admissionregistration/v1"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	dynfake "k8s.io/client-go/dynamic/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	dnsSuffix = "DNS"
	name      = "NAME"
)

var GVKCatalog = common.GetRancherMgmtAPIGVKForKind("Catalog")
var GVKNodeDriver = common.GetRancherMgmtAPIGVKForKind("NodeDriver")
var GVKDynamicSchema = common.GetRancherMgmtAPIGVKForKind("DynamicSchema")
var GVKNodeDriverList = common.GetRancherMgmtAPIGVKForKind(GVKNodeDriver.Kind + "List")

func createKontainerDriver(name string) *unstructured.Unstructured {
	kontainerDriver := &unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	kontainerDriver.SetGroupVersionKind(common.GetRancherMgmtAPIGVKForKind(common.KontainerDriverKind))
	kontainerDriver.SetName(name)
	kontainerDriver.UnstructuredContent()["spec"] = map[string]interface{}{}
	return kontainerDriver
}

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
		systemCatalog        = "system-library"
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

	catalog := &unstructured.Unstructured{}
	catalog.SetGroupVersionKind(GVKCatalog)
	catalog.SetName(systemCatalog)

	scheme := getScheme()
	scheme.AddKnownTypeWithName(GVKNodeDriverList, &unstructured.UnstructuredList{})
	fakeDynamicClient := dynfake.NewSimpleDynamicClient(scheme, nodeDriver1, nodeDriver2, dynamicSchemaND1, dynamicSchemaND2, catalog)
	setDynamicClientFunc(func() (dynamic.Interface, error) { return fakeDynamicClient, nil })
	defer func() {
		resetDynamicClientFunc()
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

	// Check that system-library catalog is no longer found
	_, err = fakeDynamicClient.Resource(catalogGVR).Get(ctx, systemCatalog, metav1.GetOptions{})
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

// TestActivateKontainerDriver tests the TestActivateKontainerDriver function
// GIVEN a client with an inactive kontainerdriver
// WHEN  ActivateKontainerDriver is called
// THEN  the kontainerdriver object is activated
func TestActivateKontainerDriver(t *testing.T) {
	// Initialize kontainerdriver object
	driverName := common.KontainerDriverOCIName
	driverObj := createKontainerDriver(driverName)
	driverObj.UnstructuredContent()["spec"].(map[string]interface{})["active"] = false

	// Setup clients and context
	scheme := getScheme()
	scheme.AddKnownTypeWithName(common.GetRancherMgmtAPIGVKForKind(common.KontainerDriverKind), &unstructured.Unstructured{})
	fakeDynamicClient := dynfake.NewSimpleDynamicClient(scheme, driverObj)
	setDynamicClientFunc(func() (dynamic.Interface, error) { return fakeDynamicClient, nil })
	defer func() {
		resetDynamicClientFunc()
	}()

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
	compContext := spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)
	dynClient, err := getDynamicClientFunc()()
	assert.NoError(t, err)
	err = common.ActivateKontainerDriver(compContext, dynClient, driverName)
	assert.NoError(t, err)

	// Fetch the object and confirm it was updated
	gvr := common.GetRancherMgmtAPIGVRForResource(common.KontainerDriversResourceName)
	kdObj, err := fakeDynamicClient.Resource(gvr).Get(context.TODO(), driverName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, kdObj.UnstructuredContent()["spec"].(map[string]interface{})["active"].(bool))
}

// TestUpdateKontainerDriverURLs tests the TestUpdateKontainerDriverURLs function
// GIVEN a client that has it's dns domain updated
// WHEN  UpdateKontainerDriverURLs is called
// THEN  the driver URLs are updated
func TestUpdateKontainerDriverURLs(t *testing.T) {
	initialURL := "https://test.domain1.io/driver/test.yaml"
	expectedURL := "https://test.domain2.io/driver/test.yaml"

	// Initialize kontainerdriver and ingress objects
	driverObj1Name := common.KontainerDriverOCIName
	driverObj1 := createKontainerDriver(driverObj1Name)
	driverObj1.UnstructuredContent()["spec"].(map[string]interface{})["url"] = initialURL

	driverObj2Name := common.KontainerDriverOKEName
	driverObj2 := createKontainerDriver(driverObj2Name)
	driverObj2.UnstructuredContent()["spec"].(map[string]interface{})["url"] = initialURL

	driverObj3Name := common.KontainerDriverOKECAPIName
	driverObj3 := createKontainerDriver(driverObj3Name)
	driverObj3.UnstructuredContent()["spec"].(map[string]interface{})["url"] = initialURL

	ingress := &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   common.CattleSystem,
			Name:        common.RancherName,
			Annotations: map[string]string{"cert-manager.io/common-name": "test.domain2.io"},
		},
	}

	// Setup clients and context
	scheme := getScheme()
	scheme.AddKnownTypeWithName(common.GetRancherMgmtAPIGVKForKind(common.KontainerDriverKind), &unstructured.Unstructured{})
	fakeDynamicClient := dynfake.NewSimpleDynamicClient(scheme, driverObj1, driverObj2, driverObj3)
	setDynamicClientFunc(func() (dynamic.Interface, error) { return fakeDynamicClient, nil })
	defer func() {
		resetDynamicClientFunc()
	}()

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ingress).Build()
	compContext := spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)
	dynClient, err := getDynamicClientFunc()()
	assert.NoError(t, err)
	err = common.UpdateKontainerDriverURLs(compContext, dynClient)
	assert.NoError(t, err)

	// Fetch the objects and confirm they were updated
	gvr := common.GetRancherMgmtAPIGVRForResource(common.KontainerDriversResourceName)

	kdObj1, err := fakeDynamicClient.Resource(gvr).Get(context.TODO(), driverObj1Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, expectedURL, kdObj1.UnstructuredContent()["spec"].(map[string]interface{})["url"].(string))

	kdObj2, err := fakeDynamicClient.Resource(gvr).Get(context.TODO(), driverObj2Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, expectedURL, kdObj2.UnstructuredContent()["spec"].(map[string]interface{})["url"].(string))

	kdObj3, err := fakeDynamicClient.Resource(gvr).Get(context.TODO(), driverObj3Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, expectedURL, kdObj3.UnstructuredContent()["spec"].(map[string]interface{})["url"].(string))
}
