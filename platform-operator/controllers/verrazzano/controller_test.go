// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	vzappclusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	constants2 "github.com/verrazzano/verrazzano/pkg/mcconstants"
	clustersapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	helm2 "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/rbac"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakes "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// For unit testing
const testBomFilePath = "testdata/test_bom.json"

// Generate mocks for the Kerberos Client and StatusWriter interfaces for use in tests.
//go:generate mockgen -destination=../../mocks/controller_mock.go -package=mocks -copyright_file=../../hack/boilerplate.go.txt sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter

const installPrefix = "verrazzano-install-"
const uninstallPrefix = "verrazzano-uninstall-"

type nsMatcher struct {
	Name string
}

func (nm nsMatcher) Matches(i interface{}) bool {
	ns, ok := i.(*corev1.Namespace)
	if !ok {
		return false
	}
	return ns.Name == nm.Name
}

func (nsMatcher) String() string {
	return "namespace matcher"
}

// TestGetClusterRoleBindingName tests generating a ClusterRoleBinding name
// GIVEN a name and namespace
// WHEN the method is called
// THEN return the generated ClusterRoleBinding name
func TestGetClusterRoleBindingName(t *testing.T) {
	name := "role"
	namespace := "verrazzano"
	roleBindingName := buildClusterRoleBindingName(namespace, name)
	assert.Equalf(t, installPrefix+namespace+"-"+name, roleBindingName, "Expected ClusterRoleBinding name did not match")
}

// TestGetServiceAccountName tests generating a ServiceAccount name
// GIVEN a name
// WHEN the method is called
// THEN return the generated ServiceAccount name
func TestGetServiceAccountName(t *testing.T) {
	name := "sa"
	saName := buildServiceAccountName(name)
	assert.Equalf(t, installPrefix+name, saName, "Expected ServiceAccount name did not match")
}

// TestGetUninstallJobName tests generating a Job name
// GIVEN a name
// WHEN the method is called
// THEN return the generated Job name
func TestGetUninstallJobName(t *testing.T) {
	name := "test"
	jobName := buildUninstallJobName(name)
	assert.Equalf(t, uninstallPrefix+name, jobName, "Expected uninstall job name did not match")
}

// TestInstall tests the Reconcile method for the following use case
// GIVEN a request to reconcile a Verrazzano resource
// WHEN a Verrazzano resource has been applied
// THEN ensure all the objects are already created and
//      ensure a finalizer is added if it doesn't exist
func TestInstall(t *testing.T) {
	tests := []struct {
		namespace string
		name      string
		finalizer string
	}{
		{"verrazzano", "test", ""},
		{"verrazzano", "test", finalizerName},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			unitTesting = true
			namespace := test.namespace
			name := test.name
			labels := map[string]string{"label1": "test"}
			var verrazzanoToUse vzapi.Verrazzano
			asserts := assert.New(t)
			mocker := gomock.NewController(t)
			mock := mocks.NewMockClient(mocker)
			mockStatus := mocks.NewMockStatusWriter(mocker)
			asserts.NotNil(mockStatus)

			config.TestHelmConfigDir = "../../helm_config"
			defer func() { config.TestHelmConfigDir = "" }()

			config.TestProfilesDir = "../../manifests/profiles"
			defer func() { config.TestProfilesDir = "" }()

			verrazzanoToUse.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzanoToUse.ObjectMeta = metav1.ObjectMeta{
				Namespace:  namespace,
				Name:       name,
				Labels:     labels,
				Finalizers: []string{test.finalizer}}
			verrazzanoToUse.Spec.Components.DNS = &vzapi.DNSComponent{External: &vzapi.External{Suffix: "mydomain.com"}}

			verrazzanoToUse.Status.State = vzapi.VzStateReady
			verrazzanoToUse.Status.Components = makeVerrazzanoComponentStatusMap()

			// Sample bom file for version validation functions
			config.SetDefaultBomFilePath(testBomFilePath)
			defer config.SetDefaultBomFilePath("")

			registry.OverrideGetComponentsFn(func() []spi.Component {
				return []spi.Component{
					fakeComponent{},
				}
			})
			defer registry.ResetGetComponentsFn()

			// Expect a call to get the Verrazzano resource.
			expectGetVerrazzanoExists(mock, verrazzanoToUse, namespace, name, labels)

			// Expect a call to get the service account
			expectGetServiceAccountExists(mock, name, labels)

			// Expect a call to get the ClusterRoleBinding
			expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

			// Expect a call to get the Verrazzano system namespace (return exists)
			expectGetVerrazzanoSystemNamespaceExists(mock, asserts)

			// Expect a call to get the status writer and return a mock.
			mock.EXPECT().Status().Return(mockStatus).AnyTimes()

			// Expect a call to update the finalizers - return success
			if test.finalizer != finalizerName {
				mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			}

			// Expect a call to update the status of the Verrazzano resource
			mockStatus.EXPECT().
				Update(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
					asserts.Len(verrazzano.Status.Conditions, 1)
					return nil
				}).Times(1)

			// Expect local registration calls
			expectSyncLocalRegistration(t, mock, name)

			// Create and make the request
			request := newRequest(namespace, name)
			reconciler := newVerrazzanoReconciler(mock)
			reconcileCounterMetric, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.ReconcileCounter)
			assert.NoError(t, err)
			reconcileCounterBefore := testutil.ToFloat64(reconcileCounterMetric.Get())
			result, err := reconciler.Reconcile(context.TODO(), request)
			reconcileCounterAfter := testutil.ToFloat64(reconcileCounterMetric.Get())
			asserts.Equal(reconcileCounterBefore, reconcileCounterAfter-1)

			assert.NoError(t, err)

			// Validate the results
			mocker.Finish()
			asserts.NoError(err)
			asserts.Equal(false, result.Requeue)
			asserts.Equal(time.Duration(0), result.RequeueAfter)
		})
	}
}

