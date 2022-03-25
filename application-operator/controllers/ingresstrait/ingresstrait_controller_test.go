// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingresstrait

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"text/template"
	"time"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/golang/mock/gomock"
	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	istionet "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
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
	mock.EXPECT().
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
			assert.Equal("DeploymentList", list.GetKind())
			return nil
		})
	// Expect a call to list the child Service resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("ServiceList", list.GetKind())
			return appendAsUnstructured(list, k8score.Service{
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
					}}},
				Spec: k8score.ServiceSpec{
					ClusterIP: "10.11.12.13",
					Ports:     []k8score.ServicePort{{Port: 42}}}})
		})
	// Expect a call to create the certificate and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, certificate *certapiv1.Certificate, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the certificate related to the ingress trait
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "test-space-myapp-cert"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Certificate"}, "test-space-myapp-cert"))
	// Expect a call to get the app config and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "myapp"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, app *v1alpha2.ApplicationConfiguration) error {
			app.TypeMeta = metav1.TypeMeta{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ApplicationConfiguration",
			}
			return nil
		})
	// Expect a call to get the gateway resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-space-myapp-gw"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Gateway"}, "test-space-myapp-gw"))
	// Expect a call to create the Gateway resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, gateway *istioclient.Gateway, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the virtual service resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name-rule-0-vs"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "VirtualService"}, "test-trait-name-rule-0-vs"))

	// Expect a call to create the VirtualService resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, virtualservice *istioclient.VirtualService, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the IngressTrait.
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.IngressTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			assert.Len(trait.Status.Resources, 3)
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

// TestSuccessfullyCreateNewIngressWithCertSecret tests the Reconcile method for the following use case.
// GIVEN a request to reconcile an ingress trait resource that specifies a certificate secret to use for security
// WHEN the trait exists but the ingress does not
// THEN ensure that the trait is created.
func TestSuccessfullyCreateNewIngressWithCertSecret(t *testing.T) {
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
			trait.Spec.TLS = vzapi.IngressSecurity{SecretName: "cert-secret"}
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
	mock.EXPECT().
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
			assert.Equal("DeploymentList", list.GetKind())
			return nil
		})
	// Expect a call to list the child Service resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("ServiceList", list.GetKind())
			return appendAsUnstructured(list, k8score.Service{
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
					}}},
				Spec: k8score.ServiceSpec{
					ClusterIP: "10.11.12.13",
					Ports:     []k8score.ServicePort{{Port: 42}}},
			})
		})
	// Expect a call to get the gateway resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-space-myapp-gw"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Gateway"}, "test-space-myapp-gw"))
	// Expect a call to create the ingress/gateway resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, gateway *istioclient.Gateway, opts ...client.CreateOption) error {
			assert.Equal(istionet.ServerTLSSettings_SIMPLE, gateway.Spec.Servers[0].Tls.Mode, "Wrong Tls Mode")
			assert.Equal("cert-secret", gateway.Spec.Servers[0].Tls.CredentialName, "Wrong secret name")
			return nil
		})
	// Expect a call to get the app config and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "myapp"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, app *v1alpha2.ApplicationConfiguration) error {
			app.TypeMeta = metav1.TypeMeta{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ApplicationConfiguration",
			}
			return nil
		})
	// Expect a call to get the virtual service resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name-rule-0-vs"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "VirtualService"}, "test-trait-name-rule-0-vs"))

	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, virtualservice *istioclient.VirtualService, opts ...client.CreateOption) error {
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

// TestSuccessfullyUpdateIngressWithCertSecret tests the Reconcile method for the following use case.
// GIVEN a request to reconcile an ingress trait resource that specifies a certificate secret to use for security
// WHEN the trait and ingress/gateway exist
// THEN ensure that the trait is updated with the expected hosts.
func TestSuccessfullyUpdateIngressWithCertSecret(t *testing.T) {
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
				Hosts: []string{"Test-host"},
				Paths: []vzapi.IngressPath{{Path: "test-path"}}}}
			trait.Spec.TLS = vzapi.IngressSecurity{SecretName: "cert-secret"}
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
	mock.EXPECT().
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
			assert.Equal("DeploymentList", list.GetKind())
			return nil
		})
	// Expect a call to list the child Service resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("ServiceList", list.GetKind())
			return appendAsUnstructured(list, k8score.Service{
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
					}}},
				Spec: k8score.ServiceSpec{
					ClusterIP: "10.11.12.13",
					Ports:     []k8score.ServicePort{{Port: 42}}},
			})
		})
	// Expect a call to get the gateway resource related to the ingress trait and return it.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-space-myapp-gw"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, gateway *istioclient.Gateway) error {
			gateway.TypeMeta = metav1.TypeMeta{
				APIVersion: gatewayAPIVersion,
				Kind:       gatewayKind}
			gateway.ObjectMeta = metav1.ObjectMeta{
				Namespace: "test-space",
				Name:      "test-space-myapp-gw"}
			gateway.Spec = istionet.Gateway{
				Servers: []*istionet.Server{{
					Port: &istionet.Port{
						Name:     "https",
						Number:   443,
						Protocol: "HTTPS"},
					Hosts: []string{"test-host" /*, "test2-host", "test3-host"*/},
				}}}
			return nil
		})
	// Expect a call to create the ingress/gateway resource and return success
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, gateway *istioclient.Gateway, opts ...client.CreateOption) error {
			assert.Equal(istionet.ServerTLSSettings_SIMPLE, gateway.Spec.Servers[0].Tls.Mode, "Wrong Tls Mode")
			assert.Equal("cert-secret", gateway.Spec.Servers[0].Tls.CredentialName, "Wrong secret name")
			assert.Contains(gateway.Spec.Servers[0].Hosts, "test-host", "doesn't contain expected host")
			//assert.Contains(gateway.Spec.Servers[0].Hosts, "test2-host", "doesn't contain expected host")
			//assert.Contains(gateway.Spec.Servers[0].Hosts, "test3-host", "doesn't contain expected host")
			return nil
		})
	// Expect a call to get the app config and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "myapp"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, app *v1alpha2.ApplicationConfiguration) error {
			app.TypeMeta = metav1.TypeMeta{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ApplicationConfiguration",
			}
			return nil
		})
	// Expect a call to get the virtual service resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name-rule-0-vs"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "VirtualService"}, "test-trait-name-rule-0-vs"))

	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, virtualservice *istioclient.VirtualService, opts ...client.CreateOption) error {
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

// TestFailureCreateNewIngressWithSecretNoHosts tests the Reconcile method for the following use case.
// GIVEN a request to reconcile an ingress trait resource that specifies a certificate secret to use for security
// WHEN the secret is specified but no associated hosts are configured
// THEN ensure that the trait creation fails
func TestFailureCreateNewIngressWithSecretNoHosts(t *testing.T) {
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
				Paths: []vzapi.IngressPath{{Path: "test-path"}}}}
			trait.Spec.TLS = vzapi.IngressSecurity{SecretName: "cert-secret"}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ContainerizedWorkload",
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the ingress trait.  The status is checked for the expected error condition.
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.IngressTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			assert.Equal("all rules must specify at least one host when a secret is specified for TLS transport", trait.Status.Conditions[0].Message, "Unexpected error message")
			assert.Len(trait.Status.Resources, 1)
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

