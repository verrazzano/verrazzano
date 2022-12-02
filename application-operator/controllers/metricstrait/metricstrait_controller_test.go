// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"testing"
	"time"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	foo         = "foo"
	bar         = "bar"
	getError    = "get error"
	updateError = "update error"
)

type erroringGetClient struct {
	client.Client
}

type erroringUpdateClient struct {
	client.Client
}

func (eg *erroringGetClient) Get(_ context.Context, _ client.ObjectKey, _ client.Object) error {
	return errors.NewTooManyRequests(getError, 0)
}

func (eu *erroringUpdateClient) Update(_ context.Context, _ client.Object, _ ...client.UpdateOption) error {
	return errors.NewTooManyRequests(updateError, 0)
}

// TestReconcilerSetupWithManager test the creation of the metrics trait reconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var mgr *mocks.MockManager
	var cli *mocks.MockClient
	var scheme *runtime.Scheme
	var reconciler Reconciler
	var err error

	mocker = gomock.NewController(t)
	mgr = mocks.NewMockManager(mocker)
	cli = mocks.NewMockClient(mocker)
	scheme = runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	reconciler = Reconciler{Client: cli, Scheme: scheme, Scraper: "istio-system/prometheus"}
	mgr.EXPECT().GetControllerOptions().AnyTimes()
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(logr.Discard())
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}

// TestMetricsTraitCreatedForContainerizedWorkload tests the creation of a metrics trait related to a containerized workload.
// GIVEN a metrics trait that has been created
// AND the metrics trait is related to a containerized workload
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedForContainerizedWorkload(t *testing.T) {
	assert := asserts.New(t)

	c := containerizedWorkloadClient(false, false, false)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
	trait := vzapi.MetricsTrait{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: "test-trait-name", Namespace: "test-namespace"}, &trait)
	assert.NoError(err)
	assert.Equal("test-namespace", trait.Namespace)
	assert.Equal("test-trait-name", trait.Name)
	assert.Len(trait.Finalizers, 1)
	assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
}

// TestMetricsTraitCreatedForVerrazzanoWorkload tests the creation of a metrics trait related to a Verrazzano workload.
// The Verrazzano workload contains the real workload so we need to unwrap it.
// GIVEN a metrics trait that has been created
// AND the metrics trait is related to a Verrazzano workload
// WHEN the metrics trait Reconcile method is invoked
// THEN the contained workload should be unwrapped
// AND verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedForVerrazzanoWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)

	c := cohWorkloadClient(false, -1)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
	trait := vzapi.MetricsTrait{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: "test-trait-name", Namespace: "test-namespace"}, &trait)
	assert.NoError(err)
	assert.Equal("test-namespace", trait.Namespace)
	assert.Equal("test-trait-name", trait.Name)
	assert.Len(trait.Finalizers, 1)
	assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
}

// TestMetricsTraitCreatedForDeploymentWorkload tests the creation of a metrics trait related to a native Kubernetes Deployment workload.
// GIVEN a metrics trait that has been created
// AND the metrics trait is related to a k8s deployment workload
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedForDeploymentWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)

	c := deploymentWorkloadClient(false)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
	trait := vzapi.MetricsTrait{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: "test-trait-name", Namespace: "test-namespace"}, &trait)
	assert.NoError(err)
	assert.Equal("test-namespace", trait.Namespace)
	assert.Equal("test-trait-name", trait.Name)
	assert.Len(trait.Finalizers, 1)
	assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
}

// TestMetricsTraitDeletedForContainerizedWorkload tests deletion of a metrics trait related to a containerized workload.
// GIVEN a metrics trait with a non-zero deletion time
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is removed
// AND verify that pod annotations are cleaned up
// AND verify that the scraper configmap is cleanup up
// AND verify that the scraper pod is restarted
func TestMetricsTraitDeletedForContainerizedWorkload(t *testing.T) {
	assert := asserts.New(t)

	c := containerizedWorkloadClient(true, false, false)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
	trait := vzapi.MetricsTrait{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: "test-trait-name", Namespace: "test-namespace"}, &trait)
	assert.NoError(err)
	assert.Len(trait.Finalizers, 0)
}

// TestMetricsTraitDeletedForContainerizedWorkload tests deletion of a metrics trait related to a containerized workload.
// GIVEN a metrics trait with a non-zero deletion time
// GIVEN the related deployment resource no longer exists
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is removed
// AND verify that the scraper configmap is cleanup up
// AND verify that the scraper pod is restarted
// AND verify that the finalizer is removed
func TestMetricsTraitDeletedForContainerizedWorkloadWhenDeploymentDeleted(t *testing.T) {
	assert := asserts.New(t)

	c := containerizedWorkloadClient(true, true, false)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
	trait := vzapi.MetricsTrait{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: "test-trait-name", Namespace: "test-namespace"}, &trait)
	assert.NoError(err)
	assert.Len(trait.Finalizers, 0)
}

// TestMetricsTraitDeletedForContainerizedWorkload tests deletion of a metrics trait related to a containerized workload.
// GIVEN a metrics trait with a non-zero deletion time
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is removed
// AND verify that pod annotations are cleaned up
// AND verify that the scraper configmap is cleanup up
// AND verify that the scraper pod is restarted
func TestMetricsTraitDeletedForDeploymentWorkload(t *testing.T) {
	assert := asserts.New(t)

	c := deploymentWorkloadClient(true)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
	trait := vzapi.MetricsTrait{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: "test-trait-name", Namespace: "test-namespace"}, &trait)
	assert.NoError(err)
	assert.Len(trait.Finalizers, 0)
}

