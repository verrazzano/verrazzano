// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/stretchr/testify/assert"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	ocifake "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

var (
	scheme *runtime.Scheme
	//go:embed testdata/base.yaml
	testBase []byte
	//go:embed testdata/existing-vcn-patch.yaml
	testExistingVCNPatch []byte
	//go:embed testdata/new-vcn-patch.yaml
	testNewVCNPatch []byte
	//go:embed testdata/completed-patch.yaml
	testCompletedPatch []byte
	//go:embed testdata/new-vcn-private-registry.yaml
	testNewVCNPrivateRegistryPatch []byte
	testOCNEVersions               = "../controller/ocne/testdata/ocne-versions.yaml"
	testLoader                     = &ocifake.CredentialsLoaderImpl{
		Credentials: &oci.Credentials{
			Region:  "",
			Tenancy: "a",
			User:    "b",
			PrivateKey: `abc
def
ghi
`,
			Fingerprint:          "d",
			Passphrase:           "e",
			UseInstancePrincipal: "false",
		},
	}
	testOCIClientGetter = func(creds *oci.Credentials) (oci.Client, error) {
		return &ocifake.ClientImpl{
			AvailabilityDomains: []oci.AvailabilityDomain{
				{
					Name: "x",
					FaultDomains: []oci.FaultDomain{
						{Name: "y"},
					},
				},
			},
		}, nil
	}
	privateRegistrySecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRegistrySecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte("foo"),
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}
)

const (
	testNamespace          = "test"
	testName               = testNamespace
	testRegistrySecretName = "registry"
)

func init() {
	scheme = runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vmcv1alpha1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
}

func testCreateCR(patch []byte) (*vmcv1alpha1.OCNEOCIQuickCreate, error) {
	baseCR := &vmcv1alpha1.OCNEOCIQuickCreate{}
	patchCR := &vmcv1alpha1.OCNEOCIQuickCreate{}
	if err := yaml.Unmarshal(testBase, baseCR); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(patch, patchCR); err != nil {
		return nil, err
	}
	baseCR.Spec = patchCR.Spec
	baseCR.Status = patchCR.Status
	return baseCR, nil
}

func testOCNEConfigMap() *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	b, _ := os.ReadFile(testOCNEVersions)
	_ = yaml.Unmarshal(b, cm)
	return cm
}

func testReconciler(cli clipkg.Client) *ClusterReconciler {
	return &ClusterReconciler{
		Base: &controller.Base{
			Client: cli,
		},
		Scheme:            scheme,
		CredentialsLoader: testLoader,
		OCIClientGetter:   testOCIClientGetter,
	}
}

func getTestCR(cli clipkg.Client) (*vmcv1alpha1.OCNEOCIQuickCreate, error) {
	ctx := context.TODO()
	q := &vmcv1alpha1.OCNEOCIQuickCreate{}
	err := cli.Get(ctx, types.NamespacedName{
		Namespace: testNamespace,
		Name:      testName,
	}, q)

	return q, err
}

func TestReconcile(t *testing.T) {
	existingVCNCR, err := testCreateCR(testExistingVCNPatch)
	assert.NoError(t, err)
	completedCR, err := testCreateCR(testCompletedPatch)
	assert.NoError(t, err)
	noFinalizerCR := existingVCNCR.DeepCopy()
	noFinalizerCR.Finalizers = nil
	provisioningCR, err := testCreateCR(testNewVCNPatch)
	assert.NoError(t, err)
	privateRegistryCR, err := testCreateCR(testNewVCNPrivateRegistryPatch)
	assert.NoError(t, err)
	notFoundReconciler := testReconciler(fake.NewClientBuilder().WithScheme(scheme).Build())
	quickCreateReconciler := testReconciler(fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(existingVCNCR, testOCNEConfigMap()).WithObjects(existingVCNCR, testOCNEConfigMap()).Build())
	completedReconciler := testReconciler(fake.NewClientBuilder().WithScheme(scheme).WithObjects(completedCR, testOCNEConfigMap()).Build())
	noFinalizerReconciler := testReconciler(fake.NewClientBuilder().WithScheme(scheme).WithObjects(noFinalizerCR, testOCNEConfigMap()).Build())
	provisioningReconciler := testReconciler(fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(provisioningCR, testOCNEConfigMap()).WithObjects(provisioningCR, testOCNEConfigMap()).Build())
	privateRegistryReconciler := testReconciler(fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(privateRegistrySecret, privateRegistryCR, testOCNEConfigMap()).WithObjects(privateRegistrySecret, privateRegistryCR, testOCNEConfigMap()).Build())
	var tests = []struct {
		name        string
		reconciler  *ClusterReconciler
		assertsFunc func(t *testing.T)
	}{
		{
			"nothing to do when resource not found",
			notFoundReconciler,
			func(t *testing.T) {},
		},
		{
			"when quick creating, a CAPI cluster is applied",
			quickCreateReconciler,
			func(t *testing.T) {
				ctx := context.TODO()
				assert.NoError(t, quickCreateReconciler.Client.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      testName,
				}, &v1beta1.Cluster{}))
			},
		},
		{
			"completed CRs are deleted",
			completedReconciler,
			func(t *testing.T) {
				q, err := getTestCR(completedReconciler.Client)
				assert.NoError(t, err)
				assert.False(t, q.GetDeletionTimestamp().IsZero())
			},
		},
		{
			"finalizers are added if not present",
			noFinalizerReconciler,
			func(t *testing.T) {
				q, _ := getTestCR(noFinalizerReconciler.Client)
				assert.NotNil(t, q)
				assert.NotNil(t, q.Finalizers)
				assert.Equal(t, finalizerKey, q.Finalizers[0])
			},
		},
		{
			"provisoning CR creates addon resources",
			provisioningReconciler,
			func(t *testing.T) {
				err := provisioningReconciler.Client.Get(context.TODO(), types.NamespacedName{
					Name:      fmt.Sprintf("%s-csi", testName),
					Namespace: testNamespace,
				}, &corev1.Secret{})
				assert.NoError(t, err)
			},
		},
		{
			"private registry secret is created when using private registry credentials",
			privateRegistryReconciler,
			func(t *testing.T) {
				err := privateRegistryReconciler.Client.Get(context.TODO(), types.NamespacedName{
					Name:      fmt.Sprintf("%s-image-pull-secret", testName),
					Namespace: testNamespace,
				}, &corev1.Secret{})
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.reconciler.Reconcile(context.TODO(), ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: testNamespace,
					Name:      testName,
				},
			})
			assert.NoError(t, err)
			tt.assertsFunc(t)
		})
	}
}
