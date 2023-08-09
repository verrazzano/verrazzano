// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	clustersapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	constants3 "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	cmissuer "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/issuer"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	vzContext "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/context"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	time2 "helm.sh/helm/v3/pkg/time"
	batchv1 "k8s.io/api/batch/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/dynamic"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	vzappclusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	constants2 "github.com/verrazzano/verrazzano/pkg/mcconstants"
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
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakes "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// For unit testing
const testBomFilePath = "../testdata/test_bom.json"

// Generate mocks for the Kerberos Client and StatusWriter interfaces for use in tests.
//
//go:generate mockgen -destination=../../../mocks/controller_mock.go -package=mocks -copyright_file=../../../hack/boilerplate.go.txt sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter
//go:generate mockgen -destination=../../../mocks/runtime_controller_mock.go -package=mocks -copyright_file=../../../hack/boilerplate.go.txt sigs.k8s.io/controller-runtime/pkg/controller Controller
const installPrefix = "verrazzano-install-"
const uninstallPrefix = "verrazzano-uninstall-"
const relativeProfilesDir = "../../../manifests/profiles"
const relativeHelmConfig = "../../../helm_config"
const unExpectedError = "unexpected error"

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
//
//	ensure a finalizer is added if it doesn't exist
func TestInstall(t *testing.T) {
	metricsexporter.Init()
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

			config.TestHelmConfigDir = relativeHelmConfig
			defer func() { config.TestHelmConfigDir = "" }()

			config.TestProfilesDir = relativeProfilesDir
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

			// Expect a call to get the ingressList
			expectGetIngressListExists(mock)

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

	config.TestHelmConfigDir = relativeHelmConfig

	config.TestProfilesDir = relativeProfilesDir
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
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
		return helm.CreateActionConfig(true, name, release.StatusDeployed, vzlog.DefaultLogger(), func(name string, releaseStatus release.Status) *release.Release {
			now := time2.Now()
			return &release.Release{
				Name:      name,
				Namespace: namespace,
				Info: &release.Info{
					FirstDeployed: now,
					LastDeployed:  now,
					Status:        releaseStatus,
					Description:   "Named Release Stub",
				},
				Version: 1,
			}
		})
	})

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
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil()), gomock.Any()).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoSystemNamespace, Resource: "Secret"}, constants.MCAgentSecret))

	// Expect a call to get the local registration secret in the verrazzano-system namespace - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCLocalRegistrationSecret}, gomock.Not(gomock.Nil()), gomock.Any()).
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
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil()), gomock.Any()).
		Return(fmt.Errorf("Unexpected error getting secret"))

	// Expect a call to get the local registration secret in the verrazzano-system namespace - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCLocalRegistrationSecret}, gomock.Not(gomock.Nil()), gomock.Any()).
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
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
		return helm.CreateActionConfig(true, name, release.StatusDeployed, vzlog.DefaultLogger(), func(name string, releaseStatus release.Status) *release.Release {
			now := time2.Now()
			return &release.Release{
				Name:      name,
				Namespace: namespace,
				Info: &release.Info{
					FirstDeployed: now,
					LastDeployed:  now,
					Status:        releaseStatus,
					Description:   "Named Release Stub",
				},
				Version: 1,
			}
		})
	})

	config.TestProfilesDir = relativeProfilesDir
	defer func() { config.TestProfilesDir = "" }()

	// Expect a call to get the Verrazzano resource.
	expectGetVerrazzanoExists(mock, vzToUse, namespace, name, labels)

	// Expect a call to get the DNS config secret and return it
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoInstallNamespace, Name: "test-oci-config-secret"}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
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

	// Expect a call to get the ingressList
	expectGetIngressListExists(mock)

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

	k8sutil.GetCoreV1Func = common.MockGetCoreV1()
	k8sutil.GetDynamicClientFunc = common.MockDynamicClient()
	defer func() {
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
	}()
	testFunc := func(client typedcorev1.CoreV1Interface, dynClient dynamic.Interface) (bool, error) { return false, nil }
	rancher.SetCheckClusterProvisionedFunc(testFunc)
	defer rancher.SetDefaultCheckClusterProvisionedFunc()

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Verrazzano resource.  Return resource with deleted timestamp.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano, opts ...client.GetOption) error {
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

	defer func() { cmCleanupFunc = cmissuer.UninstallCleanup }()
	cmCleanupFunc = func(log vzlog.VerrazzanoLogger, cli client.Client, namespace string) error { return nil }

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

	config.TestProfilesDir = relativeProfilesDir
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
// THEN ensure an uninstall job is started
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

	k8sutil.GetCoreV1Func = common.MockGetCoreV1()
	k8sutil.GetDynamicClientFunc = common.MockDynamicClient()
	defer func() {
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
	}()
	testFunc := func(client typedcorev1.CoreV1Interface, dynClient dynamic.Interface) (bool, error) { return false, nil }
	rancher.SetCheckClusterProvisionedFunc(testFunc)
	defer rancher.SetDefaultCheckClusterProvisionedFunc()

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	setFakeComponentsDisabled()
	defer registry.ResetGetComponentsFn()

	defer func() { cmCleanupFunc = cmissuer.UninstallCleanup }()
	cmCleanupFunc = func(log vzlog.VerrazzanoLogger, cli client.Client, namespace string) error { return nil }

	// Expect a call to get the Verrazzano resource.  Return resource with deleted timestamp.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano, opts ...client.GetOption) error {
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

	config.TestProfilesDir = relativeProfilesDir
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

	k8sutil.GetCoreV1Func = common.MockGetCoreV1()
	k8sutil.GetDynamicClientFunc = common.MockDynamicClient()
	defer func() {
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
	}()
	testFunc := func(client typedcorev1.CoreV1Interface, dynClient dynamic.Interface) (bool, error) { return false, nil }
	rancher.SetCheckClusterProvisionedFunc(testFunc)
	defer rancher.SetDefaultCheckClusterProvisionedFunc()

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer func() { cmCleanupFunc = cmissuer.UninstallCleanup }()
	cmCleanupFunc = func(log vzlog.VerrazzanoLogger, cli client.Client, namespace string) error { return nil }

	// Expect a call to get the Verrazzano resource.  Return resource with deleted timestamp.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano, opts ...client.GetOption) error {
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

	config.TestProfilesDir = relativeProfilesDir
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
// GIVEN a request for a Verrazzano custom resource
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
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil()), gomock.Any()).
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
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil()), gomock.Any()).
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
		Get(gomock.Any(), types.NamespacedName{Name: constants.VerrazzanoSystemNamespace}, gomock.Not(gomock.Nil()), gomock.Any()).
		Return(errors.NewBadRequest(errMsg))

	config.TestProfilesDir = relativeProfilesDir
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
		Get(gomock.Any(), types.NamespacedName{Name: constants.VerrazzanoSystemNamespace}, gomock.Not(gomock.Nil()), gomock.Any()).
		Return(errors.NewNotFound(schema.ParseGroupResource("namespaces"), constants.VerrazzanoSystemNamespace))

	// Expect a call to create the Verrazzano system namespace - return a failure error
	mock.EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Namespace{})).
		Return(errors.NewBadRequest(errMsg))

	config.TestProfilesDir = relativeProfilesDir
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
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoInstallNamespace, Name: "test-oci-config-secret"}, gomock.Not(gomock.Nil()), gomock.Any()).
		Return(errors.NewBadRequest("failed to get Secret"))

	config.TestProfilesDir = relativeProfilesDir
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
		StatusUpdater:     &vzstatus.FakeVerrazzanoStatusUpdater{Client: c},
	}
	return reconciler
}