// TestInstallInitComponents tests the reconcile method for the following use case
// GIVEN a request to reconcile a Verrazzano resource when Status.Components is empty
// THEN ensure that the Status.components is populated
func TestInstallInitComponents(t *testing.T) {
	unitTesting = true
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	var verrazzanoToUse vzapi.Verrazzano
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	config.TestHelmConfigDir = "../../helm_config"

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	verrazzanoToUse.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	verrazzanoToUse.ObjectMeta = metav1.ObjectMeta{
		Namespace:  namespace,
		Name:       name,
		Labels:     labels,
		Finalizers: []string{finalizerName}}
	verrazzanoToUse.Status.State = vzapi.VzStateReady

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	// Expect a call to get the Verrazzano resource.
	expectGetVerrazzanoExists(mock, verrazzanoToUse, namespace, name, labels)

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource to update components
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.NotZero(result.RequeueAfter)

	// Validate the results
	mocker.Finish()
}

// TestCreateLocalRegistrationSecret tests the syncLocalRegistrationSecret method for the following use case
// GIVEN a request to sync the local cluster MC registration secret
// WHEN a the secret does not exist
// THEN ensure the secret is created successfully
func TestCreateLocalRegistrationSecret(t *testing.T) {
	unitTesting = true

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoSystemNamespace, Resource: "Secret"}, constants.MCAgentSecret))

	// Expect a call to get the local registration secret in the verrazzano-system namespace - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCLocalRegistrationSecret}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoSystemNamespace, Resource: "Secret"}, constants.MCLocalRegistrationSecret))

	// Expect a call to create the registration secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			secret.Data = map[string][]byte{
				constants2.ManagedClusterNameKey: []byte("cluster1"),
			}
			return nil
		})

	// Create and make the request
	reconciler := newVerrazzanoReconciler(mock)
	err := reconciler.syncLocalRegistrationSecret()

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
}

// TestCreateLocalRegistrationSecretUnexpectedError tests the syncLocalRegistrationSecret method for the following use case
// GIVEN a request to sync the local cluster MC registration secret
// WHEN a call to get the secret does returns an error other than IsNotFound
// THEN an error is returned and no attempt is made to create the secret
func TestCreateLocalRegistrationSecretUnexpectedError(t *testing.T) {
	unitTesting = true

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil())).
		Return(fmt.Errorf("Unexpected error getting secret"))

	// Expect a call to get the local registration secret in the verrazzano-system namespace - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCLocalRegistrationSecret}, gomock.Not(gomock.Nil())).
		Times(0)

	// Expect a call to create the registration secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Times(0)

	// Create and make the request
	reconciler := newVerrazzanoReconciler(mock)
	err := reconciler.syncLocalRegistrationSecret()

	// Validate the results
	mocker.Finish()
	asserts.Error(err)
}

