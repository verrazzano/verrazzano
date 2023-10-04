// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/capi"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testCAPIClusterName = "test-capi-cluster"
	testCAPINamespace   = "c-12345"
	testVMCName         = "test-vmc"
	testVMCNamespace    = "verrazzano-mc"
)

// TestUpdateStatus tests the updateStatus function
func TestUpdateStatus(t *testing.T) {
	// clear any cached user auth tokens when the test completes
	defer rancherutil.DeleteStoredTokens()

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect the requests for the existing VMC resource
	mock.EXPECT().Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: testManagedCluster}, gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, nsn types.NamespacedName, vmc *v1alpha1.VerrazzanoManagedCluster, opts ...client.GetOption) error {
			return nil
		}).AnyTimes()

	// GIVEN a VMC with a status state unset and the last agent connect time set
	// WHEN the updateStatus function is called
	// THEN the status state is updated to pending
	mock.EXPECT().Status().Return(mockStatus)
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *v1alpha1.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(v1alpha1.StatePending, vmc.Status.State)
			return nil
		})

	vmc := v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testManagedCluster,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Status: v1alpha1.VerrazzanoManagedClusterStatus{
			LastAgentConnectTime: &metav1.Time{
				Time: time.Now(),
			},
		},
	}
	reconciler := newVMCReconciler(mock)
	reconciler.log = vzlog.DefaultLogger()

	err := reconciler.updateStatus(context.TODO(), &vmc)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)

	// GIVEN a VMC with a status state of pending and the last agent connect time set
	// WHEN the updateStatus function is called
	// THEN the status state is updated to active
	mock.EXPECT().Status().Return(mockStatus)
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *v1alpha1.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(v1alpha1.StateActive, vmc.Status.State)
			return nil
		})

	err = reconciler.updateStatus(context.TODO(), &vmc)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)

	// GIVEN a VMC with a last agent connect time set in the past
	// WHEN the updateStatus function is called
	// THEN the status state is updated to inactive
	past := metav1.Unix(0, 0)
	vmc.Status.LastAgentConnectTime = &past

	// Expect the Rancher registration status to be set appropriately
	mock.EXPECT().Status().Return(mockStatus)
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *v1alpha1.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(v1alpha1.StateInactive, vmc.Status.State)
			return nil
		})

	err = reconciler.updateStatus(context.TODO(), &vmc)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
}

// TestUpdateStateCAPI tests that updateState correctly updates the status.state of the VMC when
// the VMC has a reference to a CAPI cluster
func TestUpdateStateCAPI(t *testing.T) {
	a := assert.New(t)
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	vmc := newVMCWithClusterRef(testVMCName, testVMCNamespace, testCAPIClusterName, testCAPINamespace)

	tests := []struct {
		testName         string
		capiCluster      *unstructured.Unstructured
		capiPhase        string
		expectedVMCState string
		err              error
	}{
		{
			"valid CAPI phase",
			newCAPICluster(testCAPIClusterName, testCAPINamespace),
			string(v1alpha1.StateProvisioned),
			string(v1alpha1.StateProvisioned),
			nil,
		},
		{
			"empty CAPI phase",
			newCAPICluster(testCAPIClusterName, testCAPINamespace),
			"",
			string(v1alpha1.StateUnknown),
			nil,
		},
		{
			"nonexistent CAPI cluster",
			nil,
			"",
			"",
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			// GIVEN CAPI cluster with a phase of tt.capiPhase
			//   and a VMC with a clusterRef to that CAPI cluster and an unset status state
			// WHEN updateState is called
			// THEN expect the VMC's status state to be tt.expectedVMCState
			vmc.Status.State = ""
			objects := []client.Object{vmc}
			if tt.capiCluster != nil {
				unstructured.SetNestedField(tt.capiCluster.Object, tt.capiPhase, "status", "phase")
				objects = append(objects, tt.capiCluster)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: fakeClient,
				log:    vzlog.DefaultLogger(),
			}

			err := r.updateState(vmc)

			a.Equal(tt.err, err)
			a.Equal(tt.expectedVMCState, string(vmc.Status.State))
		})
	}
}

// newVMCWithClusterRef returns a VMC struct pointer, with the status.clusterRef field set to point to a CAPI Cluster
func newVMCWithClusterRef(name, namespace, clusterName, clusterNamespace string) *v1alpha1.VerrazzanoManagedCluster {
	vmc := &v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: v1alpha1.VerrazzanoManagedClusterStatus{
			ClusterRef: &v1alpha1.ClusterReference{
				Name:       clusterName,
				Namespace:  clusterNamespace,
				APIVersion: capi.GVKCAPICluster.GroupVersion().String(),
				Kind:       capi.GVKCAPICluster.Kind,
			},
		},
	}
	return vmc
}

func newCAPICluster(name, namespace string) *unstructured.Unstructured {
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(capi.GVKCAPICluster)
	cluster.SetName(name)
	cluster.SetNamespace(namespace)
	return cluster
}