// Expect syncLocalRegistration related calls, happy-path secret exists
func expectSyncLocalRegistration(_ *testing.T, mock *mocks.MockClient, _ string) {
	// Expect a call to get the Agent secret in the verrazzano-system namespace - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil()), gomock.Any()).
		Return(nil)
}

// expectGetVerrazzanoSystemNamespaceExists expects a call to get the Verrazzano system namespace and returns
// that it exists
func expectGetVerrazzanoSystemNamespaceExists(mock *mocks.MockClient, _ *assert.Assertions) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: constants.VerrazzanoSystemNamespace}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace, opts ...client.GetOption) error {
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
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: clusterRoleBindingName}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, nsName types.NamespacedName, clusterRoleBinding *rbacv1.ClusterRoleBinding, opts ...client.GetOption) error {
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
		Get(gomock.Any(), types.NamespacedName{Namespace: getInstallNamespace(), Name: buildServiceAccountName(name)}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, serviceAccount *corev1.ServiceAccount, opts ...client.GetOption) error {
			newSA := rbac.NewServiceAccount(name.Namespace, name.Name, []string{}, labels)
			serviceAccount.ObjectMeta = newSA.ObjectMeta
			return nil
		}).AnyTimes()
}

// expectGetVerrazzanoExists expects a call to get a Verrazzano with the given namespace and name, and returns
// one that has the same content as the verrazzanoToUse argument
func expectGetVerrazzanoExists(mock *mocks.MockClient, verrazzanoToUse vzapi.Verrazzano, namespace string, name string, _ map[string]string) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano, opts ...client.GetOption) error {
			verrazzano.TypeMeta = verrazzanoToUse.TypeMeta
			verrazzano.ObjectMeta = verrazzanoToUse.ObjectMeta
			verrazzano.Spec.Components.DNS = verrazzanoToUse.Spec.Components.DNS
			verrazzano.Status = verrazzanoToUse.Status
			return nil
		}).AnyTimes()
}

// expectGetIngressListExists expects a call to get the ingressList
func expectGetIngressListExists(mock *mocks.MockClient) {
	// Expect a call to get the ServiceAccount - return that it exists
	mock.EXPECT().
		List(gomock.Any(), &networkingv1.IngressList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, ingressList *networkingv1.IngressList, options ...*client.ListOptions) error {
			return nil
		}).AnyTimes()
}

func expectSharedNamespaceDeletes(mock *mocks.MockClient) {
	const fakeNS = "fake"
	for _, ns := range sharedNamespaces {
		mock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Name: ns}, gomock.Not(gomock.Nil()), gomock.Any()).
			Return(nil)
		mock.EXPECT().Delete(gomock.Any(), nsMatcher{Name: ns}, gomock.Any()).Return(nil)
		mock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Name: ns}, gomock.Not(gomock.Nil()), gomock.Any()).
			Return(errors.NewNotFound(schema.ParseGroupResource("Namespace"), ns))
	}
	// Expect delete for component namesapces
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: fakeNS}, gomock.Not(gomock.Nil()), gomock.Any()).
		Return(nil)
	mock.EXPECT().Delete(gomock.Any(), nsMatcher{Name: fakeNS}, gomock.Any()).Return(nil)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: fakeNS}, gomock.Not(gomock.Nil()), gomock.Any()).
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
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil()), gomock.Any()).
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
	_, err = reconciler.Reconcile(context.TODO(), errorRequest)
	if err != nil {
		return
	}
	errorCounterAfter := testutil.ToFloat64(reconcileErrorCounterMetric.Get())
	assert.NoError(t, err)
	asserts.Equal(errorCounterBefore, errorCounterAfter-1)
}