// TestFetchTraitError tests a failure to fetch the trait during reconcile.
// GIVEN a valid new metrics trait
// WHEN the metrics trait Reconcile method is invoked
// AND a failure occurs fetching the metrics trait
// THEN verify the metrics trait finalizer is added
// AND verify the error is propigated to the caller
func TestFetchTraitError(t *testing.T) {
	assert := asserts.New(t)

	scheme := k8scheme.Scheme
	_ = vzapi.AddToScheme(scheme)

	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create and make the request
	reconciler := newMetricsTraitReconciler(c)
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.Nil(err)
	assert.Equal(false, result.Requeue)
}

// TestWorkloadFetchError tests failing to fetch the workload during reconcile.
// GIVEN a valid new metrics trait
// WHEN the the metrics trait Reconcile method is invoked
// AND a failure occurs fetching the metrics trait
// THEN verify the metrics trait finalizer is added
// AND verify the error is propigated to the caller
func TestWorkloadFetchError(t *testing.T) {
	assert := asserts.New(t)

	scheme := k8scheme.Scheme
	_ = vzapi.AddToScheme(scheme)

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&vzapi.MetricsTrait{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind,
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: "test-namespace",
				Name:      "test-trait-name",
				Labels:    map[string]string{appObjectMetaLabel: "test-app", compObjectMetaLabel: "test-comp"},
			},
			Spec: vzapi.MetricsTraitSpec{
				WorkloadReference: oamrt.TypedReference{
					APIVersion: oamcore.SchemeGroupVersion.Identifier(),
					Kind:       oamcore.ContainerizedWorkloadKind,
					Name:       "test-workload-name",
				},
			},
		},
	).Build()

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NotNil(err)
	assert.Equal(true, result.Requeue)
}

// TestDeploymentUpdateError testing failing to update a workload child deployment during reconcile.
// GIVEN a metrics trait that has been updated
// WHEN the metrics trait Reconcile method is invoked
// AND an error occurs updating the scraper deployment
// THEN verify an error is recorded in the status
func TestDeploymentUpdateError(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	testDeployment := k8sapps.Deployment{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
			Kind:       "Deployment",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:              "test-deployment-name",
			Namespace:         "test-namespace",
			CreationTimestamp: k8smeta.Now(),
			OwnerReferences: []k8smeta.OwnerReference{{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name",
				UID:        "test-workload-uid"}}}}
	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels: map[string]string{
					appObjectMetaLabel:  "test-app",
					compObjectMetaLabel: "test-comp",
				}}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name"}
			return nil
		}).Times(2)
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the containerized workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		}).Times(2)
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			deployment.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			deployment.Kind = deploymentKind
			deployment.Namespace = name.Namespace
			deployment.Name = name.Name
			return nil
		})
	// Expect a call to get the containerized workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.True(list.GetKind() == deploymentKind || list.GetKind() == serviceKind)
			if list.GetKind() == deploymentKind {
				return appendAsUnstructured(list, testDeployment)
			}
			return nil
		}).Times(2)
	// Expect a call to get the deployment definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-deployment-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			deployment.ObjectMeta = testDeployment.ObjectMeta
			deployment.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to update the child with annotations but return an error.
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("test-error"))
	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the ingress trait.
	// The status should include the error updating the deployment
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			assert.Equal(oamrt.ReasonReconcileError, trait.Status.Conditions[0].Reason)
			return nil
		})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUnsupportedWorkloadType tests a metrics trait with an unsupported workload type