func makeVerrazzanoComponentStatusMap() vzapi.ComponentStatusMap {
	statusMap := make(vzapi.ComponentStatusMap)
	for _, comp := range registry.GetComponents() {
		if comp.IsOperatorInstallSupported() {
			statusMap[comp.Name()] = &vzapi.ComponentStatusDetails{
				Name: comp.Name(),
				Conditions: []vzapi.Condition{
					{
						Type:   vzapi.CondInstallComplete,
						Status: corev1.ConditionTrue,
					},
				},
				State: vzapi.CompStateReady,
			}
		}
	}
	return statusMap
}

// TestCreateVerrazzanoWithOCIDNS tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource with OCI DNS configured
// WHEN a Verrazzano resource has been created
// THEN ensure all the objects are created
func TestCreateVerrazzanoWithOCIDNS(t *testing.T) {
	unitTesting = true
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test1"}

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	var vzToUse vzapi.Verrazzano
	vzToUse.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	vzToUse.ObjectMeta = metav1.ObjectMeta{
		Namespace:  namespace,
		Name:       name,
		Labels:     labels,
		Finalizers: []string{finalizerName}}
	vzToUse.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: &vzapi.OCI{
			OCIConfigSecret:        "test-oci-config-secret",
			DNSZoneCompartmentOCID: "test-dns-zone-ocid",
			DNSZoneOCID:            "test-dns-zone-ocid",
			DNSZoneName:            "test-dns-zone-name",
		},
	}
	vzToUse.Status.Components = makeVerrazzanoComponentStatusMap()
	vzToUse.Status.State = vzapi.VzStateReady

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Expect a call to get the Verrazzano resource.
	expectGetVerrazzanoExists(mock, vzToUse, namespace, name, labels)

	// Expect a call to get the DNS config secret and return it
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoInstallNamespace, Name: "test-oci-config-secret"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			data := make(map[string][]byte)
			data["passphrase"] = []byte("passphraseValue")
			secret.ObjectMeta = metav1.ObjectMeta{
				Name:      "private-key",
				Namespace: "default",
				Labels:    nil,
			}
			data = make(map[string][]byte)
			data[vzapi.OciConfigSecretFile] = []byte("auth:\n  region: us-phoenix-1\n  tenancy: ocid1.tenancy.ocid\n  user: ocid1.user.ocid\n  key: |\n    -----BEGIN RSA PRIVATE KEY-----\n    someencodeddata\n    -----END RSA PRIVATE KEY-----\n  fingerprint: theFingerprint\n  passphrase: passphraseValue")
			secret.Data = data
			secret.Type = corev1.SecretTypeOpaque
			return nil
		})

	// Expect a call to get the Verrazzano system namespace (return exists)
	expectGetVerrazzanoSystemNamespaceExists(mock, asserts)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 1)
			return nil
		}).Times(1)

	// Expect local registration calls
	expectSyncLocalRegistration(t, mock, name)

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUninstallComplete tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource
// WHEN a Verrazzano resource has been deleted
// THEN ensure all the objects are deleted
func TestUninstallComplete(t *testing.T) {
	unitTesting = true
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	var verrazzanoToUse vzapi.Verrazzano

	deleteTime := metav1.Time{
		Time: time.Now(),
	}

	setFakeComponentsDisabled()
	defer registry.ResetGetComponentsFn()

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Verrazzano resource.  Return resource with deleted timestamp.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:         name.Namespace,
				Name:              name.Name,
				DeletionTimestamp: &deleteTime,
				Finalizers:        []string{finalizerName}}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State:      vzapi.VzStateReady,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondUninstallComplete,
					},
				},
			}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	expectMCCleanup(mock)

	// Expect node-exporter cleanup
	expectNodeExporterCleanup(mock)

	// Expect calls to delete the shared namespaces
	expectSharedNamespaceDeletes(mock)

	// Expect a call to update the finalizers - return success
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 2)
			return nil
		})

	expectIstioCertRemoval(mock, 1)

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUninstallStarted tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource
// WHEN a Verrazzano resource has been deleted
// THEN ensure an unisntall job is started
func TestUninstallStarted(t *testing.T) {
	unitTesting = true
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	var verrazzanoToUse vzapi.Verrazzano

	verrazzanoToUse.Status.State = vzapi.VzStateReady

	deleteTime := metav1.Time{
		Time: time.Now(),
	}

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	setFakeComponentsDisabled()
	defer registry.ResetGetComponentsFn()

	// Expect a call to get the Verrazzano resource.  Return resource with deleted timestamp.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:         name.Namespace,
				Name:              name.Name,
				Labels:            labels,
				DeletionTimestamp: &deleteTime,
				Finalizers:        []string{finalizerName}}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State: vzapi.VzStateReady,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondUninstallStarted,
					},
				},
			}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	expectMCCleanup(mock)

	// Expect node-exporter cleanup
	expectNodeExporterCleanup(mock)

	// Expect calls to delete the shared namespaces
	expectSharedNamespaceDeletes(mock)

	// Expect a call to update the job - return success
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to get the status writer and return a mock.
	mockStatus.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	expectIstioCertRemoval(mock, 1)

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.False(result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

func setFakeComponentsDisabled() {
	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{
				HelmComponent: helm2.HelmComponent{
					ReleaseName:    "fake",
					ChartNamespace: "fake",
				},
				isInstalledFunc: func(ctx spi.ComponentContext) (bool, error) {
					return false, nil
				},
			},
		}
	})
}