// TestUninstallJobCleanup tests the uninstall job cleanup function
// GIVEN a completed uninstall job
// WHEN the function is called
// THEN the job and associated pod are deleted
func TestUninstallJobCleanup(t *testing.T) {
	asserts := assert.New(t)
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.VerrazzanoInstallNamespace,
				Name:      "uninstall-201321",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"job-name": "uninstall-201321"},
			},
		},
	).Build()
	reconciler := newVerrazzanoReconciler(c)
	err := reconciler.cleanupUninstallJob("uninstall-201321", constants.VerrazzanoInstallNamespace, vzlog.DefaultLogger())
	asserts.Nil(err)
	job := &batchv1.Job{}
	err = c.Get(context.TODO(), client.ObjectKey{Name: "one-off-backup-20221018-201321", Namespace: constants.KeycloakNamespace}, job)
	asserts.NotNil(err)
	asserts.True(errors.IsNotFound(err))
}

// TestMysqlBackupJobCleanup tests the MySQL backup job cleanup function
// GIVEN a completed backup job
// WHEN the function is called
// THEN the job and associated pod are deleted
func TestMysqlBackupJobCleanup(t *testing.T) {
	asserts := assert.New(t)
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.KeycloakNamespace,
				Name:      "one-off-backup-20221018-201321",
				Labels:    map[string]string{"app.kubernetes.io/created-by": constants3.MySQLOperator},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"job-name": "one-off-backup-20221018-201321"},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "operator-backup-job",
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								ExitCode: 0,
							},
						},
					},
				},
			},
		},
	).Build()
	reconciler := newVerrazzanoReconciler(c)
	err := reconciler.cleanupMysqlBackupJob(vzlog.DefaultLogger())
	asserts.Nil(err)
	job := &batchv1.Job{}
	err = c.Get(context.TODO(), client.ObjectKey{Name: "one-off-backup-20221018-201321", Namespace: constants.KeycloakNamespace}, job)
	asserts.NotNil(err)
	asserts.True(errors.IsNotFound(err))
}

// TestMysqlScheduledBackupJobCleanup tests the MySQL backup job cleanup function
// GIVEN a completed scheduled backup job
// WHEN the function is called
// THEN the job and associated pod are deleted
func TestMysqlScheduledBackupJobCleanup(t *testing.T) {
	asserts := assert.New(t)
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "one-off-backup-schedule20221018-201321",
				Namespace: constants.KeycloakNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "batch/batchv1",
						Kind:       "CronJob",
						Name:       "one-off-backup-schedule-20221018-201321",
					},
				},
			},
		},
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.KeycloakNamespace,
				Name:      "one-off-backup-schedule-20221018-201321",
				Labels:    map[string]string{"app.kubernetes.io/created-by": constants3.MySQLOperator},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"job-name": "one-off-backup-20221018-201321"},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "operator-backup-job",
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								ExitCode: 0,
							},
						},
					},
				},
			},
		},
	).Build()
	reconciler := newVerrazzanoReconciler(c)
	err := reconciler.cleanupMysqlBackupJob(vzlog.DefaultLogger())
	asserts.Nil(err)
	job := &batchv1.Job{}
	err = c.Get(context.TODO(), client.ObjectKey{Name: "one-off-backup-20221018-201321", Namespace: constants.KeycloakNamespace}, job)
	asserts.NotNil(err)
	asserts.True(errors.IsNotFound(err))
	cronJob := &batchv1.CronJob{}
	err = c.Get(context.TODO(), client.ObjectKey{Name: "one-off-backup-schedule-20221018-201321", Namespace: constants.KeycloakNamespace}, cronJob)
	asserts.Nil(err)
}

// TestInProgressMysqlBackupJobCleanup tests the MySQL backup job cleanup function
// GIVEN a not completed backup job
// WHEN the function is called
// THEN the job and associated pod are not deleted
func TestInProgressMysqlBackupJobCleanup(t *testing.T) {
	asserts := assert.New(t)
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.KeycloakNamespace,
				Name:      "one-off-backup-20221018-201321",
				Labels:    map[string]string{"app.kubernetes.io/created-by": constants3.MySQLOperator},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"job-name": "one-off-backup-20221018-201321"},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "operator-backup-job",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
			},
		},
	).Build()
	reconciler := newVerrazzanoReconciler(c)
	err := reconciler.cleanupMysqlBackupJob(vzlog.DefaultLogger())
	asserts.NotNil(err)
	job := &batchv1.Job{}
	err = c.Get(context.TODO(), client.ObjectKey{Name: "one-off-backup-20221018-201321", Namespace: constants.KeycloakNamespace}, job)
	asserts.Nil(err)
	asserts.NotNil(job)
}