// GIVEN a metrics trait has an unsupported workload type of ConfigMap
// WHEN the metrics trait Reconcile method is invoked
// THEN verify the trait is deleted
func TestUnsupportedWorkloadType(t *testing.T) {
	assert := asserts.New(t)

	scheme := k8scheme.Scheme
	_ = vzapi.AddToScheme(scheme)

	workload := unstructured.Unstructured{}
	workload.SetAPIVersion("fakeAPIVersion")
	workload.SetKind("fakeKind")
	workload.SetName("test-workload-name")
	workload.SetNamespace("test-namespace")

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&vzapi.MetricsTrait{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind,
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: "test-namespace",
				Name:      "test-trait-name",
				Labels:    map[string]string{appObjectMetaLabel: "test-app", compObjectMetaLabel: "test-comp"},
			},
			Spec: vzapi.MetricsTraitSpec{
				WorkloadReference: oamrt.TypedReference{
					APIVersion: "fakeAPIVersion",
					Kind:       "fakeKind",
					Name:       "test-workload-name",
				},
			},
		},
		&workload,
	).Build()

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestMetricsTraitCreatedForWLSWorkload tests creation of a metrics trait related to a WLS workload.
// GIVEN a metrics trait that has been created
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedForWLSWorkload(t *testing.T) {
	assert := asserts.New(t)

	c := wlsWorkloadClient(false)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestMetricsTraitDeletedForWLSWorkload tests reconciling a deleted metrics trait related to a WLS workload.
// GIVEN a metrics trait with a non-zero deletion time
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is removed
// AND verify that pod annotations are cleaned up
// AND verify that the scraper configmap is cleanup up
// AND verify that the scraper pod is restarted
func TestMetricsTraitDeletedForWLSWorkload(t *testing.T) {
	assert := asserts.New(t)

	c := wlsWorkloadClient(true)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
}

// TestMetricsTraitCreatedWithMultiplePorts tests the creation of a metrics trait related to a Coherence workload.
// GIVEN a metrics trait that has been created that specifies multiple metrics ports
// AND the metrics trait is related to a Coherence workload
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedWithMultiplePorts(t *testing.T) {
	assert := asserts.New(t)

	c := cohWorkloadClient(false, -1, 8080, 8081, 8082)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestMetricsTraitCreatedWithMultiplePortsAndPort tests the creation of a metrics trait related to a Coherence workload.
// GIVEN a metrics trait that has been created that specifies multiple metrics ports and a single port
// AND the metrics trait is related to a Coherence workload
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedWithMultiplePortsAndPort(t *testing.T) {
	assert := asserts.New(t)

	c := cohWorkloadClient(false, 8080, 8081, 8082, 8083)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestMetricsTraitDeletedForCOHWorkload tests deletion of a metrics trait related to a coherence workload.
// GIVEN a metrics trait with a non-zero deletion time
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is removed
// AND verify that pod annotations are cleaned up
// AND verify that the scraper configmap is cleanup up
// AND verify that the scraper pod is restarted
func TestMetricsTraitDeletedForCOHWorkload(t *testing.T) {
	assert := asserts.New(t)

	c := cohWorkloadClient(true, -1)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
}

// TestUseHTTPSForScrapeTargetFalseConditions tests that false is returned for the following conditions
// GIVEN a unlabeled Istio namespace or a workload of kind VerrazzanoCoherenceWorkload or a workload of kind Coherence
// WHEN the useHttpsForScrapeTarget method is invoked
// THEN verify that the false boolean value is returned since all those conditions require an http scrape target
func TestUseHTTPSForScrapeTargetFalseConditions(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mtrait := vzapi.MetricsTrait{
		TypeMeta: k8smeta.TypeMeta{
			Kind: "VerrazzanoCoherenceWorkload",
		},
	}

	testNamespace := k8score.Namespace{
		TypeMeta: k8smeta.TypeMeta{
			Kind: "Namespace",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name: "test-namespace",
		},
	}

	// Expect a call to get the namespace definition
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, namespace *k8score.Namespace) error {
			namespace.TypeMeta = testNamespace.TypeMeta
			namespace.ObjectMeta = testNamespace.ObjectMeta
			return nil
		})

	mtrait.Spec.WorkloadReference.Kind = "VerrazzanoCoherenceWorkload"
	https, _ := useHTTPSForScrapeTarget(context.TODO(), nil, &mtrait)
	// Expect https to be false for scrape target of Kind VerrazzanoCoherenceWorkload
	assert.False(https, "Expected https to be false for Workload of Kind VerrazzanoCoherenceWorkload")

	mtrait.Spec.WorkloadReference.Kind = "Coherence"
	https, _ = useHTTPSForScrapeTarget(context.TODO(), nil, &mtrait)
	// Expect https to be false for scrape target of Kind Coherence
	assert.False(https, "Expected https to be false for Workload of Kind Coherence")

	reconciler := newMetricsTraitReconciler(mock)

	mtrait.Spec.WorkloadReference.Kind = ""
	https, _ = useHTTPSForScrapeTarget(context.TODO(), reconciler.Client, &mtrait)
	// Expect https to be false for namespaces NOT labeled for istio-injection
	assert.False(https, "Expected https to be false for namespace NOT labeled for istio injection")
	mocker.Finish()
}

// TestUseHTTPSForScrapeTargetTrueCondition tests that true is returned for namespaces marked for Istio injection
// GIVEN a labeled Istio namespace
// WHEN the useHttpsForScrapeTarget method is invoked
// THEN verify that the true boolean value is returned since pods in Istio namespaces require an https scrape target because of MTLS
func TestUseHTTPSForScrapeTargetTrueCondition(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mtrait := vzapi.MetricsTrait{
		TypeMeta: k8smeta.TypeMeta{
			Kind: "ContainerizedWorkload",
		},
	}

	testNamespace := k8score.Namespace{
		TypeMeta: k8smeta.TypeMeta{
			Kind: "Namespace",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name: "test-namespace",
		},
	}

	// Expect a call to get the namespace definition
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, namespace *k8score.Namespace) error {
			namespace.TypeMeta = testNamespace.TypeMeta
			namespace.ObjectMeta = testNamespace.ObjectMeta
			return nil
		})

	reconciler := newMetricsTraitReconciler(mock)

	labels := map[string]string{
		"istio-injection": "enabled",
	}
	testNamespace.ObjectMeta.Labels = labels
	https, _ := useHTTPSForScrapeTarget(context.TODO(), reconciler.Client, &mtrait)
	// Expect https to be true for namespaces labeled for istio-injection
	assert.True(https, "Expected https to be true for namespaces labeled for Istio injection")
	mocker.Finish()
}

// newMetricsTraitReconciler creates a new reconciler for testing
// cli - The Kerberos client to inject into the reconciler
func newMetricsTraitReconciler(cli client.Client) Reconciler {
	scheme := runtime.NewScheme()
	vzapi.AddToScheme(scheme)
	reconciler := Reconciler{
		Client:  cli,
		Log:     zap.S(),
		Scheme:  scheme,
		Scraper: "istio-system/prometheus",
	}
	return reconciler
}

// convertToUnstructured converts an object to an Unstructured version
// object - The object to convert to Unstructured
func convertToUnstructured(object interface{}) (unstructured.Unstructured, error) {
	bytes, err := json.Marshal(object)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	var u map[string]interface{}
	json.Unmarshal(bytes, &u)
	return unstructured.Unstructured{Object: u}, nil
}

// appendAsUnstructured appends an object to the list after converting it to an Unstructured
// list - The list to append to.
// object - The object to convert to Unstructured and append to the list
func appendAsUnstructured(list *unstructured.UnstructuredList, object interface{}) error {
	u, err := convertToUnstructured(object)
	if err != nil {
		return err
	}
	list.Items = append(list.Items, u)
	return nil
}

// TestReconcileKubeSystem tests to make sure we do not reconcile
// Any resource that belong to the kube-system namespace
func TestReconcileKubeSystem(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// create a request and reconcile it
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: vzconst.KubeSystem, Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	assert.Nil(err)
	assert.True(result.IsZero())
}

// TestMetricsTraitDisabledForContainerizedWorkload tests the creation of a metrics trait related to a containerized workload.
// GIVEN a metrics trait that has been disabled
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that the scraper configmap is updated with the scrape job being removed
func TestMetricsTraitDisabledForContainerizedWorkload(t *testing.T) {
	assert := asserts.New(t)

	c := containerizedWorkloadClient(false, false, true)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)
	// Validate the results
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
}