// TestFailureCreateGatewayCertNoAppName tests the Reconcile method for the following use case.
// GIVEN a request to reconcile an ingress trait resource
// WHEN the trait exists but doesn't specify an oam app label
// THEN ensure that an error is generated
func TestFailureCreateGatewayCertNoAppName(t *testing.T) {
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

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the ingress trait.  The status is checked for the expected error condition.
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.IngressTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			assert.Equal("failed to obtain app name from ingress trait", trait.Status.Conditions[0].Message, "Unexpected error message")
			assert.Len(trait.Status.Resources, 0)
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

// TestSuccessfullyCreateNewIngressForVerrazzanoWorkload tests the Reconcile method for the following use case.
// GIVEN a request to reconcile an ingress trait resource that applies to a Verrazzano workload type
// WHEN the trait exists but the ingress does not
// THEN ensure that the workload is unwrapped and the trait is created.
func TestSuccessfullyCreateNewIngressForVerrazzanoWorkload(t *testing.T) {
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
				Paths: []vzapi.IngressPath{{Path: "test-path"}},
				Destination: vzapi.IngressDestination{
					Host: "test-service.test-space.svc.local",
					Port: 0,
				}}}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: "oam.verrazzano.io/v1alpha1",
				Kind:       "VerrazzanoCoherenceWorkload",
				Name:       "test-workload-name"}
			return nil
		})

	containedName := "test-contained-workload-name"
	containedResource := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": containedName,
		},
	}

	// Expect a call to get the Verrazzano Coherence workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetAPIVersion("oam.verrazzano.io/v1alpha1")
			workload.SetKind("VerrazzanoCoherenceWorkload")
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			unstructured.SetNestedMap(workload.Object, containedResource, "spec", "template")
			return nil
		})
	// Expect a call to get the contained Coherence resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: containedName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetUnstructuredContent(containedResource)
			workload.SetNamespace(name.Namespace)
			workload.SetAPIVersion("coherence.oracle.com/v1")
			workload.SetKind("Coherence")
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the containerized workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "coherences.coherence.oracle.com"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *v1alpha2.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []v1alpha2.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("DeploymentList", list.GetKind())
			return nil
		})
	// Expect a call to get the certificate related to the ingress trait
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "test-space-myapp-cert"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Certificate"}, "test-space-myapp-cert"))
	// Expect a call to list the child Service resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("ServiceList", list.GetKind())
			return appendAsUnstructured(list, k8score.Service{
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
					}}},
				Spec: k8score.ServiceSpec{
					ClusterIP: "10.11.12.13",
					Ports:     []k8score.ServicePort{{Port: 42}}},
			})
		})
	// Expect a call to get the app config and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "myapp"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, app *v1alpha2.ApplicationConfiguration) error {
			app.TypeMeta = metav1.TypeMeta{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ApplicationConfiguration",
			}
			return nil
		})
	// Expect a call to create the certificate and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, certificate *certapiv1.Certificate, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the gateway resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-space-myapp-gw"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Gateway"}, "test-space-myapp-gw"))
	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, gateway *istioclient.Gateway, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the virtual service resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name-rule-0-vs"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "VirtualService"}, "test-trait-name-rule-0-vs"))
	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, virtualservice *istioclient.VirtualService, opts ...client.CreateOption) error {
			assert.Len(virtualservice.Spec.Http, 1)
			assert.Len(virtualservice.Spec.Http[0].Route, 1)
			assert.Equal("test-service.test-space.svc.local", virtualservice.Spec.Http[0].Route[0].Destination.Host)
			return nil
		})
	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the ingress trait.
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.IngressTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			assert.Len(trait.Status.Resources, 3)
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
	// Expect a call to create the certificate and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, certificate *certapiv1.Certificate, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the certificate related to the ingress trait
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "test-space-myapp-cert"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Certificate"}, "test-space-myapp-cert"))

	// Expect a call to get the gateway resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-space-myapp-gw"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Gateway"}, "test-space-myapp-gw"))
	// Expect a call to create the gateway and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, gateway *istioclient.Gateway, opts ...client.CreateOption) error {
			return nil
		})

	// Expect a call to get the app config
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "myapp"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, app *v1alpha2.ApplicationConfiguration) error {
			app.TypeMeta = metav1.TypeMeta{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ApplicationConfiguration",
			}
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
	assert.Nil(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Milliseconds(), time.Duration(0).Milliseconds())
}

// TestFailureToGetWorkloadDefinition tests the Reconcile method for the following use case
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
	// Expect a call to create the certificate and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, certificate *certapiv1.Certificate, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the certificate related to the ingress trait
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "test-space-myapp-cert"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Certificate"}, "test-space-myapp-cert"))

	// Expect a call to get the gateway resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-space-myapp-gw"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Gateway"}, "test-space-myapp-gw"))
	// Expect a call to create the gateway and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, gateway *istioclient.Gateway, opts ...client.CreateOption) error {
			return nil
		})

	// Expect a call to get the app config
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "myapp"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, app *v1alpha2.ApplicationConfiguration) error {
			app.TypeMeta = metav1.TypeMeta{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ApplicationConfiguration",
			}
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
	mock.EXPECT().
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
	assert.Nil(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Milliseconds(), time.Duration(0).Milliseconds())
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
	mock.EXPECT().
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
			assert.Equal("DeploymentList", list.GetKind())
			return nil
		})
	// Expect a call to list the child Service resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("ServiceList", list.GetKind())
			return appendAsUnstructured(list, k8score.Service{
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
					}}},
				Spec: k8score.ServiceSpec{
					ClusterIP: "10.11.12.13",
					Ports:     []k8score.ServicePort{{Port: 42}}},
			})
		})
	// Expect a call to get the certificate related to the ingress trait
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "test-space-myapp-cert"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Certificate"}, "test-space-myapp-cert"))
	// Expect a call to get the app config and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "myapp"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, app *v1alpha2.ApplicationConfiguration) error {
			app.TypeMeta = metav1.TypeMeta{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ApplicationConfiguration",
			}
			return nil
		})
	// Expect a call to create the certificate and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, certificate *certapiv1.Certificate, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the gateway resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-space-myapp-gw"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Gateway"}, "test-space-myapp-gw"))
	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, gateway *istioclient.Gateway, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the gateway resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name-rule-0-vs"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Virtualservice"}, "test-trait-name-rule-0-vs"))
	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, virtualService *istioclient.VirtualService, opts ...client.CreateOption) error {
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
	assert.Nil(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Milliseconds(), time.Duration(0).Milliseconds())
}

// TestBuildAppHostNameForDNS tests building a DNS hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is not nip.io
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
	// Expect a call to get the Verrazzano ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"external-dns.alpha.kubernetes.io/target": "verrazzano-ingress.my.host.com"}}
			return nil
		})

	// Build the host name
	domainName, err := buildAppFullyQualifiedHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal("myapp.myns.my.host.com", domainName)
}

// TestBuildAppHostNameIgnoreWildcardForDNS tests building a DNS hostname for the application
// GIVEN an appName and a trait with wildcard hostnames and empty hostnames
// WHEN the buildAppFullyQualifiedHostName function is called
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

	// Expect a call to get the Verrazzano ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"external-dns.alpha.kubernetes.io/target": "verrazzano-ingress.my.host.com"}}
			return nil
		})

	// Build the host name
	domainName, err := buildAppFullyQualifiedHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal("myapp.myns.my.host.com", domainName)
}

// TestFailureBuildAppHostNameForDNS tests failure of building a DNS hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is not nip.io and the Verrazzano annotation is missing
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
	// Expect a call to get the Verrazzano ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			return nil
		})

	// Build the host name
	_, err := buildAppFullyQualifiedHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.Error(err)
	assert.Contains(err.Error(), "Annotation external-dns.alpha.kubernetes.io/target missing from Verrazzano ingress")
}

// TestBuildAppHostNameLoadBalancerNIP tests building a hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is nip.io and LoadBalancer is used
// THEN ensure that the correct DNS name is built
func TestBuildAppHostNameLoadBalancerNIP(t *testing.T) {
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
	// Expect a call to get the Verrazzano ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Annotations: map[string]string{
					"external-dns.alpha.kubernetes.io/target": "verrazzano-ingress.1.2.3.4.nip.io",
					"verrazzano.io/dns.wildcard.domain":       "nip.io",
				},
			}
			return nil
		})

	// Expect a call to get the Istio service
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *k8score.Service) error {
			service.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1"}
			service.Spec.Type = "LoadBalancer"
			service.Status.LoadBalancer.Ingress = []k8score.LoadBalancerIngress{{
				IP: "5.6.7.8",
			}}
			return nil
		})

	// Build the host name
	domainName, err := buildAppFullyQualifiedHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal("myapp.myns.5.6.7.8.nip.io", domainName)
}

// TestBuildAppHostNameExternalLoadBalancerNIP tests building a hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is nip.io and an external LoadBalancer is used
// THEN ensure that the correct DNS name is built
func TestBuildAppHostNameExternalLoadBalancerNIP(t *testing.T) {
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
	// Expect a call to get the Verrazzano ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Annotations: map[string]string{
					"external-dns.alpha.kubernetes.io/target": "verrazzano-ingress.1.2.3.4.nip.io",
					"verrazzano.io/dns.wildcard.domain":       "nip.io",
				},
			}
			return nil
		})

	// Expect a call to get the Istio service
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *k8score.Service) error {
			service.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1"}
			service.Spec.Type = "LoadBalancer"
			service.Spec.ExternalIPs = []string{"5.6.7.8"}
			return nil
		})

	// Build the host name
	domainName, err := buildAppFullyQualifiedHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal("myapp.myns.5.6.7.8.nip.io", domainName)
}

// TestBuildAppHostNameExternalLoadBalancerNIPNotFound tests building a hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is nip.io and an external LoadBalancer is used, but no IP is found
// THEN ensure that an error is returned
func TestBuildAppHostNameExternalLoadBalancerNIPNotFound(t *testing.T) {
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
	// Expect a call to get the Verrazzano ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Annotations: map[string]string{
					"external-dns.alpha.kubernetes.io/target": "verrazzano-ingress.1.2.3.4.nip.io",
					"verrazzano.io/dns.wildcard.domain":       "nip.io",
				},
			}
			return nil
		})

	// Expect a call to get the Istio service
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *k8score.Service) error {
			service.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1"}
			service.Spec.Type = "LoadBalancer"
			return nil
		})

	// Build the host name
	_, err := buildAppFullyQualifiedHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.Error(err)
}

// TestFailureBuildAppHostNameLoadBalancerNIP tests a failure when building a hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is nip.io and LoadBalancer is used, but an error occurs
// THEN ensure that the correct error is returned
func TestFailureBuildAppHostNameLoadBalancerNIP(t *testing.T) {
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
	// Expect a call to get the Verrazzano ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Annotations: map[string]string{
					"external-dns.alpha.kubernetes.io/target": "verrazzano-ingress.1.2.3.4.nip.io",
					"verrazzano.io/dns.wildcard.domain":       "nip.io",
				},
			}
			return nil
		})

	// Expect a call to get the Istio service
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *k8score.Service) error {
			service.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1"}
			service.Spec.Type = "LoadBalancer"
			return nil
		})

	// Build the host name
	_, err := buildAppFullyQualifiedHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.Error(err)
	assert.Equal("istio-ingressgateway is missing loadbalancer IP", err.Error())
}