// TestFailedMysqlBackupJobCleanup tests the MySQL backup job cleanup function
// GIVEN a completed backup job with a non-zero exit code
// WHEN the function is called
// THEN the job and associated pod are not deleted
func TestFailedMysqlBackupJobCleanup(t *testing.T) {
	asserts := assert.New(t)
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.KeycloakNamespace,
				Name:      "one-off-backup-20221018-201321",
				Labels:    map[string]string{"app.kubernetes.io/created-by": constants3.MySQLOperator},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"job-name": "one-off-backup-20221018-201321"},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "operator-backup-job",
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								ExitCode: 1,
							},
						},
					},
				},
			},
		},
	).Build()
	reconciler := newVerrazzanoReconciler(c)
	err := reconciler.cleanupMysqlBackupJob(vzlog.DefaultLogger())
	asserts.NotNil(err)
	job := &batchv1.Job{}
	err = c.Get(context.TODO(), client.ObjectKey{Name: "one-off-backup-20221018-201321", Namespace: constants.KeycloakNamespace}, job)
	asserts.Nil(err)
	asserts.NotNil(job)
}

// erroringFakeClient wraps a k8s client and returns an error when List is called
type erroringFakeClient struct {
	client.Client
}

// List always returns an error - used to simulate an error listing a resource
func (e *erroringFakeClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return errors.NewNotFound(schema.GroupResource{}, "")
}

// TestNoMysqlBackupJobsFound tests the MySQL backup job cleanup function
// GIVEN no jobs are available
// WHEN the function is called
// THEN the cleanup is skipped
func TestNoMysqlBackupJobsFound(t *testing.T) {
	asserts := assert.New(t)
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	reconciler := newVerrazzanoReconciler(&erroringFakeClient{c})
	err := reconciler.cleanupMysqlBackupJob(vzlog.DefaultLogger())
	asserts.Nil(err)
}

// TestMysqlOperatorJobPredicateWrongNamespace tests the MySQL operator job predicate
// GIVEN a create event for a job in a namespace other than 'keycloak'
// WHEN the function is called
// THEN the result is false
func TestMysqlOperatorJobPredicateWrongNamespace(t *testing.T) {
	asserts := assert.New(t)
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	reconciler := newVerrazzanoReconciler(c)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "WrongNamespace",
			Name:      "one-off-backup-20221018-201321",
			Labels:    map[string]string{"app.kubernetes.io/created-by": constants3.MySQLOperator},
		},
	}
	isMysqlJob := reconciler.isMysqlOperatorJob(event.CreateEvent{Object: job}, vzlog.DefaultLogger())
	asserts.False(isMysqlJob)
}

// TestMysqlOperatorJobPredicateOwnedByOperatorCronJob tests the MySQL operator job predicate
// GIVEN a create event for a job owned by a cron job created by the operator
// WHEN the function is called
// THEN the result is true
func TestMysqlOperatorJobPredicateOwnedByOperatorCronJob(t *testing.T) {
	asserts := assert.New(t)
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.KeycloakNamespace,
				Name:      "one-off-backup-schedule-20221018-201321",
				Labels:    map[string]string{"app.kubernetes.io/created-by": constants3.MySQLOperator},
			},
		},
	).Build()
	reconciler := newVerrazzanoReconciler(c)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: mysql.ComponentNamespace,
			Name:      "one-off-backup-20221018-201321",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "batch/batchv1",
					Kind:       "CronJob",
					Name:       "one-off-backup-schedule-20221018-201321",
				},
			},
		},
	}
	isMysqlJob := reconciler.isMysqlOperatorJob(event.CreateEvent{Object: job}, vzlog.DefaultLogger())
	asserts.True(isMysqlJob)
}

// TestMysqlOperatorJobPredicateValidBackupJob tests the MySQL operator job predicate
// GIVEN a create event for a job directly created by the operator
// WHEN the function is called
// THEN the result is true
func TestMysqlOperatorJobPredicateValidBackupJob(t *testing.T) {
	asserts := assert.New(t)
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	reconciler := newVerrazzanoReconciler(c)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: mysql.ComponentNamespace,
			Name:      "one-off-backup-20221018-201321",
			Labels:    map[string]string{"app.kubernetes.io/created-by": constants3.MySQLOperator},
		},
	}
	isMysqlJob := reconciler.isMysqlOperatorJob(event.CreateEvent{Object: job}, vzlog.DefaultLogger())
	asserts.True(isMysqlJob)
}