// TestLegacyPrometheusScraper tests the case where the scraper is the default (legacy) VMO Prometheus.
func TestLegacyPrometheusScraper(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN a containerized workload and the reconciler scraper is configured to use the default (legacy) Prometheus scraper
	//  WHEN we reconcile metrics traits
	//  THEN the trait is updated with a finalizer and a ServiceMonitor has been created
	c := containerizedWorkloadClient(false, false, false)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(c)
	reconciler.Scraper = constants.DefaultScraperName

	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.True(result.RequeueAfter > 0)

	trait := vzapi.MetricsTrait{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: "test-trait-name", Namespace: "test-namespace"}, &trait)
	assert.NoError(err)
	assert.Equal("test-namespace", trait.Namespace)
	assert.Equal("test-trait-name", trait.Name)
	assert.Len(trait.Finalizers, 1)
	assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])

	monitor := &promoperapi.ServiceMonitor{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "test-namespace", Name: "test-app-test-namespace-test-comp"}, monitor)
	assert.NoError(err)
}

// TestCreateScrapeConfigFromTrait tests the createScrapeConfigFromTrait func calls
func TestCreateScrapeConfigFromTrait(t *testing.T) {
	scheme := k8scheme.Scheme
	_ = promoperapi.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = oamcore.SchemeBuilder.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&k8score.Namespace{}).Build()
	testWorkLoad := unstructured.Unstructured{}
	testWorkLoad.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)

	testWLSWorkload := unstructured.Unstructured{}
	testWLSWorkload.SetAPIVersion("weblogic.oracle")
	testWLSWorkload.SetKind("Domain")
	tests := []struct {
		name        string
		trait       vzapi.MetricsTrait
		workload    *unstructured.Unstructured
		wantErr     bool
		errContains string
		isConfigNil bool
		secret      *k8score.Secret
	}{
		// GIVEN a metricstrait and workload with default values
		// WHEN createScrapeConfigFromTrait method is called
		// THEN a non nil scrape config is created and no error is returned
		{
			"Create scrape config from a trait correctly",
			vzapi.MetricsTrait{
				ObjectMeta: k8smeta.ObjectMeta{
					Labels: map[string]string{
						appObjectMetaLabel:  foo,
						compObjectMetaLabel: bar,
					},
				},
			},
			&testWorkLoad,
			false,
			"",
			false,
			nil,
		},
		// GIVEN a metricstrait missing app name label
		// WHEN createScrapeConfigFromTrait is called
		// THEN an error is returned
		{
			"Trait missing app name label",
			vzapi.MetricsTrait{
				ObjectMeta: k8smeta.ObjectMeta{
					Labels: map[string]string{
						compObjectMetaLabel: bar,
					},
				},
			},
			nil,
			true,
			"metrics trait missing application name label",
			true,
			nil,
		},
		// GIVEN a metricstrait with missing component label
		// WHEN createScrapeConfigFromTrait is called
		// THEN an error is returned
		{
			"Trait missing component label",
			vzapi.MetricsTrait{
				ObjectMeta: k8smeta.ObjectMeta{
					Labels: map[string]string{
						appObjectMetaLabel: foo,
					},
				},
			},
			&testWorkLoad,
			true,
			"metrics trait missing component name label",
			true,
			nil,
		},
		// GIVEN a nil workload and correct metricstrait
		// WHEN createScrapeConfigFromTrait is called
		// THEN no error is returned along with a nil config
		{
			"Nil workload",
			vzapi.MetricsTrait{
				ObjectMeta: k8smeta.ObjectMeta{
					Labels: map[string]string{
						appObjectMetaLabel:  foo,
						compObjectMetaLabel: bar,
					},
				},
			},
			nil,
			false,
			"",
			true,
			nil,
		},
		// GIVEN a wls workload and correct metricstrait
		// WHEN createScrapeConfigFromTrait is called
		// THEN no error is returned along with a non nil config
		{
			"WLS Workload",
			vzapi.MetricsTrait{
				ObjectMeta: k8smeta.ObjectMeta{
					Labels: map[string]string{
						appObjectMetaLabel:  foo,
						compObjectMetaLabel: bar,
					},
				},
			},
			&testWLSWorkload,
			false,
			"",
			false,
			nil,
		},
		{
			"Basic auth",
			vzapi.MetricsTrait{
				ObjectMeta: k8smeta.ObjectMeta{
					Labels: map[string]string{
						appObjectMetaLabel:  foo,
						compObjectMetaLabel: bar,
					},
				},
			},
			&testWLSWorkload,
			false,
			"",
			false,
			&k8score.Secret{
				Data: map[string][]byte{
					"username": []byte(foo),
					"password": []byte(bar),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, job, err := createScrapeConfigFromTrait(context.Background(), &tt.trait, 0, tt.secret, tt.workload, c)
			if tt.wantErr {
				asserts.ErrorContains(t, err, tt.errContains)
			} else {
				asserts.NoError(t, err)
				if tt.isConfigNil {
					asserts.Nil(t, job)
				} else {
					asserts.NotNil(t, job)
					asserts.True(t, job.Exists("scheme"))
					asserts.Equal(t, tt.secret != nil, job.Exists(basicAuthLabel))
				}
			}
		})
	}
}

