// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingresstrait

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	istionet "istio.io/api/networking/v1alpha3"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	v1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

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
	vzapi.AddToScheme(scheme)
	reconciler = Reconciler{Client: cli, Scheme: scheme}
	mgr.EXPECT().GetConfig().Return(&rest.Config{})
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(log.NullLogger{})
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}

// TestSuccessfullyCreateNewIngress tests the Reconcile method for the following use case.
// GIVEN a request to reconcile an ingress trait resource
// WHEN the trait exists but the ingress does not
// THEN ensure that the trait is created.
func TestSuccessfullyCreateNewIngress(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	// Expect a call to get the ingress trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.IngressTrait) error {
			trait.TypeMeta = metav1.TypeMeta{
				APIVersion: "oam.verrazzano.io/v1alpha1",
				Kind:       "IngressTrait"}
			trait.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    map[string]string{oam.LabelAppName: "myapp"}}
			trait.Spec.Rules = []vzapi.IngressRule{{
				Hosts: []string{"test-host"},
				Paths: []vzapi.IngressPath{{Path: "test-path"}}}}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ContainerizedWorkload",
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to get the containerized workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetAPIVersion("core.oam.dev/v1alpha2")
			workload.SetKind("ContainerizedWorkload")
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the containerized workload resource definition
	mock.EXPECT(). // get workload definition
			Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *v1alpha2.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []v1alpha2.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Deployment", list.GetKind())
			return nil
		})
	// Expect a call to list the child Service resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			return appendAsUnstructured(list, v1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: "core.oam.dev/v1alpha2",
						Kind:       "ContainerizedWorkload",
						Name:       "test-workload-name",
						UID:        "test-workload-uid",
					}}}})
		})
	// Expect a call to get the gateway resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name-rule-0-gw"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Gateway"}, "test-trait-name-rule-0-gw"))
	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, gateway *istioclinet.Gateway, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the virtual service resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name-rule-0-vs"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "VirtualService"}, "test-trait-name-rule-0-vs"))

	// Expect a call to get the Rancher ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "cattle-system", Name: "rancher"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"nginx.ingress.kubernetes.io/auth-realm": "my.host.com auth"}}
			return nil
		})
	// Expect a call to get the Rancher ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "cattle-system", Name: "rancher"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"nginx.ingress.kubernetes.io/auth-realm": "my.host.com auth"}}
			return nil
		})
	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, virtualservice *istioclinet.VirtualService, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the ingress trait.
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.IngressTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			assert.Len(trait.Status.Resources, 2)
			return nil
		})

	// Create and make the request
	request := newRequest("test-space", "test-trait-name")
	reconciler := newIngressTraitReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestFailureToGetWorkload tests the Reconcile method for the following use case.