// TestReconcilerInitForVzResource tests initForVzResource to verify the initialization for the given Verrazzano resource
func TestReconcilerInitForVzResource(t *testing.T) {
	type args struct {
		vz  *vzapi.Verrazzano
		log vzlog.VerrazzanoLogger
	}
	logger := vzlog.DefaultLogger()
	mocker := gomock.NewController(t)
	podKind := &source.Kind{Type: &corev1.Pod{}}
	jobKind := &source.Kind{Type: &batchv1.Job{}}
	secretKind := &source.Kind{Type: &corev1.Secret{}}
	namespaceKind := &source.Kind{Type: &corev1.Namespace{}}
	getNoErrorMock := func() client.Client {
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Delete(context.TODO(), gomock.Not(nil), gomock.Any()).Return(nil).AnyTimes()
		return mockClient
	}
	getDeleteErrorMock := func() client.Client {
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Delete(context.TODO(), gomock.Not(nil), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	getSADeleteErrorMock := func() client.Client {
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Delete(context.TODO(), gomock.Not(nil), gomock.Any()).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{}), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	getUpdateErrorMock := func() client.Client {
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Update(context.TODO(), gomock.Any()).Return(fmt.Errorf(unExpectedError)).Times(1)
		return mockClient
	}
	setMockControllerNoErr := func(reconciler *Reconciler) {
		controller := mocks.NewMockController(mocker)
		controller.EXPECT().Watch(gomock.Eq(podKind), gomock.Any(), gomock.Any()).Return(nil)
		controller.EXPECT().Watch(gomock.Eq(jobKind), gomock.Any(), gomock.Any()).Return(nil)
		// watches 2 secrets - managed cluster registration and Thanos internal user
		controller.EXPECT().Watch(gomock.Eq(secretKind), gomock.Any(), gomock.Any()).Return(nil).Times(2)
		controller.EXPECT().Watch(gomock.Eq(namespaceKind), gomock.Any(), gomock.Any()).Return(nil)
		reconciler.Controller = controller
	}
	setMockControllerPodWatchErr := func(reconciler *Reconciler) {
		controller := mocks.NewMockController(mocker)
		controller.EXPECT().Watch(gomock.Eq(podKind), gomock.Any(), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		reconciler.Controller = controller
	}
	setMockControllerJobWatchErr := func(reconciler *Reconciler) {
		controller := mocks.NewMockController(mocker)
		// pod watch succeeds, job watch fails
		controller.EXPECT().Watch(gomock.Eq(podKind), gomock.Any(), gomock.Any()).Return(nil)
		controller.EXPECT().Watch(gomock.Eq(jobKind), gomock.Any(), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		reconciler.Controller = controller
	}
	setMockControllerSecretWatchErr := func(reconciler *Reconciler) {
		controller := mocks.NewMockController(mocker)
		// pod and job watch succeeds, first secret watch fails
		// TODO find a way to know which secret is being watched and fail each one selectively
		controller.EXPECT().Watch(gomock.Eq(podKind), gomock.Any(), gomock.Any()).Return(nil)
		controller.EXPECT().Watch(gomock.Eq(jobKind), gomock.Any(), gomock.Any()).Return(nil)
		controller.EXPECT().Watch(gomock.Eq(secretKind), gomock.Any(), gomock.Any()).DoAndReturn(
			func(kind *source.Kind, handler handler.EventHandler, funcs predicate.Funcs) error {
				return fmt.Errorf(unExpectedError)
			})
		reconciler.Controller = controller
	}
	setMockControllerNamespaceWatchErr := func(reconciler *Reconciler) {
		controller := mocks.NewMockController(mocker)
		// pod and job watch succeeds, first secret watch fails
		// TODO find a way to know which secret is being watched and fail each one selectively
		controller.EXPECT().Watch(gomock.Eq(podKind), gomock.Any(), gomock.Any()).Return(nil)
		controller.EXPECT().Watch(gomock.Eq(jobKind), gomock.Any(), gomock.Any()).Return(nil)
		controller.EXPECT().Watch(gomock.Eq(secretKind), gomock.Any(), gomock.Any()).Return(nil).Times(2)
		controller.EXPECT().Watch(gomock.Eq(namespaceKind), gomock.Any(), gomock.Any()).DoAndReturn(
			func(kind *source.Kind, handler handler.EventHandler, funcs predicate.Funcs) error {
				return fmt.Errorf(unExpectedError)
			})
		reconciler.Controller = controller
	}
	vzName := "vzTestName"
	arg := args{
		&vzapi.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Name: vzName,
			},
		},
		logger,
	}
	testVZ := "testVZ"
	initializedSet[testVZ] = true
	defer func() {
		unitTesting = true
		mocker.Finish()
	}()
	argWithInit := args{
		&vzapi.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Name:       testVZ,
				Finalizers: []string{finalizerName},
			},
		},
		logger,
	}
	argWithFinalizer := args{
		&vzapi.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Name:       vzName,
				Finalizers: []string{finalizerName},
			},
		},
		logger,
	}
	tests := []struct {
		name               string
		args               args
		getClientFunc      func() client.Client
		mockControllerFunc func(*Reconciler)
		want               ctrl.Result
		wantErr            bool
	}{
		// GIVEN Verrazzano CR
		// WHEN initForVzResource is called and error occurs while updating k8s resource
		// THEN error is returned with result for requeue with delay
		{
			"TestReconcilerInitForVzResource when updates the given obj in the K8s cluster fails",
			arg,
			getUpdateErrorMock,
			nil,
			newRequeueWithDelay(),
			true,
		},
		// GIVEN Verrazzano CR
		// WHEN initForVzResource is called and no error occurs
		// THEN no error is returned with result for no requeue
		{
			"TestReconcilerInitForVzResource when initialization is already for this resource",
			argWithInit,
			nil,
			nil,
			ctrl.Result{},
			false,
		},
		// GIVEN Verrazzano CR
		// WHEN initForVzResource is called and no error occurs
		// THEN no error is returned with result for requeue with delay
		{
			"TestReconcilerInitForVzResource when no error occurs",
			argWithFinalizer,
			getNoErrorMock,
			setMockControllerNoErr,
			ctrl.Result{Requeue: true},
			false,
		},
		// GIVEN Verrazzano CR
		// WHEN initForVzResource is called and deletion of cluster role-binding fails
		// THEN error is returned with result for requeue with delay
		{
			"TestReconcilerInitForVzResource when deletion of cluster role-binding fails",
			argWithFinalizer,
			getDeleteErrorMock,
			nil,
			newRequeueWithDelay(),
			true,
		},
		// GIVEN Verrazzano CR
		// WHEN initForVzResource is called and deletion of Service account fails
		// THEN error is returned with result for requeue with delay
		{
			"TestReconcilerInitForVzResource when deletion of Service account fails",
			argWithFinalizer,
			getSADeleteErrorMock,
			nil,
			newRequeueWithDelay(),
			true,
		},
		// GIVEN Verrazzano CR
		// WHEN initForVzResource is called and error occurs while watching pods
		// THEN error is returned with result for requeue with delay
		{"TestReconcilerInitForVzResource when watching for pods failed",
			argWithFinalizer,
			getNoErrorMock,
			setMockControllerPodWatchErr,
			newRequeueWithDelay(),
			true,
		},
		// GIVEN Verrazzano CR
		// WHEN initForVzResource is called and error occurs while watching Jobs
		// THEN error is returned with result for requeue with delay
		{"TestReconcilerInitForVzResource when watching for jobs failed",
			argWithFinalizer,
			getNoErrorMock,
			setMockControllerJobWatchErr,
			newRequeueWithDelay(),
			true,
		},
		// GIVEN Verrazzano CR
		// WHEN initForVzResource is called and error occurs while watching Secrets
		// THEN error is returned with result for requeue with delay
		{"TestReconcilerInitForVzResource when watching for registration secret failed",
			argWithFinalizer,
			getNoErrorMock,
			setMockControllerSecretWatchErr,
			newRequeueWithDelay(),
			true,
		},
		// GIVEN Verrazzano CR
		// WHEN initForVzResource is called and error occurs while watching a namespace creation/update
		// THEN error is returned with result for requeue with delay
		{"TestReconcilerInitForVzResource when watching for namespace creation failed",
			argWithFinalizer,
			getNoErrorMock,
			setMockControllerNamespaceWatchErr,
			newRequeueWithDelay(),
			true,
		},
	}
	unitTesting = false
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reconciler Reconciler
			if tt.getClientFunc != nil {
				reconciler = newVerrazzanoReconciler(tt.getClientFunc())
			} else {
				reconciler = newVerrazzanoReconciler(nil)
			}
			if tt.mockControllerFunc != nil {
				tt.mockControllerFunc(&reconciler)
			}
			got, err := reconciler.initForVzResource(tt.args.vz, tt.args.log)
			if (err != nil) != tt.wantErr {
				t.Errorf("initForVzResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("initForVzResource() got = %v, want %v", got, tt.want)
			}
			delete(initializedSet, tt.args.vz.Name)
		})
	}
}