// TestRemovedTraitReferencesFromOwner tests the removedTraitReferencesFromOwner reconcile method
func TestRemovedTraitReferencesFromOwner(t *testing.T) {
	testTrait := vzapi.MetricsTrait{
		ObjectMeta: k8smeta.ObjectMeta{
			Labels: map[string]string{
				appObjectMetaLabel:  foo,
				compObjectMetaLabel: bar,
			},
		},
	}
	testTrait.APIVersion = foo
	testTrait.Kind = bar
	ttro := runtime.Object(&testTrait)
	var scope conversion.Scope
	testTraitRawExtension := &runtime.RawExtension{}
	asserts.NoError(t, runtime.Convert_runtime_Object_To_runtime_RawExtension(&ttro, testTraitRawExtension, scope))

	scheme := k8scheme.Scheme
	_ = vzapi.AddToScheme(scheme)
	_ = oamcore.SchemeBuilder.AddToScheme(scheme)
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&k8score.Namespace{},
		&oamcore.ApplicationConfiguration{
			ObjectMeta: k8smeta.ObjectMeta{
				Name: foo,
			},
			Spec: oamcore.ApplicationConfigurationSpec{
				Components: []oamcore.ApplicationConfigurationComponent{
					{
						ComponentName: bar,
						Traits: []oamcore.ComponentTrait{
							{
								Trait: *testTraitRawExtension,
							},
						},
					},
				},
			},
		},
	).Build()

	badGetClient := &erroringGetClient{cli}
	badUpdateClient := &erroringUpdateClient{cli}

	tests := []struct {
		name            string
		client          client.Client
		wantErr         bool
		errContains     string
		operationResult controllerutil.OperationResult
	}{
		// GIVEN a scenario where client is unable to make Get call to the api server
		// WHEN a call is made to removedTraitReferencesFromOwner func by the reconciler
		// then the method returns an error
		{
			"Unsuccessful get",
			badGetClient,
			true,
			getError,
			controllerutil.OperationResultNone,
		},
		// GIVEN a scenario where client is unable to make Update call to the api server
		// WHEN a call is made to removedTraitReferencesFromOwner func by the reconciler
		// then the method returns an error
		{
			"Unsuccessful Update",
			badUpdateClient,
			true,
			updateError,
			controllerutil.OperationResultNone,
		},
		// GIVEN an appconfig with the given metricstrait as a component
		// WHEN a call is made to removedTraitReferencesFromOwner func by the reconciler
		// THEN the appconfig is updated to remove the trait from the appconfig
		{
			"Successful update",
			cli,
			false,
			"",
			controllerutil.OperationResultUpdated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := newMetricsTraitReconciler(tt.client)
			_, op, err := reconciler.removedTraitReferencesFromOwner(context.Background(), &k8smeta.OwnerReference{Name: foo}, &testTrait, vzlog.DefaultLogger())
			asserts.Equal(t, op, tt.operationResult)
			if tt.wantErr {
				asserts.ErrorContains(t, err, tt.errContains)
			} else {
				asserts.NoError(t, err)
			}
		})
	}
}

// TestDeleteServiceMonitor tests the deleteServiceMonitor func call
func TestDeleteServiceMonitor(t *testing.T) {
	scheme := k8scheme.Scheme
	_ = vzapi.AddToScheme(scheme)
	_ = promoperapi.AddToScheme(scheme)
	_ = oamcore.SchemeBuilder.AddToScheme(scheme)
	tests := []struct {
		name   string
		trait  vzapi.MetricsTrait
		result controllerutil.OperationResult
	}{
		// GIVEN a metricstrait
		// WHEN the trait is enabled and not scheduled for deletion
		// THEN the ServiceMonitor resource is not deleted
		{
			"Trait is enabled",
			vzapi.MetricsTrait{
				Spec: vzapi.MetricsTraitSpec{
					Enabled: getBoolPtr(true),
				},
			},
			controllerutil.OperationResultNone,
		},
		// GIVEN a metricstrait
		// WHEN the trait is scheduled for deletion
		// THEN the associated ServiceMonitor is deleted
		{
			"Successful Delete",
			vzapi.MetricsTrait{
				ObjectMeta: k8smeta.ObjectMeta{
					DeletionTimestamp: &k8smeta.Time{Time: time.Now()},
				},
			},
			controllerutil.OperationResultUpdated,
		},
		// GIVEN a metricstrait
		// WHEN the trait is disabled for deletion
		// THEN the associated ServiceMonitor is deleted
		{
			"Trait is disabled",
			vzapi.MetricsTrait{
				Spec: vzapi.MetricsTraitSpec{
					Enabled: getBoolPtr(false),
				},
			},
			controllerutil.OperationResultUpdated,
		},
		{
			"Enabled but scheduled for deletion",
			vzapi.MetricsTrait{
				ObjectMeta: k8smeta.ObjectMeta{
					DeletionTimestamp: &k8smeta.Time{Time: time.Now()},
				},
				Spec: vzapi.MetricsTraitSpec{
					Enabled: getBoolPtr(true),
				},
			},
			controllerutil.OperationResultUpdated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&promoperapi.ServiceMonitor{
					ObjectMeta: k8smeta.ObjectMeta{
						Name:      foo,
						Namespace: bar,
					},
				},
				&k8score.Namespace{ObjectMeta: k8smeta.ObjectMeta{Name: bar}},
			).Build()
			reconciler := newMetricsTraitReconciler(cli)
			res, _ := reconciler.deleteServiceMonitor(context.TODO(), bar, foo, &tt.trait, vzlog.DefaultLogger())
			asserts.Equal(t, tt.result, res)
		})
	}
}