// TestBuildAppHostNameNodePortNIP tests building a hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is nip.io and NodePort is used
// THEN ensure that the correct DNS name is built
func TestBuildAppHostNameNodePortNIP(t *testing.T) {
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
	// Expect a call to get the Verrazzano ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Annotations: map[string]string{
					"external-dns.alpha.kubernetes.io/target": "verrazzano-ingress.1.2.3.4.nip.io",
					"verrazzano.io/dns.wildcard.domain":       "nip.io",
				},
			}
			return nil
		})

	// Expect a call to get the Istio service
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *k8score.Service) error {
			service.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1"}
			service.Spec.Type = "NodePort"
			return nil
		})

	// Expect a call to get the Istio ingress gateway pod
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, podList *k8score.PodList, opts ...client.ListOption) error {
			podList.Items = []k8score.Pod{{
				Status: k8score.PodStatus{
					HostIP: "5.6.7.8",
				},
			}}
			return nil
		})

	// Build the host name
	domainName, err := buildAppFullyQualifiedHostName(mock, &trait)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal("myapp.myns.5.6.7.8.nip.io", domainName)
}

// TestFailureBuildAppHostNameNodePortNIP tests a failure when building a hostname for the application
// GIVEN an appName and a trait
// WHEN the ingress domain is nip.io and NodePort is used, but an error occurus
// THEN ensure that the correct error is returned
func TestFailureBuildAppHostNameNodePortNIP(t *testing.T) {
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
	// Expect a call to get the Verrazzano ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Annotations: map[string]string{
					"external-dns.alpha.kubernetes.io/target": "verrazzano-ingress.1.2.3.4.nip.io",
					"verrazzano.io/dns.wildcard.domain":       "nip.io",
				},
			}
			return nil
		})

	// Expect a call to get the Istio service
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *k8score.Service) error {
			service.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1"}
			service.Spec.Type = "NodePort"
			return nil
		})

	// Expect a call to get the Istio ingress gateway pod
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, podList *k8score.PodList, opts ...client.ListOption) error {
			return errors.New("Unable to find istio pods")
		})

	// Build the host name
	_, err := buildAppFullyQualifiedHostName(mock, &trait)

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
	assert.Nil(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Milliseconds(), time.Duration(0).Milliseconds())
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
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "oam.verrazzano.io", Resource: "IngressTrait"}, "test-name"))
	reconciler := newIngressTraitReconciler(mock)
	request := newRequest("test-space", "test-name")
	result, err := reconciler.Reconcile(request)
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
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

// TestCreateHostsFromIngressTraitRule tests generation of a default host name
// GIVEN a trait rule with only wildcard hosts and an empty host
// WHEN a host slice DNS domain exists in the ingress
// THEN verify that only the default host is used
func TestCreateHostsFromIngressTraitRuleWildcards(t *testing.T) {
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

	// Expect a call to get the Verrazzano ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   name.Namespace,
				Name:        name.Name,
				Annotations: map[string]string{"external-dns.alpha.kubernetes.io/target": "verrazzano-ingress.my.host.com"}}
			return nil
		})

	rule := vzapi.IngressRule{Hosts: []string{"*", "", "*host", "host*", "ho*st"}}
	hosts, err := createHostsFromIngressTraitRule(mock, rule, &trait)

	mocker.Finish()
	assert.NoError(err)
	assert.Len(hosts, 1)
	assert.Equal("myapp.myns.my.host.com", hosts[0])
}

// TestCreateHostsFromIngressTraitRule tests various use cases of createHostsFromIngressTraitRule
func TestCreateHostsFromIngressTraitRule(t *testing.T) {
	assert := asserts.New(t)
	var rule vzapi.IngressRule
	var hosts []string

	// GIVEN a trait rule with a valid hosts
	// WHEN a host slice is requested for use
	// THEN verify that valid hosts are used
	rule = vzapi.IngressRule{Hosts: []string{"host-1", "host-2"}}
	hosts, err := createHostsFromIngressTraitRule(nil, rule, nil)
	assert.NoError(err)
	assert.Len(hosts, 2)
	assert.Equal("host-1", hosts[0])
	assert.Equal("host-2", hosts[1])

	// GIVEN a trait rule with a mix of hosts including an empty host and wildcard host
	// WHEN a host slice is requested for use
	// THEN verify that the empty host is ignored and the defaultHost is not used
	rule = vzapi.IngressRule{Hosts: []string{"host-1", "", "*", "host-2"}}
	hosts, err = createHostsFromIngressTraitRule(nil, rule, nil)
	assert.NoError(err)
	assert.Len(hosts, 2)
	assert.Equal("host-1", hosts[0])
	assert.Equal("host-2", hosts[1])
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
	var services []*k8score.Service
	var dest *istionet.HTTPRouteDestination

	// GIVEN one service with no cluster-IP defined
	// WHEN a destination is created from the service
	// THEN verify that destination created successfully
	service1 := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"}}
	services = append(services, &service1)
	dest, err := createDestinationFromService(services)
	assert.Equal("test-service-name", dest.Destination.Host)
	assert.Nil(dest.Destination.Port)
	assert.NoError(err)

	// GIVEN a service with no ports defined
	// WHEN a destination is created from the service
	// THEN verify that the port is nil.
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec:       k8score.ServiceSpec{ClusterIP: "10.10.10.3"}}
	services[0] = &service1
	dest, err = createDestinationFromService(services)
	assert.Equal("test-service-name", dest.Destination.Host)
	assert.Nil(dest.Destination.Port)
	assert.NoError(err)

	// GIVEN a service with a valid port defined
	// WHEN a destination is created from the service
	// THEN verify that the service's port is used.
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec:       k8score.ServiceSpec{ClusterIP: "10.10.10.3", Ports: []k8score.ServicePort{{Port: 42}}}}
	services[0] = &service1
	dest, err = createDestinationFromService(services)
	assert.Equal(uint32(42), dest.Destination.Port.Number)
	assert.NoError(err)

	// GIVEN a service with multiple valid ports defined
	// WHEN a destination is created from the service
	// THEN verify that the service's port with name having "http" prefix is used.
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.3",
			Ports: []k8score.ServicePort{{Port: 42}, {Port: 777, Name: "http"}}}}
	services[0] = &service1
	dest, err = createDestinationFromService(services)
	assert.Equal("test-service-name", dest.Destination.Host)
	assert.Equal(uint32(777), dest.Destination.Port.Number)
	assert.NoError(err)

	// GIVEN a service with multiple valid ports defined and none of them named with "http" prefix
	// WHEN a destination is created from the service
	// THEN verify that an error is returned.
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.3",
			Ports: []k8score.ServicePort{{Port: 42}, {Port: 777}}}}
	services[0] = &service1
	dest, err = createDestinationFromService(services)
	assert.Nil(dest, "No destination should have been created")
	assert.Error(err, "An error should have been returned")

	// GIVEN a service with multiple valid ports defined and many of them named with "http" prefix
	// WHEN a destination is created from the service
	// THEN verify that an error is returned.
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.3",
			Ports: []k8score.ServicePort{{Port: 42, Name: "http-1"}, {Port: 777, Name: "http"}}}}
	services[0] = &service1
	dest, err = createDestinationFromService(services)
	assert.Nil(dest, "No destination should have been created")
	assert.Error(err, "An error should have been returned")

	// GIVEN multiple services and one of them having a port name with the prefix "http"
	// WHEN a destination is created from the service
	// THEN verify that destination created successfully using the service with the prefix "http"
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec:       k8score.ServiceSpec{Ports: []k8score.ServicePort{{Port: 42}}}}
	service2 := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test1-service-name"},
		Spec:       k8score.ServiceSpec{Ports: []k8score.ServicePort{{Name: "http", Port: 777}}}}
	services = append(services, &service2)
	services[0] = &service1
	dest, err = createDestinationFromService(services)
	assert.Equal("test1-service-name", dest.Destination.Host)
	assert.Equal(uint32(777), dest.Destination.Port.Number)
	assert.NoError(err)

	// GIVEN multiple services defined and many of them having the port names with the prefix "http"
	// WHEN a destination is created from the service
	// THEN verify that an error is returned
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.2",
			Ports: []k8score.ServicePort{{Port: 42}, {Port: 777, Name: "metrics"}}}}
	service2 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service1-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.3",
			Ports: []k8score.ServicePort{{Port: 777}}}}
	services[0] = &service1
	services[1] = &service2
	dest, err = createDestinationFromService(services)
	assert.Nil(dest, "No destination should have been created")
	assert.Error(err, "An error should have been returned")

	// GIVEN multiple services defined and more than one having ports named with the prefix "http"
	// WHEN a destination is created from the service
	// THEN verify that an error is returned
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "http-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.2",
			Ports: []k8score.ServicePort{{Port: 42, Name: "http"}, {Port: 777, Name: "metrics"}}}}
	service2 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "http-service1-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.3",
			Ports: []k8score.ServicePort{{Port: 777, Name: "http1"}}}}
	services[0] = &service1
	services[1] = &service2
	dest, err = createDestinationFromService(services)
	assert.Nil(dest, "No destination should have been created")
	assert.Error(err, "An error should have been returned")

	// GIVEN no services
	// WHEN a destination is created from the service
	// THEN verify that function fails
	dest, err = createDestinationFromService(nil)
	assert.Nil(dest, "No destination should have been created")
	assert.Error(err, "An error should have been returned")
}