// TestUninstallSucceeded tests the Reconcile method for the following use case
// GIVEN an uninstall has succeeded
// WHEN a Verrazzano resource has been deleted
// THEN ensure all the objects are deleted
func TestUninstallSucceeded(t *testing.T) {
	unitTesting = true
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	var verrazzanoToUse vzapi.Verrazzano

	deleteTime := metav1.Time{
		Time: time.Now(),
	}

	setFakeComponentsDisabled()
	defer registry.ResetGetComponentsFn()

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Verrazzano resource.  Return resource with deleted timestamp.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:         name.Namespace,
				Name:              name.Name,
				DeletionTimestamp: &deleteTime,
				Finalizers:        []string{finalizerName}}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State: vzapi.VzStateReady}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	expectMCCleanup(mock)

	// Expect node-exporter cleanup
	expectNodeExporterCleanup(mock)

	// Expect calls to delete the shared namespaces
	expectSharedNamespaceDeletes(mock)

	// Expect a call to update the finalizers - return success
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			return nil
		}).AnyTimes()

	expectIstioCertRemoval(mock, 1)

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestVerrazzanoNotFound tests the Reconcile method for the following use case
// GIVEN an reqyest for a Verrazzano custom resource
// WHEN it does not exist
// THEN ensure the error not found is handled
func TestVerrazzanoNotFound(t *testing.T) {
	unitTesting = true
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Verrazzano resource - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Verrazzano"}, name))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestVerrazzanoGetError tests the Reconcile method for the following use case
// GIVEN an reqyest for a Verrazzano custom resource
// WHEN there is a failure getting it
// THEN ensure the error is handled
func TestVerrazzanoGetError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Verrazzano resource - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		Return(errors.NewBadRequest("failed to get Verrazzano custom resource"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.NotZero(result.RequeueAfter)
}

// TestVZSystemNamespaceGetError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource
// WHEN a there is an error getting the Verrazzano system namespace
// THEN return error
func TestVZSystemNamespaceGetError(t *testing.T) {
	unitTesting = true
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	var verrazzanoToUse vzapi.Verrazzano
	asserts.NotNil(mockStatus)

	verrazzanoToUse.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	verrazzanoToUse.ObjectMeta = metav1.ObjectMeta{
		Namespace:  namespace,
		Name:       name,
		Labels:     labels,
		Finalizers: []string{finalizerName}}
	verrazzanoToUse.Status = vzapi.VerrazzanoStatus{
		State: vzapi.VzStateReady}
	verrazzanoToUse.Status.Components = makeVerrazzanoComponentStatusMap()

	// Expect a call to get the Verrazzano resource.
	expectGetVerrazzanoExists(mock, verrazzanoToUse, namespace, name, labels)

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	errMsg := "get vz system namespace error"
	// Expect a call to get the Verrazzano system namespace - return a failure error
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: constants.VerrazzanoSystemNamespace}, gomock.Not(gomock.Nil())).
		Return(errors.NewBadRequest(errMsg))

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.NotZero(result.RequeueAfter)
}