// TestReconcilerWatch tests AddWatch and ClearWatch
// GIVEN component name to watch
// WHEN AddWatch is called
// THEN component is added to watch component map
// WHEN ClearWatch is called
// THEN component is cleared from watch component map
func TestReconcilerWatch(t *testing.T) {
	asserts := assert.New(t)
	reconciler := newVerrazzanoReconciler(nil)
	watchComp := "testName"
	reconciler.AddWatch(watchComp)
	asserts.True(reconciler.WatchedComponents[watchComp])
	reconciler.ClearWatch(watchComp)
	asserts.False(reconciler.WatchedComponents[watchComp])
}

// TestIsManagedClusterRegistrationSecret tests isManagedClusterRegistrationSecret
// GIVEN Secret resource
// WHEN isManagedClusterRegistrationSecret is called
// THEN true is returned if secret is MCRegistrationSecret
// THEN false is returned if secret is not MCRegistrationSecret
func TestIsManagedClusterRegistrationSecret(t *testing.T) {
	asserts := assert.New(t)
	reconciler := newVerrazzanoReconciler(nil)
	vzSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.MCRegistrationSecret,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	secret := corev1.Secret{}
	asserts.True(reconciler.isManagedClusterRegistrationSecret(&vzSecret))
	asserts.False(reconciler.isManagedClusterRegistrationSecret(&secret))
}

// TestIsThanosInternalUserSecret tests isThanosInternalUserSecret
// GIVEN Secret resource
// WHEN isThanosInternalUserSecret is called
// THEN true is returned if secret is the Thanos internal user secret
// THEN false is returned if secret is not the Thanos internal user secret
func TestIsThanosInternalUserSecret(t *testing.T) {
	asserts := assert.New(t)
	reconciler := newVerrazzanoReconciler(nil)
	thanosSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.ThanosInternalUserSecretName,
			Namespace: constants.VerrazzanoMonitoringNamespace,
		},
	}
	secretSameNS := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-other-secret",
			Namespace: constants.VerrazzanoMonitoringNamespace,
		},
	}
	secretSameNameDiffNS := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.ThanosInternalUserSecretName,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	asserts.True(reconciler.isThanosInternalUserSecret(&thanosSecret))
	asserts.False(reconciler.isThanosInternalUserSecret(&secretSameNS))
	asserts.False(reconciler.isThanosInternalUserSecret(&secretSameNameDiffNS))
}

// TestIsCattleGlobalDataNamespace tests isCattleGlobalDataNamespace
// GIVEN Namespace resource
// WHEN isCattleGlobalDataNamespace is called
// THEN true is returned if namespace is the cattle global data namespace
// THEN false is returned if namespace is not the cattle global data namespace
func TestIsCattleGlobalDataNamespace(t *testing.T) {
	asserts := assert.New(t)
	reconciler := newVerrazzanoReconciler(nil)
	globalDataNamesapce := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: rancher.CattleGlobalDataNamespace,
		},
	}
	otherNS := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "some-other-ns",
		},
	}
	asserts.True(reconciler.isCattleGlobalDataNamespace(&globalDataNamesapce))
	asserts.False(reconciler.isCattleGlobalDataNamespace(&otherNS))
}