// TestCreateDestinationForWeblogicWorkload test various use cases of createDestinationFromService for weblogic workload
func TestCreateDestinationForWeblogicWorkload(t *testing.T) {
	assert := asserts.New(t)
	var services []*k8score.Service
	var dest *istionet.HTTPRouteDestination

	// GIVEN a weblogic workload service with one weblogic port defined
	// WHEN a destination is created from the service
	// THEN verify that the destination is created successfully
	service1 := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{Selector: map[string]string{"weblogic.createdByOperator": "true"},
			ClusterIP: "10.10.10.3",
			Ports:     []k8score.ServicePort{{Port: 42, Name: "tcp-1"}, {Port: 777, Name: "tcp-ldap"}}}}
	services = append(services, &service1)
	dest, err := createDestinationFromService(services)
	assert.Equal(uint32(777), dest.Destination.Port.Number)
	assert.NoError(err)

	// GIVEN a weblogic workload service with one http port defined
	// WHEN a destination is created from the service
	// THEN verify that the destination is created successfully
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{Selector: map[string]string{"weblogic.createdByOperator": "true"},
			ClusterIP: "10.10.10.3",
			Ports:     []k8score.ServicePort{{Port: 42, Name: "tcp-1"}, {Port: 777, Name: "http-default"}}}}
	services[0] = &service1
	dest, err = createDestinationFromService(services)
	assert.Equal(uint32(777), dest.Destination.Port.Number)
	assert.NoError(err)

	// GIVEN a weblogic workload service with two known weblogic http ports defined
	// WHEN a destination is created from the service
	// THEN verify that the destination creation fails
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{Selector: map[string]string{"weblogic.createdByOperator": "true"},
			ClusterIP: "10.10.10.3",
			Ports:     []k8score.ServicePort{{Port: 42, Name: "tcp-cbt"}, {Port: 777, Name: "tcp-ldap"}}}}
	services[0] = &service1
	dest, err = createDestinationFromService(services)
	assert.Nil(dest, "No destination should have been created")
	assert.Error(err, "An error should have been returned")

	// GIVEN a weblogic workload service with one weblogic port defined but not created by operator
	// WHEN a destination is created from the service
	// THEN verify that the destination creation fails
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{Selector: nil,
			ClusterIP: "10.10.10.3",
			Ports:     []k8score.ServicePort{{Port: 42, Name: "tcp-test"}, {Port: 777, Name: "tcp-ldap"}}}}
	services[0] = &service1
	dest, err = createDestinationFromService(services)
	assert.Nil(dest, "No destination should have been created")
	assert.Error(err, "An error should have been returned")
}

// TestCreateDestinationFromRuleOrService test various use cases of createDestinationFromRuleOrService
func TestCreateDestinationFromRuleOrService(t *testing.T) {
	assert := asserts.New(t)
	var rule vzapi.IngressRule
	var services []*k8score.Service
	var dest *istionet.HTTPRouteDestination

	// GIVEN a rule and service with a valid port defined
	// WHEN a destination is created from the rule or service
	// THEN verify that the host and port used are that of the one defined in the rule.
	rule = vzapi.IngressRule{
		Destination: vzapi.IngressDestination{Host: "test-host", Port: 77}}
	service1 := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec:       k8score.ServiceSpec{Ports: []k8score.ServicePort{{Port: 42, Name: "test-port"}}}}
	services = append(services, &service1)
	dest, err := createDestinationFromRuleOrService(rule, services)
	assert.Equal("test-host", dest.Destination.Host)
	assert.Equal(uint32(77), dest.Destination.Port.Number)
	assert.NoError(err)

	// GIVEN a service and a rule with only valid port defined but not host
	// WHEN a destination is created from the rule or service
	// THEN verify that the host used is that of the one defined in the service for the corresponding port.
	rule = vzapi.IngressRule{
		Destination: vzapi.IngressDestination{Port: 77}}
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.2",
			Ports: []k8score.ServicePort{{Port: 88, Name: "test1-port"}, {Port: 77, Name: "test2-port"}}}}
	services[0] = &service1
	dest, err = createDestinationFromRuleOrService(rule, services)
	assert.Equal("test-service-name", dest.Destination.Host)
	assert.Equal(uint32(77), dest.Destination.Port.Number)
	assert.NoError(err)

	// GIVEN a service and a rule with only valid port defined but not host
	// WHEN a destination is created from the rule or service
	// THEN an error is returned if there is no corresponding service exists with that rule port
	rule = vzapi.IngressRule{
		Destination: vzapi.IngressDestination{Port: 77}}
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.2",
			Ports: []k8score.ServicePort{{Port: 42, Name: "test-port"}}}}
	services[0] = &service1
	dest, err = createDestinationFromRuleOrService(rule, services)
	assert.Nil(dest, "No destination should have been created")
	assert.Error(err, "An error should have been returned")

	// GIVEN a rule without destination defined and multiple ports defined for a service
	// WHEN a destination is created from the rule or service
	// THEN verify that the port with name having "http" prefix is used.
	rule = vzapi.IngressRule{}
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.2",
			Ports: []k8score.ServicePort{{Port: 42, Name: "metrics"}, {Port: 77, Name: "http"}}}}
	services[0] = &service1
	dest, err = createDestinationFromRuleOrService(rule, services)
	assert.Equal("test-service-name", dest.Destination.Host)
	assert.Equal(uint32(77), dest.Destination.Port.Number)
	assert.NoError(err)

	// GIVEN a rule without destination and multiple ports defined for a service and none of them have "http" prefix
	// WHEN a destination is created from the rule or service
	// THEN verify that an error is returned
	rule = vzapi.IngressRule{}
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.2",
			Ports: []k8score.ServicePort{{Port: 42, Name: "metrics"}, {Port: 77, Name: "test"}}}}
	services[0] = &service1
	dest, err = createDestinationFromRuleOrService(rule, services)
	assert.Nil(dest, "No destination should have been created")
	assert.Error(err, "An error should have been returned")

	// GIVEN multiple services with same port and rule with only port defined
	// WHEN a destination is created from the rule or service
	// THEN verify that the service having port name with the prefix "http" is used.
	rule = vzapi.IngressRule{
		Destination: vzapi.IngressDestination{Port: 777}}
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.2",
			Ports: []k8score.ServicePort{{Port: 777, Name: "test-port"}}}}
	service2 := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "http-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.3",
			Ports: []k8score.ServicePort{{Port: 42, Name: "metrics"}, {Port: 777, Name: "http"}}}}
	services[0] = &service1
	services = append(services, &service2)
	dest, err = createDestinationFromRuleOrService(rule, services)
	assert.Equal("http-service-name", dest.Destination.Host)
	assert.Equal(uint32(777), dest.Destination.Port.Number)
	assert.NoError(err)

	// GIVEN multiple services and rule with only port defined
	// WHEN a destination is created from the rule or service
	// THEN verify that the service corresponding to rule port is used than the one having the port name with
	// the prefix "http".
	rule = vzapi.IngressRule{
		Destination: vzapi.IngressDestination{Port: 77}}
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.2",
			Ports: []k8score.ServicePort{{Port: 77, Name: "test-port"}}}}
	service2 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "http-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.3",
			Ports: []k8score.ServicePort{{Port: 42, Name: "http"}, {Port: 777}}}}
	services[0] = &service1
	services[1] = &service2
	dest, err = createDestinationFromRuleOrService(rule, services)
	assert.Equal("test-service-name", dest.Destination.Host)
	assert.Equal(uint32(77), dest.Destination.Port.Number)
	assert.NoError(err)

	// GIVEN a rule without destination defined and multiple services defined
	// WHEN a destination is created from the rule or service
	// THEN verify that the service with prefix "http" is used.
	rule = vzapi.IngressRule{}
	service1 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.2",
			Ports: []k8score.ServicePort{{Port: 42, Name: "test-port"}}}}
	service2 = k8score.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test1-service-name"},
		Spec: k8score.ServiceSpec{ClusterIP: "10.10.10.3",
			Ports: []k8score.ServicePort{{Port: 777, Name: "http-port"}}}}
	services[0] = &service1
	services[1] = &service2
	dest, err = createDestinationFromRuleOrService(rule, services)
	assert.Equal("test1-service-name", dest.Destination.Host)
	assert.Equal(uint32(777), dest.Destination.Port.Number)
	assert.NoError(err)
}

// GIVEN a single service in the unstructured children list
// WHEN extracting the services
// THEN ensure the returned service is the child from the list
func TestExtractServicesOnlyOneService(t *testing.T) {
	assert := asserts.New(t)

	workload := &unstructured.Unstructured{}
	workload.SetAPIVersion("apps/v1")
	workload.SetKind("Deployment")
	workload.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "oam.verrazzano.io/v1alpha1", Kind: "VerrazzanoHelidonWorkload"}})

	var serviceID types.UID = "test-service-1"
	u, err := newUnstructuredService(serviceID, "11.12.13.14", 777)
	assert.NoError(err)

	children := []*unstructured.Unstructured{&u}
	var extractedServices []*k8score.Service
	reconciler := Reconciler{}
	log := vzlog.DefaultLogger()
	extractedServices, err = reconciler.extractServicesFromUnstructuredChildren(children, log)
	assert.NoError(err)
	assert.NotNil(extractedServices)
	assert.Equal(len(extractedServices), 1)
	assert.Equal(serviceID, extractedServices[0].GetObjectMeta().GetUID())
}