// TestVZSystemNamespaceCreateError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource
// WHEN a there is an error creating the Verrazzano system namespace
// THEN return error
func TestVZSystemNamespaceCreateError(t *testing.T) {
	unitTesting = true
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	var verrazzanoToUse vzapi.Verrazzano
	asserts.NotNil(mockStatus)

	verrazzanoToUse.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	verrazzanoToUse.ObjectMeta = metav1.ObjectMeta{
		Namespace:  namespace,
		Name:       name,
		Labels:     labels,
		Finalizers: []string{finalizerName}}
	verrazzanoToUse.Status = vzapi.VerrazzanoStatus{
		State: vzapi.VzStateReady}
	verrazzanoToUse.Status.Components = makeVerrazzanoComponentStatusMap()

	// Expect a call to get the Verrazzano resource.
	expectGetVerrazzanoExists(mock, verrazzanoToUse, namespace, name, labels)

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	errMsg := "create vz system namespace error"
	// Expect a call to get the Verrazzano system namespace - return an IsNotFound
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: constants.VerrazzanoSystemNamespace}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.ParseGroupResource("namespaces"), constants.VerrazzanoSystemNamespace))

	// Expect a call to create the Verrazzano system namespace - return a failure error
	mock.EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Namespace{})).
		Return(errors.NewBadRequest(errMsg))

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.NotZero(result.RequeueAfter)
}

// TestGetOCIConfigSecretError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource
// WHEN a there is an error getting the OCI CR secret
// THEN return error
func TestGetOCIConfigSecretError(t *testing.T) {
	unitTesting = true
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	var verrazzanoToUse vzapi.Verrazzano
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	verrazzanoToUse.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	verrazzanoToUse.ObjectMeta = metav1.ObjectMeta{
		Namespace:  namespace,
		Name:       name,
		Labels:     labels,
		Finalizers: []string{finalizerName}}
	verrazzanoToUse.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: &vzapi.OCI{
			OCIConfigSecret:        "test-oci-config-secret",
			DNSZoneCompartmentOCID: "test-dns-zone-ocid",
			DNSZoneOCID:            "test-dns-zone-ocid",
			DNSZoneName:            "test-dns-zone-name",
		},
	}
	verrazzanoToUse.Status = vzapi.VerrazzanoStatus{
		State: vzapi.VzStateReady}
	verrazzanoToUse.Status.Components = makeVerrazzanoComponentStatusMap()

	// Expect a call to get the Verrazzano resource.
	expectGetVerrazzanoExists(mock, verrazzanoToUse, namespace, name, labels)

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the DNS config secret but return a not found error
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoInstallNamespace, Name: "test-oci-config-secret"}, gomock.Not(gomock.Nil())).
		Return(errors.NewBadRequest("failed to get Secret"))

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.NotZero(result.RequeueAfter)
}

// Test_appendConditionIfNecessary tests whether conditions are appended or updated correctly
// GIVEN a list of conditions
// WHEN the new condition already exists,
// THEN it should be updated and duplicates removed
// OTHERWISE it should be appended to the list of conditions
func Test_appendConditionIfNecessary(t *testing.T) {
	asserts := assert.New(t)
	tests := []struct {
		name                string
		conditions          []vzapi.Condition
		expectNumConditions int
	}{
		{
			name:                "no existing conditions",
			conditions:          []vzapi.Condition{},
			expectNumConditions: 1,
		},
		{
			name: "one InstallStarted condition",
			conditions: []vzapi.Condition{
				{Type: vzapi.CondInstallStarted, Status: corev1.ConditionFalse, LastTransitionTime: "some time"},
			},
			expectNumConditions: 1,
		},
		{
			name: "multiple InstallStarted conditions",
			conditions: []vzapi.Condition{
				{Type: vzapi.CondInstallStarted, Status: corev1.ConditionFalse, LastTransitionTime: "some time"},
				{Type: vzapi.CondInstallStarted, Status: corev1.ConditionFalse, LastTransitionTime: "some other time"},
			},
			expectNumConditions: 1,
		},
		{
			name: "one some other condition",
			conditions: []vzapi.Condition{
				{Type: vzapi.CondUpgradeFailed, Status: corev1.ConditionFalse, LastTransitionTime: "some time"},
			},
			expectNumConditions: 2,
		},
		{
			name: "multiple other conditions",
			conditions: []vzapi.Condition{
				{Type: vzapi.CondUpgradeStarted, Status: corev1.ConditionTrue, LastTransitionTime: "some time 1"},
				{Type: vzapi.CondUpgradeFailed, Status: corev1.ConditionFalse, LastTransitionTime: "some time 2"},
			},
			expectNumConditions: 3,
		},
		{
			name: "multiple conditions with InstallStarted duplicates",
			conditions: []vzapi.Condition{
				{Type: vzapi.CondInstallStarted, Status: corev1.ConditionFalse, LastTransitionTime: "some time before"},
				{Type: vzapi.CondUpgradeFailed, Status: corev1.ConditionFalse, LastTransitionTime: "some time2"},
				{Type: vzapi.CondInstallStarted, Status: corev1.ConditionFalse, LastTransitionTime: "some other time"},
				{Type: vzapi.CondInstallStarted, Status: corev1.ConditionTrue, LastTransitionTime: "yet another time"},
				{Type: vzapi.CondPreInstall, Status: corev1.ConditionFalse, LastTransitionTime: "some time preinstall"},
			},
			expectNumConditions: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unitTesting = true
			newCondition := vzapi.Condition{Status: corev1.ConditionTrue, Type: vzapi.CondInstallStarted, LastTransitionTime: "updatedtime"}
			updatedConditions := appendConditionIfNecessary(vzlog.DefaultLogger(), "my-vz", tt.conditions, newCondition)
			asserts.Equal(tt.expectNumConditions, len(updatedConditions))
			// the new install started condition should be the last one in the list, and its value
			// should be what we set it to
			cond := updatedConditions[tt.expectNumConditions-1]
			if cond.Type == vzapi.CondInstallStarted {
				asserts.Equal("updatedtime", cond.LastTransitionTime)
				asserts.Equal(corev1.ConditionTrue, cond.Status)
			}
		})
	}
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	// _ = clientgoscheme.AddToScheme(scheme)
	// _ = core.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	return scheme
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