// GIVEN a request to reconcile an ingress trait resource
// WHEN the workload related to the trait cannot be found
// THEN ensure that an error is returned
func TestFailureToGetWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	// Expect a call to get the ingress trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.IngressTrait) error {
			trait.TypeMeta = metav1.TypeMeta{
				APIVersion: "oam.verrazzano.io/v1alpha1",
				Kind:       "IngressTrait"}
			trait.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.Rules = []vzapi.IngressRule{{
				Hosts: []string{"test-host"},
				Paths: []vzapi.IngressPath{{Path: "test-path"}}}}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ContainerizedWorkload",
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to get the containerized workload resource and return an error
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "ContainerizedWorkload"}, "test-workload-name")
		})

	// Create and make the request
	request := newRequest("test-space", "test-trait-name")
	reconciler := newIngressTraitReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	assert.Error(err, "Expected and error")
	assert.Contains(err.Error(), "not found")
	assert.Equal(false, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestFailureToGetWorkloadDefinition tests tje Reconcile method for the following use case
// GIVEN a request to reconcile an ingress trait resource
// WHEN the workload definition of the workload related to the trait cannot be found
// THEN ensure that an error is returned
func TestFailureToGetWorkloadDefinition(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	// Expect a call to get the ingress trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.IngressTrait) error {
			trait.TypeMeta = metav1.TypeMeta{
				APIVersion: "oam.verrazzano.io/v1alpha1",
				Kind:       "IngressTrait"}
			trait.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.Rules = []vzapi.IngressRule{{
				Hosts: []string{"test-host"},
				Paths: []vzapi.IngressPath{{Path: "test-path"}}}}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ContainerizedWorkload",
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to get the containerized workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetAPIVersion("core.oam.dev/v1alpha2")
			workload.SetKind("ContainerizedWorkload")
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the containerized workload resource definition and return an error
	mock.EXPECT(). // get workload definition
			Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *v1alpha2.WorkloadDefinition) error {
			return k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: "WorkloadDefinition"}, "containerizedworkloads.core.oam.dev")
		})

	// Create and make the request
	request := newRequest("test-space", "test-trait-name")
	reconciler := newIngressTraitReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	assert.Error(err)
	assert.Contains(err.Error(), "not supported")
	assert.Equal(false, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestFailureToUpdateStatus tests tje Reconcile method for the following use case
// GIVEN a request to reconcile an ingress trait resource
// WHEN the request to update the trait status fails
// THEN ensure an error is returned
func TestFailureToUpdateStatus(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	// Expect a call to get the ingress trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.IngressTrait) error {
			trait.TypeMeta = metav1.TypeMeta{
				APIVersion: "oam.verrazzano.io/v1alpha1",
				Kind:       "IngressTrait"}
			trait.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    map[string]string{oam.LabelAppName: "myapp"}}
			trait.Spec.Rules = []vzapi.IngressRule{{
				Hosts: []string{"test-host"},
				Paths: []vzapi.IngressPath{{Path: "test-path"}}}}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ContainerizedWorkload",
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to get the containerized workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetAPIVersion("core.oam.dev/v1alpha2")
			workload.SetKind("ContainerizedWorkload")
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the containerized workload resource definition
	mock.EXPECT(). // get workload definition
			Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *v1alpha2.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []v1alpha2.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Deployment", list.GetKind())
			return nil
		})
	// Expect a call to list the child Service resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			return appendAsUnstructured(list, v1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: "core.oam.dev/v1alpha2",
						Kind:       "ContainerizedWorkload",
						Name:       "test-workload-name",
						UID:        "test-workload-uid",
					}}}})
		})
	// Expect a call to get the gateway resource related to the ingress trait and return that it is not found.
	mock.EXPECT(). // get ingress (for createOrUpdate)
			Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name-rule-0-gw"}, gomock.Not(gomock.Nil())).
			Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Gateway"}, "test-trait-name-rule-0-gw"))
	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, gateway *istioclinet.Gateway, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the gateway resource related to the ingress trait and return that it is not found.
	mock.EXPECT(). // get ingress (for createOrUpdate)
			Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name-rule-0-vs"}, gomock.Not(gomock.Nil())).
			Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Virtualservice"}, "test-trait-name-rule-0-vs"))
	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, virtualService *istioclinet.VirtualService, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the Rancher ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "cattle-system", Name: "rancher"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"nginx.ingress.kubernetes.io/auth-realm": "my.host.com auth"}}
			return nil
		})
	// Expect a call to get the Rancher ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "cattle-system", Name: "rancher"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"nginx.ingress.kubernetes.io/auth-realm": "my.host.com auth"}}
			return nil
		})
	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus)
	// Expect a call to update the status of the ingress trait.
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.IngressTrait, opts ...client.UpdateOption) error {
			return k8serrors.NewApplyConflict([]metav1.StatusCause{{Type: "test-cause-type", Message: "test-cause-message", Field: "test-cause-field"}}, "test-error-message")
		})

	// Create and make the request
	request := newRequest("test-space", "test-trait-name")
	reconciler := newIngressTraitReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	assert.Error(err)
	assert.Contains(err.Error(), "test-error-message")
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestBuildAppHostNameForDNS tests building a DNS hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is not XIP.IO
// THEN ensure that the correct DNS name is built
func TestBuildAppHostNameForDNS(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	ns := "myns"
	trait := vzapi.IngressTrait{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "oam.verrazzano.io/v1alpha1",
			Kind:       "IngressTrait",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Labels:    map[string]string{oam.LabelAppName: "myapp"},
		},
	}
	// Expect a call to get the Rancher ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "cattle-system", Name: "rancher"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"nginx.ingress.kubernetes.io/auth-realm": "my.host.com auth"}}
			return nil
		})

	// Build the host name
	domainName, err := buildAppHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal("myapp.myns.my.host.com", domainName)
}