// GIVEN multiple services in the unstructured children list
// WHEN extracting the services
// THEN ensure the returned services has details of all the services
func TestExtractServicesMultipleServices(t *testing.T) {
	assert := asserts.New(t)

	workload := &unstructured.Unstructured{}
	updateUnstructuredFromYAMLTemplate(workload, "test/templates/wls_domain_instance.yaml", nil)

	var service1ID types.UID = "test-service-1"
	u1, err := newUnstructuredService(service1ID, clusterIPNone, 8001)
	assert.NoError(err)

	var service2ID types.UID = "test-service-2"
	u2, err := newUnstructuredService(service2ID, "10.0.0.1", 8002)
	assert.NoError(err)

	var service3ID types.UID = "test-service-3"
	u3, err := newUnstructuredService(service3ID, "10.0.0.2", 8003)
	assert.NoError(err)

	children := []*unstructured.Unstructured{&u1, &u2, &u3}
	var extractedServices []*k8score.Service
	reconciler := Reconciler{}
	log := vzlog.DefaultLogger()
	extractedServices, err = reconciler.extractServicesFromUnstructuredChildren(children, log)
	assert.NoError(err)
	assert.NotNil(extractedServices)
	assert.Equal(len(extractedServices), 3)
	assert.Equal(service1ID, extractedServices[0].GetObjectMeta().GetUID())
	assert.Equal(service2ID, extractedServices[1].GetObjectMeta().GetUID())
	assert.Equal(service3ID, extractedServices[2].GetObjectMeta().GetUID())
}