// newVerrazzanoReconciler creates a new reconciler for testing
// c - The Kerberos client to inject into the reconciler
func newVerrazzanoReconciler(c client.Client) Reconciler {
	scheme := newScheme()
	reconciler := Reconciler{
		Client:            c,
		Scheme:            scheme,
		WatchedComponents: map[string]bool{},
		WatchMutex:        &sync.RWMutex{},
	}
	return reconciler
}

// Expect syncLocalRegistration related calls, happy-path secret exists
func expectSyncLocalRegistration(t *testing.T, mock *mocks.MockClient, name string) {
	// Expect a call to get the Agent secret in the verrazzano-system namespace - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil())).
		Return(nil)
}

// expectGetVerrazzanoSystemNamespaceExists expects a call to get the Verrazzano system namespace and returns
// that it exists
func expectGetVerrazzanoSystemNamespaceExists(mock *mocks.MockClient, asserts *assert.Assertions) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: constants.VerrazzanoSystemNamespace}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			ns.Name = constants.VerrazzanoSystemNamespace
			ns.Labels = systemNamespaceLabels
			return nil
		})
}

// expectClusterRoleBindingExists expects a call to get the cluster role binding for the Verrazzano with the given
// namespace and name, and returns that it exists
func expectClusterRoleBindingExists(mock *mocks.MockClient, verrazzanoToUse vzapi.Verrazzano, namespace string, name string) {
	// Expect a call to get the ClusterRoleBinding - return that it exists
	clusterRoleBindingName := buildClusterRoleBindingName(namespace, name)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: clusterRoleBindingName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, nsName types.NamespacedName, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
			crb := rbac.NewClusterRoleBinding(&verrazzanoToUse, nsName.Name, getInstallNamespace(), buildServiceAccountName(nsName.Name))
			clusterRoleBinding.ObjectMeta = crb.ObjectMeta
			clusterRoleBinding.RoleRef = crb.RoleRef
			clusterRoleBinding.Subjects = crb.Subjects
			return nil
		}).AnyTimes()
}

// expectGetServiceAccountExists expects a call to get the service account for the Verrazzano with the given
// namespace and name, and returns that it exists
func expectGetServiceAccountExists(mock *mocks.MockClient, name string, labels map[string]string) {
	// Expect a call to get the ServiceAccount - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: getInstallNamespace(), Name: buildServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, serviceAccount *corev1.ServiceAccount) error {
			newSA := rbac.NewServiceAccount(name.Namespace, name.Name, []string{}, labels)
			serviceAccount.ObjectMeta = newSA.ObjectMeta
			return nil
		}).AnyTimes()
}