// TestBuildAppHostNameIgnoreWildcardForDNS tests building a DNS hostname for the application
// GIVEN an appName and a trait with wildcard hostnames and empty hostnames
// WHEN the buildAppHostName function is called
// THEN ensure that the correct DNS name is built and that the wildcard and empty names are ignored
func TestBuildAppHostNameIgnoreWildcardForDNS(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	ns := "myns"
	trait := vzapi.IngressTrait{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "oam.verrazzano.io/v1alpha1",
			Kind:       "IngressTrait",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Labels:    map[string]string{oam.LabelAppName: "myapp"},
		},
		Spec: vzapi.IngressTraitSpec{
			Rules: []vzapi.IngressRule{{
				Hosts: []string{"*name", "nam*e", "name*", "*", ""},
			}},
		},
	}

	// Expect a call to get the Rancher ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "cattle-system", Name: "rancher"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"nginx.ingress.kubernetes.io/auth-realm": "my.host.com auth"}}
			return nil
		})

	// Build the host name
	domainName, err := buildAppHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal("myapp.myns.my.host.com", domainName)
}

// TestFailureBuildAppHostNameForDNS tests failure of building a DNS hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is not XIP.IO and the rancher annotation is missing
// THEN ensure that an error is returned
func TestFailureBuildAppHostNameForDNS(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	ns := "myns"
	trait := vzapi.IngressTrait{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "oam.verrazzano.io/v1alpha1",
			Kind:       "IngressTrait",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Labels:    map[string]string{oam.LabelAppName: "myapp"},
		},
	}
	// Expect a call to get the Rancher ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "cattle-system", Name: "rancher"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			return nil
		})

	// Build the host name
	_, err := buildAppHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.Error(err)
	assert.Contains(err.Error(), "Annotation nginx.ingress.kubernetes.io/auth-realm missing from Rancher ingress")
}

// TestBuildAppHostNameLoadBalancerXIP tests building a hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is XIP.IO and LoadBalancer is used
// THEN ensure that the correct DNS name is built
func TestBuildAppHostNameLoadBalancerXIP(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	ns := "myns"
	trait := vzapi.IngressTrait{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "oam.verrazzano.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Labels:    map[string]string{oam.LabelAppName: "myapp"},
		},
	}
	// Expect a call to get the Rancher ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "cattle-system", Name: "rancher"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"nginx.ingress.kubernetes.io/auth-realm": "1.2.3.4.xip.io auth"}}
			return nil
		})

	// Expect a call to get the Istio service
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *v1.Service) error {
			service.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1"}
			service.Spec.Type = "LoadBalancer"
			service.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{{
				IP: "5.6.7.8",
			}}
			return nil
		})

	// Build the host name
	domainName, err := buildAppHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal("myapp.myns.5.6.7.8.xip.io", domainName)
}

// TestFailureBuildAppHostNameLoadBalancerXIP tests a failure when building a hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is XIP.IO and LoadBalancer is used, but an error occurs
// THEN ensure that the correct error is returned
func TestFailureBuildAppHostNameLoadBalancerXIP(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	ns := "myns"
	trait := vzapi.IngressTrait{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "oam.verrazzano.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Labels:    map[string]string{oam.LabelAppName: "myapp"},
		},
	}
	// Expect a call to get the Rancher ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "cattle-system", Name: "rancher"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"nginx.ingress.kubernetes.io/auth-realm": "1.2.3.4.xip.io auth"}}
			return nil
		})

	// Expect a call to get the Istio service
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *v1.Service) error {
			service.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1"}
			service.Spec.Type = "LoadBalancer"
			return nil
		})

	// Build the host name
	_, err := buildAppHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.Error(err)
	assert.Equal("istio-ingressgateway is missing loadbalancer IP", err.Error())
}

// TestBuildAppHostNameNodePortXIP tests building a hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is XIP.IO and NodePort is used
// THEN ensure that the correct DNS name is built
func TestBuildAppHostNameNodePortXIP(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	ns := "myns"
	trait := vzapi.IngressTrait{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "oam.verrazzano.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Labels:    map[string]string{oam.LabelAppName: "myapp"},
		},
	}
	// Expect a call to get the Rancher ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "cattle-system", Name: "rancher"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"nginx.ingress.kubernetes.io/auth-realm": "1.2.3.4.xip.io auth"}}
			return nil
		})

	// Expect a call to get the Istio service
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *v1.Service) error {
			service.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1"}
			service.Spec.Type = "NodePort"
			return nil
		})

	// Expect a call to get the Istio ingress gateway pod
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, podList *v1.PodList, opts ...client.ListOption) error {
			podList.Items = []v1.Pod{{
				Status: v1.PodStatus{
					HostIP: "5.6.7.8",
				},
			}}
			return nil
		})

	// Build the host name
	domainName, err := buildAppHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal("myapp.myns.5.6.7.8.xip.io", domainName)
}