// Test a valid existing Service is discovered and used for the destination.
// GIVEN a valid existing Service for a workload
// WHEN an ingress trait is reconciled
// THEN verify gateway and virtual service are created correctly.
func TestSelectExistingServiceForVirtualServiceDestination(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewFakeClientWithScheme(newScheme())
	params := map[string]string{
		"NAMESPACE_NAME":      "test-namespace",
		"APPCONF_NAME":        "test-appconf",
		"APPCONF_NAMESPACE":   "test-namespace",
		"COMPONENT_NAME":      "test-comp",
		"COMPONENT_NAMESPACE": "test-namespace",
		"TRAIT_NAME":          "test-trait",
		"TRAIT_NAMESPACE":     "test-namespace",
		"WORKLOAD_NAME":       "test-workload",
		"WORKLOAD_NAMESPACE":  "test-namespace",
		"WORKLOAD_KIND":       "VerrazzanoWebLogicWorkload",
		"DOMAIN_NAME":         "test-domain",
		"DOMAIN_NAMESPACE":    "test-namespace",
		"DOMAIN_UID":          "test-domain-uid",
	}

	// Create namespace
	assert.NoError(createResourceFromTemplate(cli, "test/templates/managed_namespace.yaml", params))
	// Create Verrazzano ingress
	assert.NoError(cli.Create(context.Background(), newVerrazzanoIngress("verrazzano-ingress.1.2.3.4")))
	// Create Istio ingress service
	assert.NoError(cli.Create(context.Background(), newIstioLoadBalancerService("10.11.12.13", "1.2.3.4")))
	// Create application configuration
	assert.NoError(createResourceFromTemplate(cli, "test/templates/appconf_with_ingress.yaml", params))
	// Create application component
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_component.yaml", params))
	// Create WebLogic workload definition
	assert.NoError(createResourceFromTemplate(cli, "deploy/workloaddefinition_wls.yaml", params))
	// Create trait
	assert.NoError(createResourceFromTemplate(cli, "test/templates/ingress_trait_instance.yaml", params))
	// Create workload
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_workload_instance.yaml", params))
	// Create domain
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_domain_instance.yaml", params))
	// Create a service
	service := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: params["NAMESPACE_NAME"],
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "weblogic.oracle/v8",
				Kind:       "Domain",
				Name:       params["DOMAIN_NAME"],
				UID:        types.UID(params["DOMAIN_UID"]),
			}},
		},
		Spec: k8score.ServiceSpec{
			Ports: []k8score.ServicePort{{
				Name:       "default",
				Protocol:   "TCP",
				Port:       8001,
				TargetPort: intstr.FromInt(8001),
			}},
			ClusterIP: "10.11.12.13",
			Type:      "ClusterIP",
		},
	}
	assert.NoError(cli.Create(context.Background(), &service))

	// Perform Reconcile
	request := newRequest(params["TRAIT_NAMESPACE"], params["TRAIT_NAME"])
	reconciler := newIngressTraitReconciler(cli)
	result, err := reconciler.Reconcile(request)
	assert.NoError(err)
	assert.Equal(true, result.Requeue, "Expected a requeue due to status update.")

	gw := istioclient.Gateway{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-namespace-test-appconf-gw"}, &gw)
	assert.NoError(err)
	assert.Equal("ingressgateway", gw.Spec.Selector["istio"])
	assert.Equal("test-appconf.test-namespace.1.2.3.4.nip.io", gw.Spec.Servers[0].Hosts[0])
	assert.Equal("https", gw.Spec.Servers[0].Port.Name)
	assert.Equal(uint32(443), gw.Spec.Servers[0].Port.Number)
	assert.Equal("HTTPS", gw.Spec.Servers[0].Port.Protocol)
	assert.Equal("test-namespace-test-appconf-cert-secret", gw.Spec.Servers[0].Tls.CredentialName)
	assert.Equal("SIMPLE", gw.Spec.Servers[0].Tls.Mode.String())

	vs := istioclient.VirtualService{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-trait-rule-0-vs"}, &vs)
	assert.NoError(err)
	assert.Equal("test-namespace-test-appconf-gw", vs.Spec.Gateways[0])
	assert.Len(vs.Spec.Gateways, 1)
	assert.Equal("test-appconf.test-namespace.1.2.3.4.nip.io", vs.Spec.Hosts[0])
	assert.Len(vs.Spec.Hosts, 1)
	assert.Contains(vs.Spec.Http[0].Match[0].Uri.String(), "prefix:")
	assert.Contains(vs.Spec.Http[0].Match[0].Uri.String(), "/bobbys-front-end")
	assert.Len(vs.Spec.Http[0].Match, 1)
	assert.Equal("test-service", vs.Spec.Http[0].Route[0].Destination.Host)
	assert.Equal(uint32(8001), vs.Spec.Http[0].Route[0].Destination.Port.Number)
	assert.Len(vs.Spec.Http[0].Route, 1)
	assert.Len(vs.Spec.Http, 1)
}

// Test an explicitly provided destination is used in preference to an existing Service.
// GIVEN an ingress trait containing an explicit destination
// WHEN the ingress trait is reconciled
// THEN verify the correct gateway and virtual services are created.
func TestExplicitServiceProvidedForVirtualServiceDestination(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewFakeClientWithScheme(newScheme())
	params := map[string]string{
		"NAMESPACE_NAME":      "test-namespace",
		"APPCONF_NAME":        "test-appconf",
		"APPCONF_NAMESPACE":   "test-namespace",
		"COMPONENT_NAME":      "test-comp",
		"COMPONENT_NAMESPACE": "test-namespace",
		"TRAIT_NAME":          "test-trait",
		"TRAIT_NAMESPACE":     "test-namespace",
		"WORKLOAD_NAME":       "test-workload",
		"WORKLOAD_NAMESPACE":  "test-namespace",
		"WORKLOAD_KIND":       "VerrazzanoWebLogicWorkload",
		"DOMAIN_NAME":         "test-domain",
		"DOMAIN_NAMESPACE":    "test-namespace",
		"DOMAIN_UID":          "test-domain-uid",
	}

	// Create namespace
	assert.NoError(createResourceFromTemplate(cli, "test/templates/managed_namespace.yaml", params))
	// Create Verrazzano ingress
	assert.NoError(cli.Create(context.Background(), newVerrazzanoIngress("1.2.3.4")))
	// Create Istio ingress service
	assert.NoError(cli.Create(context.Background(), newIstioLoadBalancerService("10.11.12.13", "1.2.3.4")))
	// Create application configuration
	assert.NoError(createResourceFromTemplate(cli, "test/templates/appconf_with_ingress.yaml", params))
	// Create application component
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_component.yaml", params))
	// Create WebLogic workload definition
	assert.NoError(createResourceFromTemplate(cli, "deploy/workloaddefinition_wls.yaml", params))
	// Create trait
	assert.NoError(createResourceFromTemplate(cli, "test/templates/ingress_trait_instance_with_dest.yaml", params))
	// Create workload
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_workload_instance.yaml", params))
	// Create domain
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_domain_instance.yaml", params))
	// Create a service. This service should be ignored as an explicit destination is provided.
	service := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: params["NAMESPACE_NAME"],
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "weblogic.oracle/v8",
				Kind:       "Domain",
				Name:       params["DOMAIN_NAME"],
				UID:        types.UID(params["DOMAIN_UID"]),
			}},
		},
		Spec: k8score.ServiceSpec{
			Ports: []k8score.ServicePort{{
				Name:       "default",
				Protocol:   "TCP",
				Port:       8001,
				TargetPort: intstr.FromInt(8001),
			}},
			ClusterIP: "10.11.12.13",
			Type:      "ClusterIP",
		},
	}
	assert.NoError(cli.Create(context.Background(), &service))

	// Perform Reconcile
	request := newRequest(params["TRAIT_NAMESPACE"], params["TRAIT_NAME"])
	reconciler := newIngressTraitReconciler(cli)
	result, err := reconciler.Reconcile(request)
	assert.NoError(err)
	assert.Equal(true, result.Requeue, "Expected a requeue due to status update.")

	gw := istioclient.Gateway{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-namespace-test-appconf-gw"}, &gw)
	assert.NoError(err)
	assert.Equal("ingressgateway", gw.Spec.Selector["istio"])
	assert.Equal("test-appconf.test-namespace.1.2.3.4.nip.io", gw.Spec.Servers[0].Hosts[0])
	assert.Equal("https", gw.Spec.Servers[0].Port.Name)
	assert.Equal(uint32(443), gw.Spec.Servers[0].Port.Number)
	assert.Equal("HTTPS", gw.Spec.Servers[0].Port.Protocol)
	assert.Equal("test-namespace-test-appconf-cert-secret", gw.Spec.Servers[0].Tls.CredentialName)
	assert.Equal("SIMPLE", gw.Spec.Servers[0].Tls.Mode.String())

	vs := istioclient.VirtualService{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-trait-rule-0-vs"}, &vs)
	assert.NoError(err)
	assert.Equal("test-namespace-test-appconf-gw", vs.Spec.Gateways[0])
	assert.Len(vs.Spec.Gateways, 1)
	assert.Equal("test-appconf.test-namespace.1.2.3.4.nip.io", vs.Spec.Hosts[0])
	assert.Len(vs.Spec.Hosts, 1)
	assert.Contains(vs.Spec.Http[0].Match[0].Uri.String(), "prefix:")
	assert.Contains(vs.Spec.Http[0].Match[0].Uri.String(), "/test-path")
	assert.Len(vs.Spec.Http[0].Match, 1)
	assert.Equal("test-dest-host", vs.Spec.Http[0].Route[0].Destination.Host)
	assert.Equal(uint32(777), vs.Spec.Http[0].Route[0].Destination.Port.Number)
	assert.Len(vs.Spec.Http[0].Route, 1)
	assert.Len(vs.Spec.Http, 1)
}

// Test failure for multiple service ports without an explicit destination.
// GIVEN a service with multiple ports exists for a workload
// AND no explicit ingress trait definitions are provided
// WHEN the ingress trait is reconciled
// THEN verify the correct gateway and virtual services are created.
func TestMultiplePortsOnDiscoveredService(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewFakeClientWithScheme(newScheme())
	params := map[string]string{
		"NAMESPACE_NAME":      "test-namespace",
		"APPCONF_NAME":        "test-appconf",
		"APPCONF_NAMESPACE":   "test-namespace",
		"COMPONENT_NAME":      "test-comp",
		"COMPONENT_NAMESPACE": "test-namespace",
		"TRAIT_NAME":          "test-trait",
		"TRAIT_NAMESPACE":     "test-namespace",
		"WORKLOAD_NAME":       "test-workload",
		"WORKLOAD_NAMESPACE":  "test-namespace",
		"WORKLOAD_KIND":       "VerrazzanoWebLogicWorkload",
		"DOMAIN_NAME":         "test-domain",
		"DOMAIN_NAMESPACE":    "test-namespace",
		"DOMAIN_UID":          "test-domain-uid",
	}

	// Create namespace
	assert.NoError(createResourceFromTemplate(cli, "test/templates/managed_namespace.yaml", params))
	// Create Verrazzano ingress
	assert.NoError(cli.Create(context.Background(), newVerrazzanoIngress("1.2.3.4")))
	// Create Istio ingress service
	assert.NoError(cli.Create(context.Background(), newIstioLoadBalancerService("10.11.12.13", "1.2.3.4")))
	// Create application configuration
	assert.NoError(createResourceFromTemplate(cli, "test/templates/appconf_with_ingress.yaml", params))
	// Create application component
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_component.yaml", params))
	// Create WebLogic workload definition
	assert.NoError(createResourceFromTemplate(cli, "deploy/workloaddefinition_wls.yaml", params))
	// Create trait. This trait has no destination.
	assert.NoError(createResourceFromTemplate(cli, "test/templates/ingress_trait_instance.yaml", params))
	// Create workload
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_workload_instance.yaml", params))
	// Create domain
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_domain_instance.yaml", params))
	// Create a service. This service has two ports and one with "http" prefix.
	service := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: params["NAMESPACE_NAME"],
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "weblogic.oracle/v8",
				Kind:       "Domain",
				Name:       params["DOMAIN_NAME"],
				UID:        types.UID(params["DOMAIN_UID"]),
			}},
		},
		Spec: k8score.ServiceSpec{
			Ports: []k8score.ServicePort{{
				Name:       "default",
				Protocol:   "TCP",
				Port:       8001,
				TargetPort: intstr.FromInt(8001)}, {
				Name:       "http",
				Protocol:   "TCP",
				Port:       8002,
				TargetPort: intstr.FromInt(8002)},
			},
			ClusterIP: "10.11.12.13",
			Type:      "ClusterIP",
		},
	}
	assert.NoError(cli.Create(context.Background(), &service))

	// Perform Reconcile
	request := newRequest(params["TRAIT_NAMESPACE"], params["TRAIT_NAME"])
	reconciler := newIngressTraitReconciler(cli)
	result, err := reconciler.Reconcile(request)
	assert.NoError(err, "No error because reconcile worked but needs to be retried.")
	assert.Equal(true, result.Requeue, "Expected a requeue because the discovered service has multiple ports.")

	gw := istioclient.Gateway{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-namespace-test-appconf-gw"}, &gw)
	assert.NoError(err)
	assert.Equal("ingressgateway", gw.Spec.Selector["istio"])
	assert.Equal("test-appconf.test-namespace.1.2.3.4.nip.io", gw.Spec.Servers[0].Hosts[0])
	assert.Equal("https", gw.Spec.Servers[0].Port.Name)
	assert.Equal(uint32(443), gw.Spec.Servers[0].Port.Number)
	assert.Equal("HTTPS", gw.Spec.Servers[0].Port.Protocol)
	assert.Equal("test-namespace-test-appconf-cert-secret", gw.Spec.Servers[0].Tls.CredentialName)
	assert.Equal("SIMPLE", gw.Spec.Servers[0].Tls.Mode.String())

	vs := istioclient.VirtualService{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-trait-rule-0-vs"}, &vs)
	assert.NoError(err)
	assert.Equal("test-namespace-test-appconf-gw", vs.Spec.Gateways[0])
	assert.Len(vs.Spec.Gateways, 1)
	assert.Equal("test-appconf.test-namespace.1.2.3.4.nip.io", vs.Spec.Hosts[0])
	assert.Len(vs.Spec.Hosts, 1)
	assert.Contains(vs.Spec.Http[0].Match[0].Uri.String(), "prefix:")
	assert.Contains(vs.Spec.Http[0].Match[0].Uri.String(), "/bobbys-front-end")
	assert.Len(vs.Spec.Http[0].Match, 1)
	assert.Equal("test-service", vs.Spec.Http[0].Route[0].Destination.Host)
	assert.Equal(8002, int(vs.Spec.Http[0].Route[0].Destination.Port.Number))
	assert.Len(vs.Spec.Http[0].Route, 1)
	assert.Len(vs.Spec.Http, 1)
}

// Test failure for multiple services for non-WebLogic workload without explicit destination.
// GIVEN multiple services created for a non-WebLogic workload
// AND no explicit ingress trait definitions are provided
// WHEN the ingress trait is reconciled
// THEN verify the correct gateway and virtual services are created.
func TestMultipleServicesForNonWebLogicWorkloadWithoutExplicitIngressDestination(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewFakeClientWithScheme(newScheme())
	params := map[string]string{
		"NAMESPACE_NAME":        "test-namespace",
		"APPCONF_NAME":          "test-appconf",
		"APPCONF_NAMESPACE":     "test-namespace",
		"APPCONF_UID":           "test-appconf-uid",
		"COMPONENT_NAME":        "test-comp",
		"COMPONENT_NAMESPACE":   "test-namespace",
		"TRAIT_NAME":            "test-trait",
		"TRAIT_NAMESPACE":       "test-namespace",
		"WORKLOAD_NAME":         "test-workload",
		"WORKLOAD_NAMESPACE":    "test-namespace",
		"WORKLOAD_UID":          "test-workload-uid",
		"WORKLOAD_KIND":         "VerrazzanoHelidonWorkload",
		"DEPLOYMENT_NAME":       "test-deployment",
		"DEPLOYMENT_NAMESPACE":  "test-namespace",
		"DEPLOYMENT_UID":        "test-domain-uid",
		"CONTAINER_NAME":        "test-container-name",
		"CONTAINER_IMAGE":       "test-container-image",
		"CONTAINER_PORT_NAME":   "test-container-port-name",
		"CONTAINER_PORT_NUMBER": "777",
	}

	// Create namespace
	assert.NoError(createResourceFromTemplate(cli, "test/templates/managed_namespace.yaml", params))
	// Create Verrazzano ingress
	assert.NoError(cli.Create(context.Background(), newVerrazzanoIngress("1.2.3.4")))
	// Create Istio ingress service
	assert.NoError(cli.Create(context.Background(), newIstioLoadBalancerService("10.11.12.13", "1.2.3.4")))
	// Create application configuration
	assert.NoError(createResourceFromTemplate(cli, "test/templates/appconf_with_ingress.yaml", params))
	// Create application component
	assert.NoError(createResourceFromTemplate(cli, "test/templates/helidon_component.yaml", params))
	// Create WebLogic workload definition
	assert.NoError(createResourceFromTemplate(cli, "deploy/workloaddefinition_vzhelidon.yaml", params))
	// Create workload
	assert.NoError(createResourceFromTemplate(cli, "test/templates/helidon_workload_instance.yaml", params))
	// Create trait. This trait has no destination.
	assert.NoError(createResourceFromTemplate(cli, "test/templates/ingress_trait_instance.yaml", params))
	// Create a first service.
	service1 := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-1",
			Namespace: params["APPCONF_NAMESPACE"],
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "oam.verrazzano.io/v1alpha1",
				Kind:       "VerrazzanoHelidonWorkload",
				Name:       params["WORKLOAD_NAME"],
				UID:        types.UID(params["WORKLOAD_UID"]),
			}},
		},
		Spec: k8score.ServiceSpec{
			Ports: []k8score.ServicePort{{
				Name:       "test-service-1-port",
				Protocol:   "TCP",
				Port:       8081,
				TargetPort: intstr.FromInt(8081)},
			},
			ClusterIP: "10.11.12.13",
			Type:      "NodePort",
		},
	}
	assert.NoError(cli.Create(context.Background(), &service1))
	// Create a second service.
	service2 := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-2",
			Namespace: params["APPCONF_NAMESPACE"],
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "oam.verrazzano.io/v1alpha1",
				Kind:       "VerrazzanoHelidonWorkload",
				Name:       params["WORKLOAD_NAME"],
				UID:        types.UID(params["WORKLOAD_UID"]),
			}},
		},
		Spec: k8score.ServiceSpec{
			Ports: []k8score.ServicePort{{
				Name:       "http-service-2-port",
				Protocol:   "TCP",
				Port:       8082,
				TargetPort: intstr.FromInt(8082)},
			},
			ClusterIP: "11.12.13.14",
			Type:      "NodePort",
		},
	}
	assert.NoError(cli.Create(context.Background(), &service2))

	// Perform Reconcile
	request := newRequest(params["TRAIT_NAMESPACE"], params["TRAIT_NAME"])
	reconciler := newIngressTraitReconciler(cli)
	result, err := reconciler.Reconcile(request)
	assert.NoError(err, "No error because reconcile worked but needs to be retried.")
	assert.Equal(true, result.Requeue, "Expected a requeue because the discovered service has multiple ports.")

	gw := istioclient.Gateway{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-namespace-test-appconf-gw"}, &gw)
	assert.NoError(err)
	assert.Equal("ingressgateway", gw.Spec.Selector["istio"])
	assert.Equal("test-appconf.test-namespace.1.2.3.4.nip.io", gw.Spec.Servers[0].Hosts[0])
	assert.Equal("https", gw.Spec.Servers[0].Port.Name)
	assert.Equal(uint32(443), gw.Spec.Servers[0].Port.Number)
	assert.Equal("HTTPS", gw.Spec.Servers[0].Port.Protocol)
	assert.Equal("test-namespace-test-appconf-cert-secret", gw.Spec.Servers[0].Tls.CredentialName)
	assert.Equal("SIMPLE", gw.Spec.Servers[0].Tls.Mode.String())

	vs := istioclient.VirtualService{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-trait-rule-0-vs"}, &vs)
	assert.NoError(err)
	assert.Equal("test-namespace-test-appconf-gw", vs.Spec.Gateways[0])
	assert.Len(vs.Spec.Gateways, 1)
	assert.Equal("test-appconf.test-namespace.1.2.3.4.nip.io", vs.Spec.Hosts[0])
	assert.Len(vs.Spec.Hosts, 1)
	assert.Contains(vs.Spec.Http[0].Match[0].Uri.String(), "prefix:")
	assert.Contains(vs.Spec.Http[0].Match[0].Uri.String(), "/bobbys-front-end")
	assert.Len(vs.Spec.Http[0].Match, 1)
	assert.Equal("test-service-2", vs.Spec.Http[0].Route[0].Destination.Host)
	assert.Equal(8082, int(vs.Spec.Http[0].Route[0].Destination.Port.Number))
	assert.Len(vs.Spec.Http[0].Route, 1)
	assert.Len(vs.Spec.Http, 1)
}

// Test correct WebLogic service (i.e. with ClusterIP) getting picked after reconcile failure and retry.
// GIVEN a new WebLogic workload/domain
// AND no services have been created
// WHEN an ingress trait is reconciled
// THEN ensure that no gateways or virtual services are created
// THEN create a service as the WebLogic operator would
// THEN verity that the expected gateway and virtual services are created.
func TestSelectExistingServiceForVirtualServiceDestinationAfterRetry(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewFakeClientWithScheme(newScheme())
	params := map[string]string{
		"NAMESPACE_NAME":      "test-namespace",
		"APPCONF_NAME":        "test-appconf",
		"APPCONF_NAMESPACE":   "test-namespace",
		"COMPONENT_NAME":      "test-comp",
		"COMPONENT_NAMESPACE": "test-namespace",
		"TRAIT_NAME":          "test-trait",
		"TRAIT_NAMESPACE":     "test-namespace",
		"WORKLOAD_NAME":       "test-workload",
		"WORKLOAD_NAMESPACE":  "test-namespace",
		"WORKLOAD_KIND":       "VerrazzanoWebLogicWorkload",
		"DOMAIN_NAME":         "test-domain",
		"DOMAIN_NAMESPACE":    "test-namespace",
		"DOMAIN_UID":          "test-domain-uid",
	}

	// Create namespace
	assert.NoError(createResourceFromTemplate(cli, "test/templates/managed_namespace.yaml", params))
	// Create Verrazzano ingress
	assert.NoError(cli.Create(context.Background(), newVerrazzanoIngress("1.2.3.4")))
	// Create Istio ingress service
	assert.NoError(cli.Create(context.Background(), newIstioLoadBalancerService("10.11.12.13", "1.2.3.4")))
	// Create application configuration
	assert.NoError(createResourceFromTemplate(cli, "test/templates/appconf_with_ingress.yaml", params))
	// Create application component
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_component.yaml", params))
	// Create WebLogic workload definition
	assert.NoError(createResourceFromTemplate(cli, "deploy/workloaddefinition_wls.yaml", params))
	// Create trait
	assert.NoError(createResourceFromTemplate(cli, "test/templates/ingress_trait_instance.yaml", params))
	// Create workload
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_workload_instance.yaml", params))
	// Create domain
	assert.NoError(createResourceFromTemplate(cli, "test/templates/wls_domain_instance.yaml", params))

	// Perform Reconcile
	request := newRequest(params["TRAIT_NAMESPACE"], params["TRAIT_NAME"])
	reconciler := newIngressTraitReconciler(cli)
	result, err := reconciler.Reconcile(request)
	assert.NoError(err)
	assert.Equal(true, result.Requeue, "Expected no requeue as error expected.")

	gw := istioclient.Gateway{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-namespace-test-appconf-gw"}, &gw)
	assert.False(k8serrors.IsNotFound(err), "Gateway should have been created.")

	vs := istioclient.VirtualService{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-trait-rule-0-vs"}, &vs)
	assert.True(k8serrors.IsNotFound(err), "No VirtualService should have been created.")

	// Update a service. Update the ClusterIP of the service.
	service := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: params["NAMESPACE_NAME"],
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "weblogic.oracle/v8",
				Kind:       "Domain",
				Name:       params["DOMAIN_NAME"],
				UID:        types.UID(params["DOMAIN_UID"]),
			}},
		},
		Spec: k8score.ServiceSpec{
			Ports: []k8score.ServicePort{{
				Name:       "default",
				Protocol:   "TCP",
				Port:       8001,
				TargetPort: intstr.FromInt(8001),
			}},
			ClusterIP: "10.11.12.13",
			Type:      "ClusterIP",
		},
	}
	assert.NoError(cli.Create(context.Background(), &service))

	// Reconcile again.
	result, err = reconciler.Reconcile(request)
	assert.NoError(err)
	assert.Equal(true, result.Requeue, "Expected requeue as status was updated.")

	// Verify the Gateway was created and is valid.
	gw = istioclient.Gateway{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-namespace-test-appconf-gw"}, &gw)
	assert.NoError(err)
	assert.Equal("ingressgateway", gw.Spec.Selector["istio"])
	assert.Equal("test-appconf.test-namespace.1.2.3.4.nip.io", gw.Spec.Servers[0].Hosts[0])
	assert.Equal("https", gw.Spec.Servers[0].Port.Name)
	assert.Equal(uint32(443), gw.Spec.Servers[0].Port.Number)
	assert.Equal("HTTPS", gw.Spec.Servers[0].Port.Protocol)
	assert.Equal("test-namespace-test-appconf-cert-secret", gw.Spec.Servers[0].Tls.CredentialName)
	assert.Equal("SIMPLE", gw.Spec.Servers[0].Tls.Mode.String())

	// Verify the VirtualService was created and is valid.
	vs = istioclient.VirtualService{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "test-namespace", Name: "test-trait-rule-0-vs"}, &vs)
	assert.NoError(err)
	assert.Equal("test-namespace-test-appconf-gw", vs.Spec.Gateways[0])
	assert.Len(vs.Spec.Gateways, 1)
	assert.Equal("test-appconf.test-namespace.1.2.3.4.nip.io", vs.Spec.Hosts[0])
	assert.Len(vs.Spec.Hosts, 1)
	assert.Contains(vs.Spec.Http[0].Match[0].Uri.String(), "prefix:")
	assert.Contains(vs.Spec.Http[0].Match[0].Uri.String(), "/bobbys-front-end")
	assert.Len(vs.Spec.Http[0].Match, 1)
	assert.Equal("test-service", vs.Spec.Http[0].Route[0].Destination.Host)
	assert.Equal(uint32(8001), vs.Spec.Http[0].Route[0].Destination.Port.Number)
	assert.Len(vs.Spec.Http[0].Route, 1)
	assert.Len(vs.Spec.Http, 1)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	//_ = clientgoscheme.AddToScheme(scheme)
	core.AddToScheme(scheme)
	k8sapps.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
	k8score.AddToScheme(scheme)
	certapiv1.AddToScheme(scheme)
	k8net.AddToScheme(scheme)
	istioclient.AddToScheme(scheme)
	return scheme
}

