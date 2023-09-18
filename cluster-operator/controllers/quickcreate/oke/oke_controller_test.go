// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oke

import (
	"context"
	_ "embed"
	"github.com/stretchr/testify/assert"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	ocifake "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

const (
	testNamespace = "test"
	testName      = testNamespace
)

var (
	scheme *runtime.Scheme
	//go:embed testdata/base.yaml
	testBase []byte
	//go:embed testdata/completed-patch.yaml
	testCompleted []byte
	//go:embed testdata/existing-vcn-patch.yaml
	testExistingVCN []byte
	//go:embed testdata/new-vcn-native-patch.yaml
	testNewVCN []byte
	//go:embed testdata/existing-vcn-provisioning.yaml
	testProvisioning []byte
	testLoader       = &ocifake.CredentialsLoaderImpl{
		Credentials: &oci.Credentials{},
	}
	testOCIClientGetter = func(_ *oci.Credentials) (oci.Client, error) {
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
)

func init() {
	scheme = runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vmcv1alpha1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
}

func TestReconcile(t *testing.T) {
	existingVCNCR, err := testCreateCR(testExistingVCN)
	assert.NoError(t, err)
	completedCR, err := testCreateCR(testCompleted)
	assert.NoError(t, err)
	newVCNCR, err := testCreateCR(testNewVCN)
	assert.NoError(t, err)
	provisioningCR, err := testCreateCR(testProvisioning)
	assert.NoError(t, err)

	notFoundReconciler := testReconciler(fake.NewClientBuilder().WithScheme(scheme).Build())
	existingVCNReconciler := testReconciler(fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(existingVCNCR).WithObjects(existingVCNCR).Build())
	completedReconciler := testReconciler(fake.NewClientBuilder().WithScheme(scheme).WithObjects(completedCR).Build())
	newVCNReconciler := testReconciler(fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(newVCNCR).WithObjects(newVCNCR).Build())
	provisioningReconciler := testReconciler(fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(provisioningCR).WithObjects(provisioningCR).Build())

	var tests = []struct {
		name        string
		reconciler  *ClusterReconciler
		assertsFunc func(t *testing.T)
	}{
		{
			"no error when resource not found",
			notFoundReconciler,
			func(t *testing.T) {},
		},
		{
			"create a cluster when using an existing VCN",
			existingVCNReconciler,
			func(t *testing.T) {
				ctx := context.TODO()
				assert.NoError(t, existingVCNReconciler.Client.Get(ctx, types.NamespacedName{
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
			"create a cluster when creating a new VCN",
			newVCNReconciler,
			func(t *testing.T) {
				ctx := context.TODO()
				assert.NoError(t, newVCNReconciler.Client.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      testName,
				}, &v1beta1.Cluster{}))
			},
		},
		{
			"quick create moves to completed when reconciling provisioning",
			provisioningReconciler,
			func(t *testing.T) {
				q, err := getTestCR(completedReconciler.Client)
				assert.NoError(t, err)
				assert.Equal(t, vmcv1alpha1.QuickCreatePhaseComplete, q.Status.Phase)
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

func testCreateCR(patch []byte) (*vmcv1alpha1.OKEQuickCreate, error) {
	baseCR := &vmcv1alpha1.OKEQuickCreate{}
	patchCR := &vmcv1alpha1.OKEQuickCreate{}
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

func getTestCR(cli clipkg.Client) (*vmcv1alpha1.OKEQuickCreate, error) {
	ctx := context.TODO()
	q := &vmcv1alpha1.OKEQuickCreate{}
	err := cli.Get(ctx, types.NamespacedName{
		Namespace: testNamespace,
		Name:      testName,
	}, q)

	return q, err
}