// expectGetVerrazzanoExists expects a call to get a Verrazzano with the given namespace and name, and returns
// one that has the same content as the verrazzanoToUse argument
func expectGetVerrazzanoExists(mock *mocks.MockClient, verrazzanoToUse vzapi.Verrazzano, namespace string, name string, labels map[string]string) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = verrazzanoToUse.TypeMeta
			verrazzano.ObjectMeta = verrazzanoToUse.ObjectMeta
			verrazzano.Spec.Components.DNS = verrazzanoToUse.Spec.Components.DNS
			verrazzano.Status = verrazzanoToUse.Status
			return nil
		}).AnyTimes()
}

func expectSharedNamespaceDeletes(mock *mocks.MockClient) {
	const fakeNS = "fake"
	for _, ns := range sharedNamespaces {
		mock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Name: ns}, gomock.Not(gomock.Nil())).
			Return(nil)
		mock.EXPECT().Delete(gomock.Any(), nsMatcher{Name: ns}, gomock.Any()).Return(nil)
		mock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Name: ns}, gomock.Not(gomock.Nil())).
			Return(errors.NewNotFound(schema.ParseGroupResource("Namespace"), ns))
	}
	// Expect delete for component namesapces
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: fakeNS}, gomock.Not(gomock.Nil())).
		Return(nil)
	mock.EXPECT().Delete(gomock.Any(), nsMatcher{Name: fakeNS}, gomock.Any()).Return(nil)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: fakeNS}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.ParseGroupResource("Namespace"), fakeNS))

}

// expectIstioCertRemoval creates the expects for the Istio cert removal
func expectIstioCertRemoval(mock *mocks.MockClient, numList int) {
	mock.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil).Times(numList)
}

// expectNodeExporterCleanup creates the expects for the node-exporter cleanup
func expectNodeExporterCleanup(mock *mocks.MockClient) {
	mock.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(2)
}

func expectMCCleanup(mock *mocks.MockClient) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoSystemNamespace, Resource: "Secret"}, constants.MCAgentSecret))

	mock.EXPECT().
		List(gomock.Any(), &clustersapi.VerrazzanoManagedClusterList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmcList *clustersapi.VerrazzanoManagedClusterList, options ...*client.ListOptions) error {
			vmcList.Items = []clustersapi.VerrazzanoManagedCluster{}
			return nil
		})

	mock.EXPECT().
		List(gomock.Any(), &vzappclusters.VerrazzanoProjectList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, projects *vzappclusters.VerrazzanoProjectList, options ...*client.ListOptions) error {
			projects.Items = []vzappclusters.VerrazzanoProject{}
			return nil
		}).AnyTimes()

	mock.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
}

// TestMergeMapsNilSourceMap tests mergeMaps function
// GIVEN an empty source map and a non-empty map to merge
// WHEN the mergeMaps function is called
// THEN true is returned and a new map with the merged values is created
func TestMergeMapsNilSourceMap(t *testing.T) {
	var mymap map[string]string
	systemNamespaceLabels := map[string]string{
		"istio-injection":         "enabled",
		"verrazzano.io/namespace": constants.VerrazzanoSystemNamespace,
	}
	newMap, updated := mergeMaps(mymap, systemNamespaceLabels)
	t.Logf("Merged map: %v", newMap)
	assert.True(t, updated)
	assert.Equal(t, systemNamespaceLabels, newMap)
}

// TestMergeNestedEmptyMap tests mergeMaps function
// GIVEN an empty source map nested in a struct and a non-empty map to merge
// WHEN the mergeMaps function is called
// THEN true is returned and a new map with the merged values is created
func TestMergeNestedEmptyMap(t *testing.T) {
	type mytype struct {
		MyMap map[string]string
	}
	systemNamespaceLabels := map[string]string{
		"istio-injection":         "enabled",
		"verrazzano.io/namespace": constants.VerrazzanoSystemNamespace,
	}
	var updated bool
	myInstance := mytype{}
	myInstance.MyMap, updated = mergeMaps(myInstance.MyMap, systemNamespaceLabels)
	t.Logf("Merged map: %v", myInstance.MyMap)
	assert.True(t, updated)
	assert.Equal(t, systemNamespaceLabels, myInstance.MyMap)
}