// TestFailureBuildAppHostNameNodePortXIP tests a failure when building a hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is XIP.IO and NodePort is used, but an error occurus
// THEN ensure that the correct error is returned
func TestFailureBuildAppHostNameNodePortXIP(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	ns := "myns"
	trait := vzapi.IngressTrait{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "oam.verrazzano.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Labels:    map[string]string{oam.LabelAppName: "myapp"},
		},
	}
	// Expect a call to get the Rancher ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "cattle-system", Name: "rancher"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"nginx.ingress.kubernetes.io/auth-realm": "1.2.3.4.xip.io auth"}}
			return nil
		})

	// Expect a call to get the Istio service
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *v1.Service) error {
			service.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1"}
			service.Spec.Type = "NodePort"
			return nil
		})

	// Expect a call to get the Istio ingress gateway pod
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, podList *v1.PodList, opts ...client.ListOption) error {
			return errors.New("Unable to find istio pods")
		})

	// Build the host name
	_, err := buildAppHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.Error(err)
	assert.Equal("Unable to find istio pods", err.Error())
}

// TestGetTraitFailurePropagated tests tje Reconcile method for the following use case
// GIVEN a request to reconcile an ingress trait resource
// WHEN a failure occurs getting the ingress trait resource
// THEN the error is propagated
func TestGetTraitFailurePropagated(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-name"}, gomock.Any()).
		Return(fmt.Errorf("test-error")).
		AnyTimes()
	reconciler := newIngressTraitReconciler(mock)
	request := newRequest("test-space", "test-name")
	result, err := reconciler.Reconcile(request)
	mocker.Finish()
	assert.Contains(err.Error(), "test-error")
	assert.Equal(false, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestGetNotFoundResource tests the Reconcile method for the following use case.
// GIVEN a request to reconcile an ingress trait resource
// WHEN a failure occurs indicating the resource is not found
// THEN the error is propagated
func TestGetNotFoundResource(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-name"}, gomock.Any()).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "oam.verrazzano.io", Resource: "IngressTrait"}, "test-name")).
		AnyTimes()
	reconciler := newIngressTraitReconciler(mock)
	request := newRequest("test-space", "test-name")
	result, err := reconciler.Reconcile(request)
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestConvertAPIVersionToGroupAndVersion tests multiple use cases for parsing APIVersion
func TestConvertAPIVersionToGroupAndVersion(t *testing.T) {
	assert := asserts.New(t)
	var g, v string

	// GIVEN a normal group/version string
	// WHEN it is parsed into group and version parts
	// THEN ensure the parts are correct.
	g, v = convertAPIVersionToGroupAndVersion("group/version")
	assert.Equal("group", g)
	assert.Equal("version", v)

	// GIVEN a normal group/version string with no group.
	// WHEN it is parsed into group and version parts
	// THEN ensure the group is the empty string and the version is correct.
	// This is the case for older standard kubernetes core resources.
	g, v = convertAPIVersionToGroupAndVersion("/version")
	assert.Equal("", g)
	assert.Equal("version", v)

	// GIVEN a normal group/version string with no group.
	// WHEN it is parsed into group and version parts
	// THEN ensure the group is the empty string and the version is correct.
	// This is the case for older standard kubernetes core resources.
	g, v = convertAPIVersionToGroupAndVersion("version")
	assert.Equal("", g)
	assert.Equal("version", v)
}

// GIVEN a CRs APIVersion and Kind
// WHEN converting to the related CRD namespaced name
// THEN ensure the conversion is correct
func TestConvertCRAPIVersionAndKindToCRDNamespacedName(t *testing.T) {
	assert := asserts.New(t)
	actual := convertAPIVersionAndKindToNamespacedName("core.oam.dev/v1alpha2", "ContainerizedWorkload")
	expect := types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}
	assert.Equal(expect, actual)
}

// TestCreateVirtualServiceMatchUriFromIngressTraitPath tests various use cases of createVirtualServiceMatchURIFromIngressTraitPath
func TestCreateVirtualServiceMatchUriFromIngressTraitPath(t *testing.T) {
	assert := asserts.New(t)
	var path vzapi.IngressPath
	var match *istionet.StringMatch

	// GIVEN an ingress path with normal path and type
	// WHEN a virtual service match uri is created from the ingress path
	// THEN verify the path and type were used correctly
	path = vzapi.IngressPath{Path: "/path", PathType: "exact"}
	match = createVirtualServiceMatchURIFromIngressTraitPath(path)
	assert.IsType(&istionet.StringMatch_Exact{}, match.MatchType)
	assert.Equal("/path", match.MatchType.(*istionet.StringMatch_Exact).Exact)

	// GIVEN an ingress path with path and type with whitespace and upper case
	// WHEN a virtual service match uri is created from the ingress path
	// THEN verify the path and type were updated correctly
	path = vzapi.IngressPath{Path: " /path ", PathType: " PREFIX "}
	match = createVirtualServiceMatchURIFromIngressTraitPath(path)
	assert.IsType(&istionet.StringMatch_Prefix{}, match.MatchType)
	assert.Equal("/path", match.MatchType.(*istionet.StringMatch_Prefix).Prefix)

	// GIVEN an ingress path with no path or type
	// WHEN a virtual service match uri is created from the ingress path
	// THEN verify the path and type were defaulted correctly
	path = vzapi.IngressPath{}
	match = createVirtualServiceMatchURIFromIngressTraitPath(path)
	assert.IsType(&istionet.StringMatch_Prefix{}, match.MatchType)
	assert.Equal("/", match.MatchType.(*istionet.StringMatch_Prefix).Prefix)

	// GIVEN an ingress path with only a path / and no type
	// WHEN a virtual service match uri is created from the ingress path
	// THEN verify the type were defaulted correctly to prefix
	path = vzapi.IngressPath{Path: "/"}
	match = createVirtualServiceMatchURIFromIngressTraitPath(path)
	assert.IsType(&istionet.StringMatch_Prefix{}, match.MatchType)
	assert.Equal("/", match.MatchType.(*istionet.StringMatch_Prefix).Prefix)

	// GIVEN an ingress path with only a path and no type
	// WHEN a virtual service match uri is created from the ingress path
	// THEN verify the type were defaulted correctly to exact
	path = vzapi.IngressPath{Path: "/path"}
	match = createVirtualServiceMatchURIFromIngressTraitPath(path)
	assert.IsType(&istionet.StringMatch_Exact{}, match.MatchType)
	assert.Equal("/path", match.MatchType.(*istionet.StringMatch_Exact).Exact)
}

// TestCreateHostsFromIngressTraitRule tests various use cases of createHostsFromIngressTraitRule
func TestCreateHostsFromIngressTraitRule(t *testing.T) {
	assert := asserts.New(t)
	var rule vzapi.IngressRule
	var hosts []string

	// GIVEN a trait rule with no hosts
	// WHEN a host slice is requested for use
	// THEN verify that the default host is used
	rule = vzapi.IngressRule{}
	hosts = createHostsFromIngressTraitRule(rule, "defaultHost")
	assert.Len(hosts, 1)
	assert.Equal("defaultHost", hosts[0])

	// GIVEN a trait rule with a mix of hosts including an empty host
	// WHEN a host slice is requested for use
	// THEN verify that the empty host is ignored and the defaultHost is not used
	rule = vzapi.IngressRule{Hosts: []string{"host-1", "", "host-2"}}
	hosts = createHostsFromIngressTraitRule(rule, "defaultHost")
	assert.Len(hosts, 2)
	assert.Equal("host-1", hosts[0])
	assert.Equal("host-2", hosts[1])

	// GIVEN a trait rule with only wildcard hosts and an empty host
	// WHEN a host slice is requested for use
	// THEN verify that only the default host is used
	rule = vzapi.IngressRule{Hosts: []string{"*", "", "*host", "host*", "ho*st"}}
	hosts = createHostsFromIngressTraitRule(rule, "defaultHost")
	assert.Len(hosts, 1)
	assert.Equal("defaultHost", hosts[0])
}

// TestGetPathsFromTrait tests various use cases of getPathsFromRule
func TestGetPathsFromTrait(t *testing.T) {
	assert := asserts.New(t)
	var rule vzapi.IngressRule
	var paths []vzapi.IngressPath

	// GIVEN an ingress rule with no path or type
	// WHEN the paths are obtained from the rule
	// THEN verify that path and type are defaulted
	rule = vzapi.IngressRule{}
	paths = getPathsFromRule(rule)
	assert.Len(paths, 1)
	assert.Equal("/", paths[0].Path)
	assert.Equal("prefix", paths[0].PathType)

	// GIVEN an ingress rule with valid path and type
	// WHEN the paths are obtained from the rule
	// THEN verify that path and type are the same.
	rule = vzapi.IngressRule{Paths: []vzapi.IngressPath{{
		Path:     "/test-path-name",
		PathType: "test-path-type",
	}}}
	paths = getPathsFromRule(rule)
	assert.Len(paths, 1)
	assert.Equal("/test-path-name", paths[0].Path)
	assert.Equal("test-path-type", paths[0].PathType)
}

// TestCreateDestinationFromService test various use cases of createDestinationFromService
func TestCreateDestinationFromService(t *testing.T) {
	assert := asserts.New(t)
	var service v1.Service
	var dest istionet.HTTPRouteDestination

	// GIVEN a service with no ports defined
	// WHEN a destination is created from the service
	// THEN verify that the port is nil.
	service = v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"}}
	dest = createDestinationFromService(&service)
	assert.Equal("test-service-name", dest.Destination.Host)
	assert.Nil(dest.Destination.Port)

	// GIVEN a service with a valid port defined
	// WHEN a destination is created from the service
	// THEN verify that the service's port is used.
	service = v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec:       v1.ServiceSpec{Ports: []v1.ServicePort{{Port: 42}}}}
	dest = createDestinationFromService(&service)
	assert.Equal(uint32(42), dest.Destination.Port.Number)

	// GIVEN a service with multiple valid ports defined
	// WHEN a destination is created from the service
	// THEN verify that the service's first port is used.
	service = v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec:       v1.ServiceSpec{Ports: []v1.ServicePort{{Port: 42}, {Port: 777}}}}
	dest = createDestinationFromService(&service)
	assert.Equal(uint32(42), dest.Destination.Port.Number)
}

