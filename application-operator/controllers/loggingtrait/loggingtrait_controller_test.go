// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	k8sapps "k8s.io/api/apps/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

func TestReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var mgr *mocks.MockManager
	var cli *mocks.MockClient
	var scheme *runtime.Scheme
	var recorder event.Recorder
	var discoveryCli discovery.DiscoveryClient
	var reconciler LoggingTraitReconciler
	var err error

	mocker = gomock.NewController(t)
	mgr = mocks.NewMockManager(mocker)
	cli = mocks.NewMockClient(mocker)
	scheme = runtime.NewScheme()
	vzapi.AddToScheme(scheme)
	reconciler = LoggingTraitReconciler{Client: cli, Scheme: scheme, Record: recorder, DiscoveryClient: discoveryCli}
	mgr.EXPECT().GetConfig().Return(&rest.Config{})
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(log.NullLogger{})
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}

func TestCreateLoggingTrait(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mgr := mocks.NewMockManager(mocker)
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
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.LoggingTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       "LoggingTrait"}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				OwnerReferences: []k8smeta.OwnerReference{{
					APIVersion: oamcore.SchemeGroupVersion.Identifier(),
					Kind:       oamcore.ApplicationConfigurationKind,
					Name:       "test-appconfig-name"}}}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name"}
			return nil
		})

	// Expect a call to get the parent application configuration resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-appconfig-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appconfig *oamcore.ApplicationConfiguration) error {
			appconfig.TypeMeta = k8smeta.TypeMeta{
				APIVersion: oamcore.ApplicationConfigurationKindAPIVersion,
				Kind:       oamcore.ApplicationConfigurationKind}
			appconfig.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
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
		})

	// Expect a call to get the containerized workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
			}
			return nil
		})

	// Expect a call to list the child Deployment resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Deployment", list.GetKind())
			return appendAsUnstructured(list, testDeployment)
		})

	// Expect a call to patch the workload resource.
	mockStatus.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, trait *vzapi.LoggingTrait, patch client.Patch, option client.FieldOwner) error {
			return nil
		})

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler, err := newLoggingTraitReconciler(mock, mgr)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// newMetricsTraitReconciler creates a new reconciler for testing
// cli - The Kerberos client to inject into the reconciler
func newLoggingTraitReconciler(cli client.Client, mgr *mocks.MockManager) (LoggingTraitReconciler, error) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	scheme := runtime.NewScheme()
	vzapi.AddToScheme(scheme)
	discoveryCli, err := discovery.NewDiscoveryClientForConfig(&rest.Config{})
	if err != nil {
		return LoggingTraitReconciler{}, err
	}
	reconciler := LoggingTraitReconciler{
		Client:          cli,
		Log:             ctrl.Log,
		Scheme:          scheme,
		Record:          event.NewNopRecorder(),
		DiscoveryClient: *discoveryCli,
	}
	return reconciler, nil
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