// TestMergeNestedMapEntriesExist tests mergeMaps function
// GIVEN an two maps are merged with the same values
// WHEN the mergeMaps function is called
// THEN false is returned the map is unchanged
func TestMergeNestedMapEntriesExist(t *testing.T) {
	type mytype struct {
		MyMap map[string]string
	}
	systemNamespaceLabels := map[string]string{
		"istio-injection":         "enabled",
		"verrazzano.io/namespace": constants.VerrazzanoSystemNamespace,
	}
	var updated bool
	myInstance := mytype{}
	myInstance.MyMap = map[string]string{
		"istio-injection":         "enabled",
		"verrazzano.io/namespace": constants.VerrazzanoSystemNamespace,
	}

	myInstance.MyMap, updated = mergeMaps(myInstance.MyMap, systemNamespaceLabels)
	assert.False(t, updated)
	assert.Equal(t, systemNamespaceLabels, myInstance.MyMap)
}

// TestPartialMergeNestedMap tests mergeMaps function
// GIVEN source map contains a subset of the map to merge
// WHEN the mergeMaps function is called
// THEN true is returned the new map has all expected values
func TestPartialMergeNestedMap(t *testing.T) {
	type mytype struct {
		MyMap map[string]string
	}
	systemNamespaceLabels := map[string]string{
		"istio-injection":         "enabled",
		"verrazzano.io/namespace": constants.VerrazzanoSystemNamespace,
	}
	var updated bool
	myInstance := mytype{}
	myInstance.MyMap = map[string]string{
		"istio-injection": "enabled",
	}

	myInstance.MyMap, updated = mergeMaps(myInstance.MyMap, systemNamespaceLabels)
	assert.True(t, updated)
	assert.Equal(t, systemNamespaceLabels, myInstance.MyMap)
}

// TestChangedValueMergeNestedMap tests mergeMaps function
// GIVEN source map contains the same set of keys, but with a different value
// WHEN the mergeMaps function is called
// THEN true is returned the new map has all expected values
func TestChangedValueMergeNestedMap(t *testing.T) {
	type mytype struct {
		MyMap map[string]string
	}
	systemNamespaceLabels := map[string]string{
		"istio-injection":         "disabled",
		"verrazzano.io/namespace": constants.VerrazzanoSystemNamespace,
	}
	var updated bool
	myInstance := mytype{}
	myInstance.MyMap = map[string]string{
		"istio-injection":         "enabled",
		"verrazzano.io/namespace": constants.VerrazzanoSystemNamespace,
	}

	myInstance.MyMap, updated = mergeMaps(myInstance.MyMap, systemNamespaceLabels)
	assert.True(t, updated)
	assert.Equal(t, systemNamespaceLabels, myInstance.MyMap)
}

// TestNonIntersectingMergeNestedMap tests mergeMaps function
// GIVEN source map and map to merge contain non-intersecting values
// WHEN the mergeMaps function is called
// THEN true is returned the new map is a union of all values
func TestNonIntersectingMergeNestedMap(t *testing.T) {
	type mytype struct {
		MyMap map[string]string
	}
	systemNamespaceLabels := map[string]string{
		"istio-injection":         "enabled",
		"verrazzano.io/namespace": constants.VerrazzanoSystemNamespace,
	}
	var updated bool
	myInstance := mytype{}
	myInstance.MyMap = map[string]string{
		"mylabel": "someValue",
	}

	expectedMap := map[string]string{
		"istio-injection":         "enabled",
		"verrazzano.io/namespace": constants.VerrazzanoSystemNamespace,
		"mylabel":                 "someValue",
	}

	myInstance.MyMap, updated = mergeMaps(myInstance.MyMap, systemNamespaceLabels)
	assert.True(t, updated)
	assert.Len(t, myInstance.MyMap, 3)
	assert.Equal(t, expectedMap, myInstance.MyMap)
}

// TestReconcileErrorCounter tests Reconcile function
// GIVEN a faulty request
// WHEN the reconcile function is called
// THEN an error occurs and the error counter metric is incremented
func TestReconcileErrorCounter(t *testing.T) {
	asserts := assert.New(t)
	clientBuilder := fakes.NewClientBuilder()
	fakeClient := clientBuilder.Build()
	errorRequest := newRequest("bad namespace", "test")
	reconciler := newVerrazzanoReconciler(fakeClient)
	reconcileErrorCounterMetric, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.ReconcileError)
	assert.NoError(t, err)
	errorCounterBefore := testutil.ToFloat64(reconcileErrorCounterMetric.Get())
	reconciler.Reconcile(context.TODO(), errorRequest)
	errorCounterAfter := testutil.ToFloat64(reconcileErrorCounterMetric.Get())
	assert.NoError(t, err)
	asserts.Equal(errorCounterBefore, errorCounterAfter-1)
}