// TestUpdateRelatedStatefulSet tests the updateRelatedStatefulSet func call
// GIVEN metricstrait, workload and child resources
// WHEN updateRelatedPod func call is made by the reconciler
// THEN the related workload StatefulSets are mutated
func TestUpdateRelatedStatefulSet(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = k8score.AddToScheme(scheme)
	_ = k8sapps.AddToScheme(scheme)
	port := 443
	path := foo
	trait := vzapi.MetricsTrait{
		Spec: vzapi.MetricsTraitSpec{
			Port: &port,
			Path: &path,
		},
	}
	child := unstructured.Unstructured{}
	child.SetAPIVersion(foo)
	child.SetKind(bar)
	child.SetName(foo)
	child.SetKind(bar)
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&k8sapps.StatefulSet{
			TypeMeta:   k8smeta.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
			ObjectMeta: k8smeta.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName(), CreationTimestamp: k8smeta.Now()},
		},
		&k8score.Namespace{ObjectMeta: k8smeta.ObjectMeta{Name: child.GetNamespace()}},
	).Build()
	reconciler0 := newMetricsTraitReconciler(cli)
	_, res0, err0 := reconciler0.updateRelatedStatefulSet(context.Background(), &trait, nil, nil, &child, vzlog.DefaultLogger())
	asserts.Equal(t, res0, controllerutil.OperationResultUpdated)
	asserts.NoError(t, err0)

	badGetClient := &erroringGetClient{cli}
	reconciler1 := newMetricsTraitReconciler(badGetClient)
	_, res1, err1 := reconciler1.updateRelatedStatefulSet(context.Background(), &trait, nil, nil, &child, vzlog.DefaultLogger())
	asserts.Equal(t, res1, controllerutil.OperationResultNone)
	asserts.ErrorContains(t, err1, getError)
}

// TestUpdateRelatedPod tests the updateRelatedPod func call
// GIVEN metricstrait, workload and child resources
// WHEN updateRelatedPod func call is made by the reconciler
// THEN the related workload pods are mutated
func TestUpdateRelatedPod(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = k8score.AddToScheme(scheme)

	port := 443
	path := foo
	trait := vzapi.MetricsTrait{
		Spec: vzapi.MetricsTraitSpec{
			Port: &port,
			Path: &path,
		},
	}

	child := unstructured.Unstructured{}
	child.SetAPIVersion(foo)
	child.SetKind(bar)
	child.SetName(foo)
	child.SetKind(bar)

	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&k8score.Pod{
			TypeMeta:   k8smeta.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
			ObjectMeta: k8smeta.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName(), CreationTimestamp: k8smeta.Now()},
		},
		&k8score.Namespace{ObjectMeta: k8smeta.ObjectMeta{Name: child.GetNamespace()}},
	).Build()

	reconciler0 := newMetricsTraitReconciler(cli)
	_, res0, err0 := reconciler0.updateRelatedPod(context.Background(), &trait, nil, nil, &child, vzlog.DefaultLogger())
	asserts.Equal(t, res0, controllerutil.OperationResultUpdated)
	asserts.NoError(t, err0)

	badGetClient := &erroringGetClient{cli}
	reconciler1 := newMetricsTraitReconciler(badGetClient)

	_, res1, err1 := reconciler1.updateRelatedPod(context.Background(), &trait, nil, nil, &child, vzlog.DefaultLogger())
	asserts.Equal(t, res1, controllerutil.OperationResultNone)
	asserts.ErrorContains(t, err1, getError)
}

// TestUpdatePrometheusScraperConfigMap tests the updatePrometheusScraperConfigMap func call
func TestUpdatePrometheusScraperConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = k8sapps.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = k8score.AddToScheme(scheme)

	port := 443
	path := foo
	testWorkLoad := unstructured.Unstructured{}
	testWorkLoad.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)

	testTrait := vzapi.MetricsTrait{
		ObjectMeta: k8smeta.ObjectMeta{
			Labels: map[string]string{
				appObjectMetaLabel:  foo,
				compObjectMetaLabel: bar,
			},
		},
		Spec: vzapi.MetricsTraitSpec{Port: &port, Path: &path},
	}

	testDeployment0 := k8sapps.Deployment{
		ObjectMeta: k8smeta.ObjectMeta{Name: foo},
		Spec: k8sapps.DeploymentSpec{
			Template: k8score.PodTemplateSpec{
				Spec: k8score.PodSpec{
					Volumes: []k8score.Volume{
						{
							Name: "config-volume",
							VolumeSource: k8score.VolumeSource{
								ConfigMap: &k8score.ConfigMapVolumeSource{LocalObjectReference: k8score.LocalObjectReference{Name: foo}},
							},
						},
					},
				},
			},
		},
	}

	// Deployment with an empty volume spec
	testDeployment1 := testDeployment0
	testDeployment1.Spec.Template.Spec.Volumes = []k8score.Volume{}

	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&k8score.ConfigMap{
			ObjectMeta: k8smeta.ObjectMeta{Name: foo},
		},
		// &k8score.Namespace{ObjectMeta: k8smeta.ObjectMeta{Name: bar}},
		&k8score.Namespace{},
	).Build()
	badGetClient := &erroringGetClient{cli}
	badUpdateClient := &erroringUpdateClient{cli}

	tests := []struct {
		name        string
		client      client.Client
		deployment  k8sapps.Deployment
		wantErr     bool
		errContains string
	}{
		// GIVEN a call to update the scraper ConfigMap
		// WHEN the resources are correctly configured
		// THEN the ConfigMap is updated with the scraper config
		{
			"Successful update",
			cli,
			testDeployment0,
			false,
			"",
		},
		// GIVEN a call to update the scraper config
		// WHEN the client is unable to make a Get call
		// THEN an error is returned
		{
			"Unsuccessful Get",
			badGetClient,
			testDeployment0,
			true,
			getError,
		},
		// GIVEN a call to update the scraper config
		// WHEN the client is unable to make an Update call
		// THEN an error is returned
		{
			"Unsuccessful Update",
			badUpdateClient,
			testDeployment0,
			true,
			updateError,
		},
		// GIVEN a call to update the scraper config
		// WHEN the deployment does not contain the ConfigMap name
		// THEN an error is returned
		{
			"Missing cm name",
			cli,
			testDeployment1,
			true,
			"failed to find Prometheus configmap name from deployment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := newMetricsTraitReconciler(tt.client)
			_, res, err := reconciler.updatePrometheusScraperConfigMap(context.Background(), &testTrait, &testWorkLoad, nil, &tt.deployment, vzlog.DefaultLogger())
			if tt.wantErr {
				asserts.ErrorContains(t, err, tt.errContains)
			} else {
				asserts.Equal(t, controllerutil.OperationResultUpdated, res)
				asserts.NoError(t, err)
				cm := k8score.ConfigMap{}
				asserts.NoError(t, tt.client.Get(context.Background(), types.NamespacedName{Name: foo}, &cm))
				_, ok := cm.Data[prometheusConfigKey]
				asserts.True(t, ok)
			}
		})
	}
}