// newIngressTraitReconciler creates a new reconciler for testing
// c - The Kerberos client to inject into the reconciler
func newIngressTraitReconciler(c client.Client) Reconciler {
	log := zap.S().With("test")
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

// newVerrazzanoIngress creates a new Ranger Ingress with the provided IP address.
func newVerrazzanoIngress(ipAddress string) *k8net.Ingress {
	rangerIngress := k8net.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.VzConsoleIngress,
			Namespace: constants.VerrazzanoSystemNamespace,
			Annotations: map[string]string{
				"external-dns.alpha.kubernetes.io/target": fmt.Sprintf("verrazzano-ingress.default.%s.nip.io", ipAddress),
				"verrazzano.io/dns.wildcard.domain":       "nip.io",
			},
		},
	}
	return &rangerIngress
}

// newIstioLoadBalancerService creates a new Istio LoadBalancer Service with the provided
// clusterIPAddress and loadBalancerIPAddress
func newIstioLoadBalancerService(clusterIPAddress string, loadBalancerIPAddress string) *k8score.Service {
	istioService := k8score.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-ingressgateway",
			Namespace: "istio-system",
		},
		Spec: k8score.ServiceSpec{
			ClusterIP: clusterIPAddress,
			Type:      "LoadBalancer",
		},
		Status: k8score.ServiceStatus{
			LoadBalancer: k8score.LoadBalancerStatus{
				Ingress: []k8score.LoadBalancerIngress{{
					IP: loadBalancerIPAddress}}}},
	}
	return &istioService
}

