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
	vzv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testCAPIClusterName                = "test-capi-cluster"
	testClusterClassName               = "test-cluster-class"
	testCAPINamespace                  = "c-12345"
	testVMCName                        = "test-vmc"
	testVMCNamespace                   = "verrazzano-mc"
	fakeControlPlaneProviderAPIVersion = "controlPlaneAPIversion"
	fakeControlPlaneProviderKind       = "controlPlaneKind"
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

// TestUpdateStatusImported tests that updateStatus correctly sets the VMC's status.imported field
func TestUpdateStatusImported(t *testing.T) {
	a := assert.New(t)
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)

	defaultCAPIClientFunc := getCAPIClientFunc
	defer func() { getCAPIClientFunc = defaultCAPIClientFunc }()

	tests := []struct {
		testName         string
		vmc              *v1alpha1.VerrazzanoManagedCluster
		expectedImported bool
	}{
		{
			"imported cluster",
			newVMC(testVMCName, testVMCNamespace),
			true,
		},
		{
			"ClusterAPI cluster",
			newVMCWithClusterRef(testVMCName, testVMCNamespace, testCAPIClusterName, testCAPINamespace),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			// GIVEN a VMC with either a nil or non-nil ClusterRef
			// WHEN updateStatus is called
			// THEN expect the VMC's status imported field to be set
			getCAPIClientFunc = fakeCAPIClient
			ctx := context.TODO()
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.vmc).Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: fakeClient,
				log:    vzlog.DefaultLogger(),
			}

			err := r.updateStatus(ctx, tt.vmc)
			a.NoError(err)

			retrievedVMC := &v1alpha1.VerrazzanoManagedCluster{}
			err = r.Get(ctx, types.NamespacedName{Name: tt.vmc.Name, Namespace: tt.vmc.Namespace}, retrievedVMC)
			a.NoError(err)
			a.Equal(tt.expectedImported, *retrievedVMC.Status.Imported)
		})
	}
}

// TestUpdateProvider tests that updateProvider correctly sets the VMC's status.provider field
func TestUpdateProvider(t *testing.T) {
	a := assert.New(t)
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)

	tests := []struct {
		testName             string
		vmc                  *v1alpha1.VerrazzanoManagedCluster
		capiCluster          *v1beta1.Cluster
		clusterClass         *v1beta1.ClusterClass
		controlPlaneProvider string
		infraProvider        string
		expectedVMCProvider  string
		err                  error
	}{
		{
			"imported cluster",
			newVMC(testVMCName, testVMCNamespace),
			nil,
			nil,
			"",
			"",
			importedProviderDisplayName,
			nil,
		},
		{
			"CAPI Cluster without ClusterClass and a generic provider",
			newVMCWithClusterRef(testVMCName, testVMCNamespace, testCAPIClusterName, testCAPINamespace),
			newCAPICluster(testCAPIClusterName, testCAPINamespace),
			nil,
			"SomeControlPlaneProvider",
			"SomeInfraProvider",
			"SomeControlPlaneProvider on SomeInfraProvider Infrastructure",
			nil,
		},
		{
			"CAPI Cluster without ClusterClass and Oracle OKE Provider",
			newVMCWithClusterRef(testVMCName, testVMCNamespace, testCAPIClusterName, testCAPINamespace),
			newCAPICluster(testCAPIClusterName, testCAPINamespace),
			nil,
			capi.OKEControlPlaneProvider,
			capi.OKEInfrastructureProvider,
			okeProviderDisplayName,
			nil,
		},
		{
			"CAPI Cluster with ClusterClass and Oracle OCNE Provider",
			newVMCWithClusterRef(testVMCName, testVMCNamespace, testCAPIClusterName, testCAPINamespace),
			newCAPIClusterWithClassReference(testCAPIClusterName, testClusterClassName, testCAPINamespace),
			newCAPIClusterClass(testClusterClassName, testCAPINamespace),
			capi.OCNEControlPlaneProvider,
			capi.OCNEInfrastructureProvider,
			ocneProviderDisplayName,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			objects := []client.Object{tt.vmc}
			if tt.capiCluster != nil {
				if tt.clusterClass == nil {
					tt.capiCluster.Spec.InfrastructureRef.Kind = tt.infraProvider
					tt.capiCluster.Spec.ControlPlaneRef.Kind = tt.controlPlaneProvider
				}
				objects = append(objects, tt.capiCluster)
			}
			if tt.clusterClass != nil {
				tt.clusterClass.Spec.Infrastructure.Ref.Kind = tt.infraProvider
				tt.clusterClass.Spec.ControlPlane.Ref.Kind = tt.controlPlaneProvider
				objects = append(objects, tt.clusterClass)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: fakeClient,
				log:    vzlog.DefaultLogger(),
			}

			// WHEN updateProvider is called
			// THEN expect the VMC's status provider to be tt.expectedVMCProvider
			err := r.updateProvider(tt.vmc)

			a.Equal(tt.err, err)
			a.Equal(tt.expectedVMCProvider, tt.vmc.Status.Provider)
		})
	}
}