// deploymentWorkloadClient returns a fake client with a containerized workload target in the trait
func containerizedWorkloadClient(deleting, deploymentDeleted, traitDisabled bool) client.WithWatch {
	scheme := k8scheme.Scheme
	_ = promoperapi.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = oamcore.SchemeBuilder.AddToScheme(scheme)

	testNamespace := "test-namespace"
	testWorkloadName := "test-workload-name"
	testWorkloadUID := types.UID("test-workload-uid")

	trait := vzapi.MetricsTrait{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: vzapi.SchemeGroupVersion.Identifier(),
			Kind:       vzapi.MetricsTraitKind,
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Namespace: testNamespace,
			Name:      "test-trait-name",
			Labels:    map[string]string{appObjectMetaLabel: "test-app", compObjectMetaLabel: "test-comp"},
		},
		Spec: vzapi.MetricsTraitSpec{
			WorkloadReference: oamrt.TypedReference{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       testWorkloadName,
			},
		},
	}
	if deleting {
		trait.DeletionTimestamp = &k8smeta.Time{Time: time.Now()}
	}
	trueVal := true
	if traitDisabled {
		trait.Spec.Enabled = &trueVal
	}

	objects := []client.Object{
		&k8sapps.Deployment{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Deployment",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "prometheus",
				Namespace: "istio-system",
			},
		},
		&k8score.Service{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Service",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "test-service",
				Namespace: testNamespace,
			},
		},
		&trait,
		&oamcore.ContainerizedWorkload{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: testNamespace,
				Name:      testWorkloadName,
				UID:       testWorkloadUID,
			},
		},
		&oamcore.WorkloadDefinition{
			ObjectMeta: k8smeta.ObjectMeta{
				Name: "containerizedworkloads.core.oam.dev",
			},
			Spec: oamcore.WorkloadDefinitionSpec{
				ChildResourceKinds: []oamcore.ChildResourceKind{
					{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
					{APIVersion: "v1", Kind: "Service", Selector: nil},
				},
			},
		},
		&k8score.Namespace{
			ObjectMeta: k8smeta.ObjectMeta{
				Name: "test-namespace",
			},
		},
	}

	if !deploymentDeleted {
		objects = append(objects, &k8sapps.Deployment{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Deployment",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:              "test-deployment-name",
				Namespace:         testNamespace,
				CreationTimestamp: k8smeta.Now(),
				OwnerReferences: []k8smeta.OwnerReference{{
					APIVersion: oamcore.SchemeGroupVersion.Identifier(),
					Kind:       oamcore.ContainerizedWorkloadKind,
					Name:       testWorkloadName,
					UID:        testWorkloadUID},
				},
			},
		})
	}

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

// deploymentWorkloadClient returns a fake client with a deployment target in the trait
func deploymentWorkloadClient(deleting bool) client.WithWatch {
	scheme := k8scheme.Scheme
	_ = promoperapi.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = oamcore.SchemeBuilder.AddToScheme(scheme)

	testNamespace := "test-namespace"
	testWorkloadName := "test-workload-name"

	trait := vzapi.MetricsTrait{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: vzapi.SchemeGroupVersion.Identifier(),
			Kind:       vzapi.MetricsTraitKind,
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Namespace: testNamespace,
			Name:      "test-trait-name",
			Labels:    map[string]string{appObjectMetaLabel: "test-app", compObjectMetaLabel: "test-comp"},
		},
		Spec: vzapi.MetricsTraitSpec{
			WorkloadReference: oamrt.TypedReference{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Deployment",
				Name:       testWorkloadName,
			},
		},
	}
	if deleting {
		trait.DeletionTimestamp = &k8smeta.Time{Time: time.Now()}
	}

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&k8sapps.Deployment{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Deployment",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      testWorkloadName,
				Namespace: testNamespace,
			},
		},
		&k8sapps.Deployment{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Deployment",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "prometheus",
				Namespace: "istio-system",
			},
		},
		&k8score.Service{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Service",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "test-service",
				Namespace: testNamespace,
			},
		},
		&trait,
		&oamcore.WorkloadDefinition{
			ObjectMeta: k8smeta.ObjectMeta{
				Name: "deployments.apps",
			},
			Spec: oamcore.WorkloadDefinitionSpec{
				ChildResourceKinds: []oamcore.ChildResourceKind{
					{APIVersion: "v1", Kind: "Service", Selector: nil},
				},
			},
		},
		&k8score.Namespace{
			ObjectMeta: k8smeta.ObjectMeta{
				Name: "test-namespace",
			},
		},
	).Build()
}