// newUnstructuredService creates a service and returns it in Unstructured form
// uid - The UID of the service
// clusterIP - The cluster IP of the service
func newUnstructuredService(uid types.UID, clusterIP string, port int32) (unstructured.Unstructured, error) {
	service := k8score.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID: uid,
		},
		Spec: k8score.ServiceSpec{
			ClusterIP: clusterIP,
			Ports:     []k8score.ServicePort{{Port: port}}},
	}
	return convertToUnstructured(service)
}

// executeTemplate reads a template from a file and replaces values in the template from param maps
// template - The filename of a template
// params - a vararg of param maps
func executeTemplate(templateFile string, data interface{}) (string, error) {
	file := "../../" + templateFile
	if _, err := os.Stat(file); err != nil {
		file = "../" + templateFile
		if _, err := os.Stat(file); err != nil {
			file = templateFile
			if _, err := os.Stat(file); err != nil {
				return "", err
			}
		}
	}
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	t, err := template.New(templateFile).Parse(string(b))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = t.ExecuteTemplate(&buf, templateFile, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// updateUnstructuredFromYAMLTemplate updates an unstructured from a populated YAML template file.
// uns - The unstructured to update
// template - The template file
// params - The param maps to merge into the template
func updateUnstructuredFromYAMLTemplate(uns *unstructured.Unstructured, template string, data interface{}) error {
	str, err := executeTemplate(template, data)
	if err != nil {
		return err
	}
	bytes, err := yaml.YAMLToJSON([]byte(str))
	if err != nil {
		return err
	}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(bytes, nil, uns)
	if err != nil {
		return err
	}
	return nil
}

// createResourceFromTemplate builds a resource by merging the data with the template file and then
// creates the resource using the client.
func createResourceFromTemplate(cli client.Client, template string, data interface{}) error {
	uns := unstructured.Unstructured{}
	if err := updateUnstructuredFromYAMLTemplate(&uns, template, data); err != nil {
		return err
	}
	if err := cli.Create(context.Background(), &uns); err != nil {
		return err
	}
	return nil
}

// TestSuccessfullyCreateNewIngressForVerrazzanoWorkloadWithHTTPCookie tests the Reconcile method for the following use case.
// GIVEN a request to reconcile an ingress trait resource that applies to a Verrazzano workload type with HTTPCookie defined for session affinity
// WHEN the trait exists but the ingress does not
// THEN ensure that the workload is unwrapped and the trait is created.
func TestSuccessfullyCreateNewIngressForVerrazzanoWorkloadWithHTTPCookie(t *testing.T) {
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
				Paths: []vzapi.IngressPath{{Path: "test-path"}},
				Destination: vzapi.IngressDestination{
					Host: "test-service.test-space.svc.local",
					Port: 0,
					HTTPCookie: &vzapi.IngressDestinationHTTPCookie{
						Name: "test-cookie",
						Path: "/",
						TTL:  30},
				}}}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: "oam.verrazzano.io/v1alpha1",
				Kind:       "VerrazzanoCoherenceWorkload",
				Name:       "test-workload-name"}
			return nil
		})

	containedName := "test-contained-workload-name"
	containedResource := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": containedName,
		},
	}

	// Expect a call to get the Verrazzano Coherence workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetAPIVersion("oam.verrazzano.io/v1alpha1")
			workload.SetKind("VerrazzanoCoherenceWorkload")
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			unstructured.SetNestedMap(workload.Object, containedResource, "spec", "template")
			return nil
		})
	// Expect a call to get the contained Coherence resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: containedName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetUnstructuredContent(containedResource)
			workload.SetNamespace(name.Namespace)
			workload.SetAPIVersion("coherence.oracle.com/v1")
			workload.SetKind("Coherence")
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the containerized workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "coherences.coherence.oracle.com"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *v1alpha2.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []v1alpha2.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("DeploymentList", list.GetKind())
			return nil
		})
	// Expect a call to get the certificate related to the ingress trait
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: "test-space-myapp-cert"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Certificate"}, "test-space-myapp-cert"))
	// Expect a call to list the child Service resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("ServiceList", list.GetKind())
			return appendAsUnstructured(list, k8score.Service{
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
					}}},
				Spec: k8score.ServiceSpec{
					ClusterIP: "10.11.12.13",
					Ports:     []k8score.ServicePort{{Port: 42}}},
			})
		})
	// Expect a call to get the app config and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "myapp"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, app *v1alpha2.ApplicationConfiguration) error {
			app.TypeMeta = metav1.TypeMeta{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ApplicationConfiguration",
			}
			return nil
		})
	// Expect a call to create the certificate and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, certificate *certapiv1.Certificate, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the gateway resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-space-myapp-gw"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "Gateway"}, "test-space-myapp-gw"))
	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, gateway *istioclient.Gateway, opts ...client.CreateOption) error {
			return nil
		})
	// Expect a call to get the virtual service resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name-rule-0-vs"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "VirtualService"}, "test-trait-name-rule-0-vs"))
	// Expect a call to create the ingress resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, virtualservice *istioclient.VirtualService, opts ...client.CreateOption) error {
			assert.Len(virtualservice.Spec.Http, 1)
			assert.Len(virtualservice.Spec.Http[0].Route, 1)
			assert.Equal("test-service.test-space.svc.local", virtualservice.Spec.Http[0].Route[0].Destination.Host)
			return nil
		})
	// Expect a call to get the destination rule resource related to the ingress trait and return that it is not found.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-space", Name: "test-trait-name-rule-0-dr"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "DestinationRule"}, "test-trait-name-rule-0-dr"))

	// Expect a call to create the DestinationRule resource and return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, destinationrule *istioclient.DestinationRule, opts ...client.CreateOption) error {
			assert.Equal("test-service.test-space.svc.local", destinationrule.Spec.Host)
			return nil
		})
	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the ingress trait.
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.IngressTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			assert.Len(trait.Status.Resources, 4)
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

// TestReconcileKubeSystem tests to make sure we do not reconcile
// Any resource that belong to the kube-system namespace
func TestReconcileKubeSystem(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// create a request and reconcile it
	request := newRequest(vzconst.KubeSystem, "unit-test-verrazzano-helidon-workload")
	reconciler := newIngressTraitReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.Nil(err)
	assert.True(result.IsZero())
}