// TestUpdateStateCAPI tests that updateState correctly updates the status.state of the VMC when
// the VMC has a reference to a CAPI cluster
func TestUpdateStateCAPI(t *testing.T) {
	a := assert.New(t)
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
	vmc := newVMCWithClusterRef(testVMCName, testVMCNamespace, testCAPIClusterName, testCAPINamespace)

	tests := []struct {
		testName         string
		capiCluster      *v1beta1.Cluster
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
			// GIVEN a CAPI cluster with a phase of tt.capiPhase
			//   and a VMC with a clusterRef to that CAPI cluster and an unset status state
			// WHEN updateState is called
			// THEN expect the VMC's status state to be tt.expectedVMCState
			vmc.Status.State = ""
			objects := []client.Object{vmc}
			if tt.capiCluster != nil {
				tt.capiCluster.Status.Phase = tt.capiPhase
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

// TestShouldUpdateK8sVersion tests that shouldUpdateK8sVersion correctly determines when the VMC controller should
// update the VMC's Kubernetes version.
func TestShouldUpdateK8sVersion(t *testing.T) {
	a := assert.New(t)
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
	_ = vzv1beta1.AddToScheme(scheme)

	defaultCAPIClientFunc := getCAPIClientFunc
	defer func() { getCAPIClientFunc = defaultCAPIClientFunc }()

	tests := []struct {
		testName      string
		setClusterRef bool
		vzOnCAPI      bool
		shouldUpdate  bool
		err           error
	}{
		{
			"non-ClusterAPI cluster",
			false,
			false,
			false,
			nil,
		},
		{
			"ClusterAPI cluster without Verrazzano",
			true,
			false,
			true,
			nil,
		},
		{
			"ClusterAPI cluster with Verrazzano",
			true,
			true,
			false,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			var vmc *v1alpha1.VerrazzanoManagedCluster
			if tt.setClusterRef {
				if tt.vzOnCAPI {
					getCAPIClientFunc = fakeCAPIClientWithVZ
				} else {
					getCAPIClientFunc = fakeCAPIClient
				}
				vmc = newVMCWithClusterRef(testVMCName, testVMCNamespace, testCAPIClusterName, testCAPINamespace)
			} else {
				vmc = newVMC(testVMCName, testVMCNamespace)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(vmc).Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: fakeClient,
				Scheme: scheme,
				log:    vzlog.DefaultLogger(),
			}

			shouldUpdate, err := r.shouldUpdateK8sVersion(vmc)
			a.Equal(tt.err, err)
			a.Equal(tt.shouldUpdate, shouldUpdate)
		})
	}
}

// TestUpdateK8sVersionUsingCAPI tests that updateK8sVersionUsingCAPI correctly updates the Kubernetes version
// on the VMC
func TestUpdateK8sVersionUsingCAPI(t *testing.T) {
	const cpProviderName = "SomeControlPlaneProvider"
	const expectedK8sVersion = "v9.99.9"

	a := assert.New(t)
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)

	// Create a VMC that references a CAPI cluster
	vmc := newVMCWithClusterRef(testVMCName, testVMCNamespace, testCAPIClusterName, testCAPINamespace)
	// The CAPI cluster references some control plane provider
	cluster := newCAPICluster(testCAPIClusterName, testCAPINamespace)
	cluster.Spec.ControlPlaneRef = &corev1.ObjectReference{
		Name:       cpProviderName,
		Namespace:  testCAPINamespace,
		APIVersion: fakeControlPlaneProviderAPIVersion,
		Kind:       fakeControlPlaneProviderKind,
	}
	// Create the control plane provider CR on the fake client
	cpProvider := newFakeControlPlaneProvider(testCAPINamespace, cpProviderName, expectedK8sVersion)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(vmc, cluster, cpProvider).Build()
	r := &VerrazzanoManagedClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		log:    vzlog.DefaultLogger(),
	}

	err := r.updateK8sVersionUsingCAPI(vmc)
	a.Nil(err)
	a.Equal(expectedK8sVersion, vmc.Status.Kubernetes.Version)
}