// TestReconcile tests Reconcile
func TestReconcile(t *testing.T) {
	type args struct {
		ctx context.Context
		req ctrl.Request
	}
	temp := metricsexporter.MetricsExp
	defer func() {
		metricsexporter.MetricsExp = temp
	}()
	argsWithNilCtx := args{nil, ctrl.Request{}}
	argsWithCtx := args{context.TODO(), ctrl.Request{}}
	tests := []struct {
		name     string
		args     args
		testInit func()
		want     ctrl.Result
		wantErr  bool
	}{
		// GIVEN reconciler object
		// WHEN Reconciler is called with nil context
		// THEN error is returned with empty result of a Reconciler invocation
		{
			"TestReconcile with nil context",
			argsWithNilCtx,
			nil,
			ctrl.Result{},
			true,
		},
		// GIVEN reconciler object
		// WHEN Reconciler is called
		// THEN error is returned with empty result of a Reconciler invocation if required metrics not found
		{
			"TestReconcile when ReconcileCounter metric not found",
			argsWithCtx,
			func() {
				metricsexporter.MetricsExp = metricsexporter.MetricsExporter{}
			},
			ctrl.Result{},
			true,
		},
		// GIVEN reconciler object
		// WHEN Reconciler is called and error metric not found
		// THEN error is returned with empty result of a Reconciler invocation
		{
			"TestReconcile when doReconcile throw error metric not found",
			argsWithCtx,
			nil,
			ctrl.Result{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newVerrazzanoReconciler(nil)
			if tt.testInit != nil {
				tt.testInit()
			}
			got, err := r.Reconcile(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reconcile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPersistJobLog tests persistJobLog
// GIVEN backupjob and job pod
// WHEN persistJobLog is called
// THEN false is returned if no error occurs
func TestPersistJobLog(t *testing.T) {
	type args struct {
		backupJob batchv1.Job
		jobPod    *corev1.Pod
		log       vzlog.VerrazzanoLogger
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"TestPersistJobLog",
			args{batchv1.Job{ObjectMeta: metav1.ObjectMeta{
				Name: "test-schedule-1",
			}}, &corev1.Pod{}, vzlog.DefaultLogger()},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := persistJobLog(tt.args.backupJob, tt.args.jobPod, tt.args.log); got != tt.want {
				t.Errorf("persistJobLog() = %v, want %v", got, tt.want)
			}
		})
	}
}

// dummy Status updater for testing purpose
type statusUpdater string

func (s *statusUpdater) Update(_ context.Context, _ client.Object, _ ...client.UpdateOption) error {
	return nil
}
func (s *statusUpdater) Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error {
	return nil
}

func createRelease(_ string, status release.Status) *release.Release {
	now := time2.Now()
	return &release.Release{
		Name:      rancher.ComponentName,
		Namespace: rancher.ComponentNamespace,
		Info: &release.Info{
			FirstDeployed: now,
			LastDeployed:  now,
			Status:        status,
			Description:   "Named Release Stub",
		},
		Version: 1,
	}
}

func testActionConfigWithInstallation(vzlog.VerrazzanoLogger, *cli.EnvSettings, string) (*action.Configuration, error) {
	return helm.CreateActionConfig(true, rancher.ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), createRelease)
}

func testActionConfigWithoutInstallation(vzlog.VerrazzanoLogger, *cli.EnvSettings, string) (*action.Configuration, error) {
	return helm.CreateActionConfig(false, rancher.ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), createRelease)
}

// TestReconcilerProcReadyState tests ProcReadyState
func TestReconcilerProcReadyState(t *testing.T) {
	temp := unitTesting
	defer func() {
		unitTesting = temp
	}()
	unitTesting = false
	helmOverrideNoErr := func() {
		helm.SetActionConfigFunction(testActionConfigWithInstallation)
	}

	k8sClient := fakes.NewClientBuilder().WithScheme(newScheme()).Build()
	mocker := gomock.NewController(t)
	var statusUpdaterMock statusUpdater = "test"
	getMockClient := func() client.Client {
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.Any()).Return(nil).Times(3)
		mockClient.EXPECT().Update(context.TODO(), gomock.Not(nil)).Return(nil).Times(1)
		mockClient.EXPECT().Status().Return(&statusUpdaterMock)
		mockClient.EXPECT().Delete(context.TODO(), gomock.Not(nil), gomock.Any()).Return(nil)
		return mockClient
	}
	getClientDeleteError := func() client.Client {
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.Any()).Return(nil).Times(3)
		mockClient.EXPECT().Update(context.TODO(), gomock.Not(nil)).Return(nil).Times(1)
		mockClient.EXPECT().Delete(context.TODO(), gomock.Not(nil), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	context, _ := vzContext.NewVerrazzanoContext(vzlog.DefaultLogger(), k8sClient, &vzapi.Verrazzano{}, true)
	contextWithCompReady, _ := vzContext.NewVerrazzanoContext(vzlog.DefaultLogger(), k8sClient, &vzapi.Verrazzano{
		Status: vzapi.VerrazzanoStatus{
			Components: map[string]*vzapi.ComponentStatusDetails{
				rancher.ComponentName: {
					State: vzapi.ComponentAvailable,
				},
			},
		},
	}, true)
	getCompFunc := func() []spi.Component {
		return []spi.Component{rancher.NewComponent()}
	}
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	tests := []struct {
		name           string
		vzContext      vzContext.VerrazzanoContext
		k8sClient      client.Client
		setProfileFunc func()
		helmOverride   func()
		want           ctrl.Result
		wantErr        bool
	}{
		// GIVEN Reconciler object
		// WHEN ProcReadyState is called and component is already installed
		// THEN no error is returned with result of a Reconciler invocation with delay
		{
			"TestReconcilerProcReadyState when component is already installed",
			context,
			k8sClient,
			func() {
				config.TestProfilesDir = relativeProfilesDir
			},
			helmOverrideNoErr,
			newRequeueWithDelay(),
			false,
		},
		// GIVEN Reconciler object
		// WHEN ProcReadyState is called and component is in ready state
		// THEN no error is returned with result of a Reconciler invocation with delay
		{
			"TestReconcilerProcReadyState when component is in ready state",
			contextWithCompReady,
			getMockClient(),
			func() {
				config.TestProfilesDir = relativeProfilesDir
			},
			helmOverrideNoErr,
			newRequeueWithDelay(),
			false,
		},
		// GIVEN Reconciler object
		// WHEN ProcReadyState is called and error occurs while deleting resource
		// THEN error is returned with result of a Reconciler invocation with delay
		{
			"TestReconcilerProcReadyState when error occurs while deleting resource",
			contextWithCompReady,
			getClientDeleteError(),
			func() {
				config.TestProfilesDir = relativeProfilesDir
			},
			helmOverrideNoErr,
			newRequeueWithDelay(),
			true,
		},
	}
	defer func() { config.TestProfilesDir = "" }()
	defer registry.ResetGetComponentsFn()
	defer helm.SetDefaultActionConfigFunction()
	for _, tt := range tests {
		registry.OverrideGetComponentsFn(getCompFunc)
		t.Run(tt.name, func(t *testing.T) {
			k8sutil.GetCoreV1Func = common.MockGetCoreV1WithNamespace("cattle-system")
			defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()

			r := newVerrazzanoReconciler(tt.k8sClient)
			if tt.setProfileFunc != nil {
				tt.setProfileFunc()
			}
			if tt.helmOverride != nil {
				tt.helmOverride()
			}
			got, err := r.ProcReadyState(tt.vzContext)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcReadyState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProcReadyState() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestReconcilerProcReconcilingState tests ProcReconciling state
// GIVEN a Verrazzano in Reconciling state with status version == spec version
// WHEN ProcReconcilingState is called
// THEN it proceeds with reconcile
// GIVEN a Verrazzano in Reconciling state with status version != spec version
// WHEN ProcReconcilingState is called
// THEN it updates the VZ status to Upgrading returns a requeue
func TestReconcilerProcReconcilingState(t *testing.T) {
	vzNoSpecVersion := vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: "nospecver"},
		Status: vzapi.VerrazzanoStatus{
			State:      vzapi.VzStateReconciling,
			Components: map[string]*vzapi.ComponentStatusDetails{},
			Version:    "1.6.3",
		},
	}
	// spec and status versions are equal
	vzVersionsSame := vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: "same-versions"},
		Spec: vzapi.VerrazzanoSpec{
			Version: "1.6.3",
		},
		Status: vzapi.VerrazzanoStatus{
			State:      vzapi.VzStateReconciling,
			Components: map[string]*vzapi.ComponentStatusDetails{},
			Version:    "1.6.3",
		},
	}
	// spec and status versions are different
	vzVersionsDifferent := vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: "diff-versions"},
		Spec: vzapi.VerrazzanoSpec{
			Version: "1.6.3",
		},
		Status: vzapi.VerrazzanoStatus{
			State:      vzapi.VzStateReconciling,
			Components: map[string]*vzapi.ComponentStatusDetails{},
			Version:    "1.5.3",
		},
	}
	tests := []struct {
		name        string
		vz          vzapi.Verrazzano
		wantVZState vzapi.VzStateType
		wantRequeue bool
	}{
		{"VZ with no spec version", vzNoSpecVersion, vzapi.VzStateReconciling, false},
		{"VZ with same spec and status version", vzVersionsSame, vzapi.VzStateReconciling, false},
		{"VZ with different spec and status version", vzVersionsDifferent, vzapi.VzStateUpgrading, true},
	}
	getCompFunc := func() []spi.Component {
		return []spi.Component{}
	}
	initUnitTesing()
	origSkipReconcile := unitTestSkipReconcile
	// don't test reconcile
	unitTestSkipReconcile = true

	defer func() { unitTestSkipReconcile = origSkipReconcile }()
	metricsexporter.Init()
	defer registry.ResetGetComponentsFn()
	registry.OverrideGetComponentsFn(getCompFunc)
	defer func() { config.TestProfilesDir = "" }()
	config.TestProfilesDir = relativeProfilesDir
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fakes.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(&tt.vz).Build()
			r := newVerrazzanoReconciler(k8sClient)
			vzCtx, err := vzContext.NewVerrazzanoContext(vzlog.DefaultLogger(), k8sClient, &tt.vz, true)
			assert.NoError(t, err)
			result, err := r.ProcReconcilingState(vzCtx)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantRequeue, result.Requeue)
			vz := vzapi.Verrazzano{}
			err = k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: tt.vz.Namespace, Name: tt.vz.Name}, &vz)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantVZState, vz.Status.State)
		})
	}
}