// wlsWorkloadClient returns a fake client with a WebLogic Workload target in the trait
func wlsWorkloadClient(deleting bool) client.WithWatch {
	scheme := k8scheme.Scheme
	_ = promoperapi.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = oamcore.SchemeBuilder.AddToScheme(scheme)

	testNamespace := "test-namespace"
	testWorkloadName := "test-workload-name"
	testWorkloadUID := types.UID("test-workload-uid")

	trait := vzapi.MetricsTrait{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: vzapi.SchemeGroupVersion.Identifier(),
			Kind:       vzapi.MetricsTraitKind,
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Namespace: testNamespace,
			Name:      "test-trait-name",
			Labels:    map[string]string{appObjectMetaLabel: "test-app", compObjectMetaLabel: "test-comp"},
		},
		Spec: vzapi.MetricsTraitSpec{
			WorkloadReference: oamrt.TypedReference{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzconst.VerrazzanoWebLogicWorkloadKind,
				Name:       testWorkloadName,
			},
		},
	}
	if deleting {
		trait.DeletionTimestamp = &k8smeta.Time{Time: time.Now()}
	}

	domain := unstructured.Unstructured{}
	domain.SetKind("Domain")
	domain.SetAPIVersion("weblogic.oracle/v8")
	domain.SetName("test-domain")
	domain.SetNamespace(testNamespace)

	objects := []client.Object{
		&k8sapps.Deployment{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Deployment",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "prometheus",
				Namespace: "istio-system",
			},
		},
		&k8score.Service{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Service",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "test-service",
				Namespace: testNamespace,
			},
		},
		&trait,
		&vzapi.VerrazzanoWebLogicWorkload{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzconst.VerrazzanoWebLogicWorkloadKind,
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: testNamespace,
				Name:      testWorkloadName,
				UID:       testWorkloadUID,
			},
			Spec: vzapi.VerrazzanoWebLogicWorkloadSpec{
				Template: vzapi.VerrazzanoWebLogicWorkloadTemplate{
					APIVersion: "weblogic.oracle/v8",
					Metadata: runtime.RawExtension{
						Raw:    []byte(`{"name": "test-domain"}`),
						Object: &unstructured.Unstructured{},
					},
				},
			},
		},
		&oamcore.WorkloadDefinition{
			ObjectMeta: k8smeta.ObjectMeta{
				Name: "domains.weblogic.oracle",
			},
			Spec: oamcore.WorkloadDefinitionSpec{
				ChildResourceKinds: []oamcore.ChildResourceKind{
					{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
					{APIVersion: "v1", Kind: "Service", Selector: nil},
				},
			},
		},
		&k8sapps.Deployment{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Deployment",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:              "test-deployment-name",
				Namespace:         testNamespace,
				CreationTimestamp: k8smeta.Now(),
				OwnerReferences: []k8smeta.OwnerReference{{
					APIVersion: oamcore.SchemeGroupVersion.Identifier(),
					Kind:       oamcore.ContainerizedWorkloadKind,
					Name:       testWorkloadName,
					UID:        testWorkloadUID},
				},
			},
		},
		&k8score.Namespace{
			ObjectMeta: k8smeta.ObjectMeta{
				Name: "test-namespace",
			},
		},
		&domain,
	}

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

// cohWorkloadClient returns a fake client with a Coherence Workload target in the trait
func cohWorkloadClient(deleting bool, portNum int, ports ...int) client.WithWatch {
	scheme := k8scheme.Scheme
	_ = promoperapi.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = oamcore.SchemeBuilder.AddToScheme(scheme)

	testNamespace := "test-namespace"
	testWorkloadName := "test-workload-name"
	testWorkloadUID := types.UID("test-workload-uid")

	coherence := unstructured.Unstructured{}
	coherence.SetNamespace(testNamespace)
	coherence.SetName("test-coherence")
	coherence.SetAPIVersion("coherence.oracle.com/v1")
	coherence.SetKind("Coherence")

	trait := vzapi.MetricsTrait{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: vzapi.SchemeGroupVersion.Identifier(),
			Kind:       vzapi.MetricsTraitKind,
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Namespace: testNamespace,
			Name:      "test-trait-name",
			Labels:    map[string]string{appObjectMetaLabel: "test-app", compObjectMetaLabel: "test-comp"},
		},
		Spec: vzapi.MetricsTraitSpec{
			WorkloadReference: oamrt.TypedReference{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzconst.VerrazzanoCoherenceWorkloadKind,
				Name:       testWorkloadName,
			},
		},
	}
	if deleting {
		trait.DeletionTimestamp = &k8smeta.Time{Time: time.Now()}
	}
	if portNum >= 0 {
		trait.Spec.Port = &portNum
	}
	path := "/metrics"
	if len(ports) > 0 {
		for i := range ports {
			port := vzapi.PortSpec{Port: &ports[i], Path: &path}
			trait.Spec.Ports = append(trait.Spec.Ports, port)
		}
	}

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&k8sapps.Deployment{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Deployment",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:              "test-deployment-name",
				Namespace:         testNamespace,
				CreationTimestamp: k8smeta.Now(),
				OwnerReferences: []k8smeta.OwnerReference{{
					APIVersion: vzapi.SchemeGroupVersion.Identifier(),
					Kind:       "Coherence",
					Name:       testWorkloadName,
					UID:        testWorkloadUID},
				},
			},
		},
		&k8sapps.Deployment{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Deployment",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "prometheus",
				Namespace: "istio-system",
			},
		},
		&k8score.Service{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Service",
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "test-service",
				Namespace: testNamespace,
			},
		},
		&trait,
		&vzapi.VerrazzanoCoherenceWorkload{
			TypeMeta: k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzconst.VerrazzanoCoherenceWorkloadKind,
			},
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: testNamespace,
				Name:      testWorkloadName,
				UID:       testWorkloadUID,
			},
			Spec: vzapi.VerrazzanoCoherenceWorkloadSpec{
				Template: runtime.RawExtension{
					Raw:    []byte(`{"metadata":{"name": "test-coherence"}}`),
					Object: &unstructured.Unstructured{},
				},
			},
		},
		&coherence,
		&oamcore.WorkloadDefinition{
			ObjectMeta: k8smeta.ObjectMeta{
				Name: "coherences.coherence.oracle.com",
			},
			Spec: oamcore.WorkloadDefinitionSpec{
				ChildResourceKinds: []oamcore.ChildResourceKind{
					{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
					{APIVersion: "v1", Kind: "Service", Selector: nil},
				},
			},
		},
	).Build()
}

func getBoolPtr(b bool) *bool {
	return &b
}