// fakeCAPIClient returns a fake client for a CAPI workload cluster
func fakeCAPIClient(ctx context.Context, cli client.Client, cluster types.NamespacedName, scheme *runtime.Scheme) (client.Client, error) {
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build(), nil
}

// fakeCAPIClient returns a fake client for a CAPI workload cluster
func fakeCAPIClientWithVZ(ctx context.Context, cli client.Client, cluster types.NamespacedName, scheme *runtime.Scheme) (client.Client, error) {
	vz := vzv1beta1.Verrazzano{}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(&vz).Build(), nil
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
				APIVersion: "cluster.x-k8s.io/v1beta1",
				Kind:       "Cluster",
			},
		},
	}
	return vmc
}

// newVMC returns a VMC struct pointer
func newVMC(name, namespace string) *v1alpha1.VerrazzanoManagedCluster {
	vmc := &v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return vmc
}

// newCAPICluster returns a CAPI Cluster
func newCAPICluster(name, namespace string) *v1beta1.Cluster {
	cluster := v1beta1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{},
			ControlPlaneRef:   &corev1.ObjectReference{},
		},
	}
	return &cluster
}

// newCAPIClusterWithClassReference returns a CAPI Cluster which references a ClusterClass
func newCAPIClusterWithClassReference(name, className, namespace string) *v1beta1.Cluster {
	cluster := v1beta1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.ClusterSpec{
			Topology: &v1beta1.Topology{
				Class: className,
			},
		},
	}
	return &cluster
}

// newCAPIClusterClass returns a CAPI ClusterClass
func newCAPIClusterClass(name, namespace string) *v1beta1.ClusterClass {
	clusterClass := v1beta1.ClusterClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterClass",
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.ClusterClassSpec{
			Infrastructure: v1beta1.LocalObjectTemplate{
				Ref: &corev1.ObjectReference{},
			},
			ControlPlane: v1beta1.ControlPlaneClass{
				LocalObjectTemplate: v1beta1.LocalObjectTemplate{
					Ref: &corev1.ObjectReference{},
				},
			},
		},
	}
	return &clusterClass
}

// newFakeControlPlaneProvider returns a pointer to an unstructured object, representing a control
// plane provider CR
func newFakeControlPlaneProvider(namespace, name, k8sVersion string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fakeControlPlaneProviderAPIVersion,
			"kind":       fakeControlPlaneProviderKind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
			"status": map[string]interface{}{
				"version": k8sVersion,
			},
		},
	}
}