// GIVEN a single service in the unstructured children list
// WHEN extracting the service
// THEN ensure the returned service is the child from the list
func TestExtractServiceOnlyOneService(t *testing.T) {
	assert := asserts.New(t)

	var serviceID types.UID = "test-service-1"
	u, err := newUnstructuredService(serviceID, clusterIPNone)
	assert.NoError(err)

	children := []*unstructured.Unstructured{&u}
	var extractedService *v1.Service
	extractedService, err = extractServiceFromUnstructuredChildren(children)

	assert.NoError(err)
	assert.Equal(serviceID, extractedService.GetObjectMeta().GetUID())
}

// GIVEN multiple services in the unstructured children list
// WHEN extracting the service
// THEN ensure the returned service is the first one with a cluster IP
func TestExtractServiceMultipleServices(t *testing.T) {
	assert := asserts.New(t)

	u1, err := newUnstructuredService("test-service-1", clusterIPNone)
	assert.NoError(err)

	var serviceID types.UID = "test-service-2"
	u2, err := newUnstructuredService(serviceID, "10.0.0.1")
	assert.NoError(err)

	u3, err := newUnstructuredService("test-service-3", "10.0.0.2")
	assert.NoError(err)

	children := []*unstructured.Unstructured{&u1, &u2, &u3}
	var extractedService *v1.Service
	extractedService, err = extractServiceFromUnstructuredChildren(children)

	assert.NoError(err)
	assert.Equal(serviceID, extractedService.GetObjectMeta().GetUID())
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	//_ = clientgoscheme.AddToScheme(scheme)
	//_ = core.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
	return scheme
}

// newIngressTraitReconciler creates a new reconciler for testing
// c - The Kerberos client to inject into the reconciler
func newIngressTraitReconciler(c client.Client) Reconciler {
	log := ctrl.Log.WithName("test")
	scheme := newScheme()
	reconciler := Reconciler{
		Client: c,
		Log:    log,
		Scheme: scheme}
	return reconciler
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name}}
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

// newUnstructuredService creates a service and returns it in Unstructured form
// uid - The UID of the service
// clusterIP - The cluster IP of the service
func newUnstructuredService(uid types.UID, clusterIP string) (unstructured.Unstructured, error) {
	service := v1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID: uid,
		},
		Spec: v1.ServiceSpec{
			ClusterIP: clusterIP,
		}}

	return convertToUnstructured(service)
}

func fakeGenerateDNSHostName(_ client.Reader, _ *vzapi.IngressTrait) (string, error) {
	return "fakehost", nil
}
