// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/uninstalljob"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s"
	"testing"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/rbac"
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// For unit testing
const testBomFilePath = "testdata/test_bom.json"

// Generate mocks for the Kerberos Client and StatusWriter interfaces for use in tests.
//go:generate mockgen -destination=../../mocks/controller_mock.go -package=mocks -copyright_file=../../hack/boilerplate.go.txt sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter

const installPrefix = "verrazzano-install-"
const uninstallPrefix = "verrazzano-uninstall-"

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

// TestSuccessfulInstall tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource
// WHEN a Verrazzano resource has been applied
// THEN ensure all the objects are already created
func TestSuccessfulInstall(t *testing.T) {
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
	defer func() { config.TestHelmConfigDir = "" }()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	verrazzanoToUse.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	verrazzanoToUse.ObjectMeta = metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
		Labels:    labels}
	verrazzanoToUse.Spec.Components.DNS = &vzapi.DNSComponent{External: &vzapi.External{Suffix: "mydomain.com"}}

	verrazzanoToUse.Status.State = vzapi.Ready
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
	result, err := reconciler.Reconcile(request)
	asserts.NoError(err)

	// Validate the results
	mocker.Finish()
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
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
		Namespace: namespace,
		Name:      name,
		Labels:    labels}
	verrazzanoToUse.Status.State = vzapi.Ready

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
	result, err := reconciler.Reconcile(request)
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)

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
				clusters.ManagedClusterNameKey: []byte("cluster1"),
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

// TestCreateVerrazzano tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource
// WHEN a Verrazzano resource has been created
// THEN ensure all the objects are created
func TestCreateVerrazzano(t *testing.T) {
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
		Namespace: namespace,
		Name:      name,
		Labels:    labels}

	vzToUse.Status.Components = makeVerrazzanoComponentStatusMap()
	vzToUse.Status.State = vzapi.Ready

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

	// Expect a call to get the ServiceAccount - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: getInstallNamespace(), Name: buildServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "ServiceAccount"}, buildServiceAccountName(name)))

	// Expect a call to create the ServiceAccount - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, serviceAccount *corev1.ServiceAccount, opts ...client.CreateOption) error {
			asserts.Equalf(getInstallNamespace(), serviceAccount.Namespace, "ServiceAccount namespace did not match")
			asserts.Equalf(buildServiceAccountName(name), serviceAccount.Name, "ServiceAccount name did not match")
			asserts.Equalf(labels, serviceAccount.Labels, "ServiceAccount labels did not match")
			return nil
		})

	// Expect a call to get the ClusterRoleBinding - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: buildClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "ClusterRoleBinding"}, buildClusterRoleBindingName(namespace, name)))

	// Expect a call to create the ClusterRoleBinding - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, clusterRoleBinding *rbacv1.ClusterRoleBinding, opts ...client.CreateOption) error {
			asserts.Equalf("", clusterRoleBinding.Namespace, "ClusterRoleBinding namespace did not match")
			asserts.Equalf(buildClusterRoleBindingName(namespace, name), clusterRoleBinding.Name, "ClusterRoleBinding name did not match")
			asserts.Equalf(labels, clusterRoleBinding.Labels, "ClusterRoleBinding labels did not match")
			asserts.Equalf(buildServiceAccountName(name), clusterRoleBinding.Subjects[0].Name, "ClusterRoleBinding Subjects name did not match")
			asserts.Equalf(getInstallNamespace(), clusterRoleBinding.Subjects[0].Namespace, "ClusterRoleBinding Subjects namespace did not match")
			return nil
		})

	// Expect a call to get the Verrazzano system namespace (mock does not exist) and to create it
	expectVerrazzanoSystemNamespaceDoesNotExist(mock, asserts)

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
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

func makeVerrazzanoComponentStatusMap() vzapi.ComponentStatusMap {
	statusMap := make(vzapi.ComponentStatusMap)
	for _, comp := range registry.GetComponents() {
		if comp.IsOperatorInstallSupported() {
			statusMap[comp.Name()] = &vzapi.ComponentStatusDetails{
				Name: comp.Name(),
				Conditions: []vzapi.Condition{
					{
						Type:   vzapi.InstallComplete,
						Status: corev1.ConditionTrue,
					},
				},
				State: vzapi.Ready,
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
		Namespace: namespace,
		Name:      name,
		Labels:    labels}
	vzToUse.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: &vzapi.OCI{
			OCIConfigSecret:        "test-oci-config-secret",
			DNSZoneCompartmentOCID: "test-dns-zone-ocid",
			DNSZoneOCID:            "test-dns-zone-ocid",
			DNSZoneName:            "test-dns-zone-name",
		},
	}
	vzToUse.Status.Components = makeVerrazzanoComponentStatusMap()
	vzToUse.Status.State = vzapi.Ready

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

	// Expect a call to get the ServiceAccount - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: getInstallNamespace(), Name: buildServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "ServiceAccount"}, buildServiceAccountName(name)))

	// Expect a call to create the ServiceAccount - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, serviceAccount *corev1.ServiceAccount, opts ...client.CreateOption) error {
			asserts.Equalf(getInstallNamespace(), serviceAccount.Namespace, "ServiceAccount namespace did not match")
			asserts.Equalf(buildServiceAccountName(name), serviceAccount.Name, "ServiceAccount name did not match")
			asserts.Equalf(labels, serviceAccount.Labels, "ServiceAccount labels did not match")
			return nil
		})

	// Expect a call to get the ClusterRoleBinding - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: buildClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "ClusterRoleBinding"}, buildClusterRoleBindingName(namespace, name)))

	// Expect a call to create the ClusterRoleBinding - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, clusterRoleBinding *rbacv1.ClusterRoleBinding, opts ...client.CreateOption) error {
			asserts.Equalf("", clusterRoleBinding.Namespace, "ClusterRoleBinding namespace did not match")
			asserts.Equalf(buildClusterRoleBindingName(namespace, name), clusterRoleBinding.Name, "ClusterRoleBinding name did not match")
			asserts.Equalf(labels, clusterRoleBinding.Labels, "ClusterRoleBinding labels did not match")
			asserts.Equalf(buildServiceAccountName(name), clusterRoleBinding.Subjects[0].Name, "ClusterRoleBinding Subjects name did not match")
			asserts.Equalf(getInstallNamespace(), clusterRoleBinding.Subjects[0].Namespace, "ClusterRoleBinding Subjects namespace did not match")
			return nil
		})

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
	result, err := reconciler.Reconcile(request)

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

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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
				State:      vzapi.Ready,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.UninstallComplete,
					},
				},
			}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the uninstall Job - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: getInstallNamespace(), Name: buildUninstallJobName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, job *batchv1.Job) error {
			newJob := uninstalljob.NewJob(&uninstalljob.JobConfig{
				JobConfigCommon: k8s.JobConfigCommon{
					JobName:            name.Name,
					Namespace:          name.Namespace,
					Labels:             labels,
					ServiceAccountName: buildServiceAccountName(name.Name),
					JobImage:           "image",
					DryRun:             false,
				},
			})
			job.ObjectMeta = newJob.ObjectMeta
			job.Spec = newJob.Spec
			return nil
		})

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

	expectDeleteClusterRoleBinding(mock, getInstallNamespace(), name)
	expectDeleteServiceAccount(mock, getInstallNamespace(), name)
	expectDeleteNamespace(mock)

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

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

	verrazzanoToUse.Status.State = vzapi.Ready

	deleteTime := metav1.Time{
		Time: time.Now(),
	}

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
				Labels:            labels,
				DeletionTimestamp: &deleteTime,
				Finalizers:        []string{finalizerName}}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State: vzapi.Ready,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.UninstallStarted,
					},
				},
			}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the uninstall Job - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: getInstallNamespace(), Name: buildUninstallJobName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Job"}, buildUninstallJobName(name)))

	// Expect a call to create the uninstall Job - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, job *batchv1.Job, opts ...client.CreateOption) error {
			asserts.Equalf(getInstallNamespace(), job.Namespace, "Job namespace did not match")
			asserts.Equalf(buildUninstallJobName(name), job.Name, "Job name did not match")
			asserts.Equalf(labels, job.Labels, "Job labels did not match")
			return nil
		})

	// Expect a call to update the job - return success
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUninstallFailed tests the Reconcile method for the following use case
// GIVEN an uninstall job has failed
// WHEN a Verrazzano resource has been deleted
// THEN ensure the error is handled
func TestUninstallFailed(t *testing.T) {
	unitTesting = true
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	var verrazzanoToUse vzapi.Verrazzano

	deleteTime := metav1.Time{
		Time: time.Now(),
	}

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
				State: vzapi.Ready}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the uninstall Job - return that it exists and the job failed
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: getInstallNamespace(), Name: buildUninstallJobName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, job *batchv1.Job) error {
			newJob := uninstalljob.NewJob(&uninstalljob.JobConfig{
				JobConfigCommon: k8s.JobConfigCommon{
					JobName:            name.Name,
					Namespace:          name.Namespace,
					Labels:             labels,
					ServiceAccountName: buildServiceAccountName(name.Name),
					JobImage:           "image",
					DryRun:             false,
				},
			})
			job.ObjectMeta = newJob.ObjectMeta
			job.Spec = newJob.Spec
			job.Status = batchv1.JobStatus{
				Failed: 1,
			}
			return nil
		})

	// Expect a status update on the job
	mockStatus.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// Expect a call to update the finalizers - return success
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			return nil
		})

	expectDeleteClusterRoleBinding(mock, getInstallNamespace(), name)
	expectDeleteServiceAccount(mock, getInstallNamespace(), name)
	expectDeleteNamespace(mock)

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUninstallSucceeded tests the Reconcile method for the following use case
// GIVEN an uninstall job has succeeded
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
				State: vzapi.Ready}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the uninstall Job - return that it exists and the job succeeded
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: getInstallNamespace(), Name: buildUninstallJobName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, job *batchv1.Job) error {
			newJob := uninstalljob.NewJob(&uninstalljob.JobConfig{
				JobConfigCommon: k8s.JobConfigCommon{
					JobName:            name.Name,
					Namespace:          name.Namespace,
					Labels:             labels,
					ServiceAccountName: buildServiceAccountName(name.Name),
					JobImage:           "image",
					DryRun:             false,
				},
			})
			job.ObjectMeta = newJob.ObjectMeta
			job.Spec = newJob.Spec
			job.Status = batchv1.JobStatus{
				Succeeded: 1,
			}
			return nil
		})

	// Expect a status update on the job
	mockStatus.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// Expect a call to update the finalizers - return success
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			return nil
		})

	expectDeleteClusterRoleBinding(mock, getInstallNamespace(), name)
	expectDeleteServiceAccount(mock, getInstallNamespace(), name)
	expectDeleteNamespace(mock)

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

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
	result, err := reconciler.Reconcile(request)

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
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to get Verrazzano custom resource")
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestServiceAccountGetError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource
// WHEN a Verrazzano resource has been applied
// THEN return error if failure getting ServiceAccount
func TestServiceAccountGetError(t *testing.T) {
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
		Namespace: namespace,
		Name:      name,
		Labels:    labels}
	verrazzanoToUse.Status = vzapi.VerrazzanoStatus{
		State: vzapi.Ready}

	// Expect a call to get the Verrazzano resource.
	expectGetVerrazzanoExists(mock, verrazzanoToUse, namespace, name, labels)

	// Expect a call to get the ServiceAccount - return a failure error
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: getInstallNamespace(), Name: buildServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewBadRequest("failed to get ServiceAccount"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to get ServiceAccount")
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
}

// TestServiceAccountCreateError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource
// WHEN a there is a failure creating a ServiceAccount
// THEN return error
func TestServiceAccountCreateError(t *testing.T) {
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
		Namespace: namespace,
		Name:      name,
		Labels:    labels}
	verrazzanoToUse.Status = vzapi.VerrazzanoStatus{
		State: vzapi.Ready}

	// Expect a call to get the Verrazzano resource.
	expectGetVerrazzanoExists(mock, verrazzanoToUse, namespace, name, labels)

	// Expect a call to get the ServiceAccount - return not found
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: getInstallNamespace(), Name: buildServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "ServiceAccount"}, name))

	// Expect a call to create the ServiceAccount - return failure
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(errors.NewBadRequest("failed to create ServiceAccount"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to create ServiceAccount")
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
}

// TestClusterRoleBindingGetError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource
// WHEN a there is an error getting the ClusterRoleBinding
// THEN return error
func TestClusterRoleBindingGetError(t *testing.T) {
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
		Namespace: namespace,
		Name:      name,
		Labels:    labels}
	verrazzanoToUse.Status = vzapi.VerrazzanoStatus{
		State: vzapi.Ready}

	// Expect a call to get the Verrazzano resource.
	expectGetVerrazzanoExists(mock, verrazzanoToUse, namespace, name, labels)

	// Expect a call to get the ServiceAccount - return that it exists
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding - return a failure error
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: buildClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewBadRequest("failed to get ClusterRoleBinding"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to get ClusterRoleBinding")
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
}

// TestClusterRoleBindingCreateError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an Verrazzano resource
// WHEN a there is a failure creating a ClusterRoleBinding
// THEN return error
func TestClusterRoleBindingCreateError(t *testing.T) {
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
		Namespace: namespace,
		Name:      name,
		Labels:    labels}
	verrazzanoToUse.Status = vzapi.VerrazzanoStatus{
		State: vzapi.Ready}

	// Expect a call to get the Verrazzano resource.
	expectGetVerrazzanoExists(mock, verrazzanoToUse, namespace, name, labels)

	// Expect a call to get the ServiceAccount - return that it exists
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding - return not found
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: buildClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "ClusterRoleBinding"}, name))

	// Expect a call to create the ClusterRoleBinding - return failure
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(errors.NewBadRequest("failed to create ClusterRoleBinding"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to create ClusterRoleBinding")
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
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
		Namespace: namespace,
		Name:      name,
		Labels:    labels}
	verrazzanoToUse.Status = vzapi.VerrazzanoStatus{
		State: vzapi.Ready}
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
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, errMsg)
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
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
		Namespace: namespace,
		Name:      name,
		Labels:    labels}
	verrazzanoToUse.Status = vzapi.VerrazzanoStatus{
		State: vzapi.Ready}
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
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, errMsg)
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
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
		Namespace: namespace,
		Name:      name,
		Labels:    labels}
	verrazzanoToUse.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: &vzapi.OCI{
			OCIConfigSecret:        "test-oci-config-secret",
			DNSZoneCompartmentOCID: "test-dns-zone-ocid",
			DNSZoneOCID:            "test-dns-zone-ocid",
			DNSZoneName:            "test-dns-zone-name",
		},
	}
	verrazzanoToUse.Status = vzapi.VerrazzanoStatus{
		State: vzapi.Ready}
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
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to get Secret")
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
}

// TestBuildIngressIPForNIPNodePort tests buildDomain method
// GIVEN a request to buildDomain
// WHEN an nip.io configuration is detected and the service type is NodePort
// THEN the correct domain using 127.0.0.1 is returned
func TestBuildIngressIPForNIPNodePort(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Rancher ingress
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "ingress-nginx", Name: "ingress-controller-ingress-nginx-controller"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *corev1.Service) error {
			service.Spec.Type = corev1.ServiceTypeNodePort
			return nil
		})

	suffix, err := buildDomain(mock, &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	})
	assert.NoError(t, err)
	assert.Equal(t, "default.127.0.0.1.nip.io", suffix)

	// Validate the results
	mocker.Finish()
}

// TestBuildIngressIPForNIPLoadBalancer tests buildDomain method
// GIVEN a request to buildDomain
// WHEN an nip.io configuration is detected and the service type is LoadBalancer
// THEN the correct domain is returned
func TestBuildIngressIPForNIPLoadBalancer(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Rancher ingress
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "ingress-nginx", Name: "ingress-controller-ingress-nginx-controller"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *corev1.Service) error {
			service.Spec.Type = corev1.ServiceTypeLoadBalancer
			service.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
				{
					IP:       "11.22.33.44",
					Hostname: "myhost",
				},
			}
			return nil
		})

	suffix, err := buildDomain(mock, &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	})
	assert.NoError(t, err)
	assert.Equal(t, "default.11.22.33.44.nip.io", suffix)

	// Validate the results
	mocker.Finish()
}

// TestBuildIngressIPForNIPGetError tests buildDomain method
// GIVEN a request to buildDomain
// WHEN an nip.io configuration is detected and the client.Get() call returns an error
// THEN an error is returned
func TestBuildIngressIPForNIPGetError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Rancher ingress
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "ingress-nginx", Name: "ingress-controller-ingress-nginx-controller"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *corev1.Service) error {
			return fmt.Errorf("Simulated error")
		})

	suffix, err := buildDomain(mock, &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	})
	assert.Error(t, err)
	assert.Equal(t, "", suffix)

	// Validate the results
	mocker.Finish()
}

// TestBuildIngressIPForNIPInvalidServiceType tests buildDomain method
// GIVEN a request to buildDomain
// WHEN an nip.io configuration is detected with an invalid service type
// THEN an error is returned
func TestBuildIngressIPForNIPInvalidServiceType(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Rancher ingress
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "ingress-nginx", Name: "ingress-controller-ingress-nginx-controller"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *corev1.Service) error {
			service.Spec.Type = corev1.ServiceTypeClusterIP
			return nil
		})

	suffix, err := buildDomain(mock, &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	})
	assert.Error(t, err)
	assert.Equal(t, "", suffix)

	// Validate the results
	mocker.Finish()
}

// TestBuildIngressIPForNIPLoadBalancerOLCNE tests buildDomain method
// GIVEN a request to buildDomain
// WHEN an nip.io configuration is detected and the service IP is in the expected location for OLCNE
// THEN the correct domain is returned
func TestBuildIngressIPForNIPLoadBalancerOLCNE(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Rancher ingress
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "ingress-nginx", Name: "ingress-controller-ingress-nginx-controller"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *corev1.Service) error {
			service.Spec.Type = corev1.ServiceTypeLoadBalancer
			service.Spec.ExternalIPs = []string{
				"11.22.33.44",
			}
			return nil
		})

	suffix, err := buildDomain(mock, &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	})
	assert.NoError(t, err)
	assert.Equal(t, "default.11.22.33.44.nip.io", suffix)

	// Validate the results
	mocker.Finish()
}

// TestBuildIngressIPForNIPLoadBalancerOLCNENoIPFound tests buildDomain method
// GIVEN a request to buildDomain
// WHEN an nip.io configuration is detected no service IP is in the expected location for OLCNE
// THEN an error is returned
func TestBuildIngressIPForNIPLoadBalancerOLCNENoIPFound(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Rancher ingress
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "ingress-nginx", Name: "ingress-controller-ingress-nginx-controller"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, service *corev1.Service) error {
			service.Spec.Type = corev1.ServiceTypeLoadBalancer
			return nil
		})

	suffix, err := buildDomain(mock, &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	})
	assert.Error(t, err)
	assert.Equal(t, "", suffix)

	// Validate the results
	mocker.Finish()
}

// TestBuildOCIDNSDomain tests buildDomain method
// GIVEN a request to buildDomain
// WHEN an OCI DNS configuration is detected both with and without an environment name in the spec
// THEN the correct domain is returned
func TestBuildOCIDNSDomain(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	suffix, err := buildDomain(mock, &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{OCI: &vzapi.OCI{DNSZoneName: "my.zone.com"}},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "default.my.zone.com", suffix)

	suffix, err = buildDomain(mock, &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{OCI: &vzapi.OCI{DNSZoneName: "my.zone.com"}},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "myenv.my.zone.com", suffix)

	// Validate the results
	mocker.Finish()
}

// TestBuildExternalDNSDomain tests buildDomain method
// GIVEN a request to buildDomain
// WHEN an External DNS configuration is detected both with and without an environment name in the spec
// THEN the correct domain is returned
func TestBuildExternalDNSDomain(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	suffix, err := buildDomain(mock, &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{External: &vzapi.External{Suffix: "my.external.com"}},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "default.my.external.com", suffix)

	suffix, err = buildDomain(mock, &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{External: &vzapi.External{Suffix: "my.external.com"}},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "myenv.my.external.com", suffix)

	// Validate the results
	mocker.Finish()
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	//_ = clientgoscheme.AddToScheme(scheme)
	//_ = core.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
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
		Client: c,
		Scheme: scheme}
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

// expectVerrazzanoSystemNamespaceDoesNotExist expects a call to get the Verrazzano system namespace and returns
// that it does not exist, then expects a call to create it
func expectVerrazzanoSystemNamespaceDoesNotExist(mock *mocks.MockClient, asserts *assert.Assertions) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: constants.VerrazzanoSystemNamespace}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.ParseGroupResource("namespaces"), constants.VerrazzanoSystemNamespace))

	mock.EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Namespace{})).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.CreateOption) error {
			asserts.Equalf(constants.VerrazzanoSystemNamespace, ns.Name, "Verrazzano system namespace did not match")
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
		})
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
		})
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
		})
}

// expectDeleteServiceAccount expects a call to delete the service account used by install
func expectDeleteServiceAccount(mock *mocks.MockClient, namespace string, name string) {
	mock.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
}

// expectDeleteNamespace expects a call to delete the verrazzano-system ns
func expectDeleteNamespace(mock *mocks.MockClient) {
	mock.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
}

// expectDeleteClusterRoleBinding expects a call to delete the ClusterRoleBinding for the Verrazzano with the given
// namespace and name, and returns that it exists
func expectDeleteClusterRoleBinding(mock *mocks.MockClient, namespace string, name string) {
	mock.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	//	mock.EXPECT().Delete(gomock.Any(), types.NamespacedName{Namespace: "", Name: buildClusterRoleBindingName(namespace, name)}, gomock.Any()).Return(nil)
}

// Test_commonPath tests commonPath function
// GIVEN two file paths
// WHEN the commonPath function is called
// THEN commonPath func extracts common path or containing directory of two file paths
func Test_commonPath(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{
			name: "/var/log/containers",
			a:    "/var/log/containers/kube-flannel-ds-f64g8_kube-system_kube-flannel-88.log",
			b:    "/var/log/containers/csi-oci-node-6rpr5_kube-system_csi-node-driver-99.log",
			want: "/var/log/containers/",
		},
		{
			name: "/var/log/pods",
			a:    "/var/log/pods/kube-system_csi-oci-node-6rpr5_f69cf85b-x0x0-12345cd3fbd0/csi-node-driver/0.log",
			b:    "/var/log/pods/kube-system_kube-flannel-ds-f64g8_1ff336c7-y1y1-12a345c45e6c/kube-flannel/1.log",
			want: "/var/log/pods/",
		},
		{
			name: "/u01/data/docker/containers",
			a:    "/u01/data/docker/containers/82e/82e-json.log",
			b:    "/u01/data/docker/containers/92a/92a-json.log",
			want: "/u01/data/docker/containers/",
		},
		{
			name: "/u01/data/",
			a:    "/u01/data/",
			b:    "/u01/data/docker/containers/",
			want: "/u01/data/",
		},
		{
			name: "/u01/data/",
			a:    "/u01/data/docker/containers/",
			b:    "/u01/data/",
			want: "/u01/data/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commonPath(tt.a, tt.b); got != tt.want {
				t.Errorf("commonPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_dirsOutsideVarLog tests dirsOutsideVarLog function
// GIVEN a set of file paths
// WHEN the dirsOutsideVarLog function is called
// THEN dirsOutsideVarLog func collects containing directories of given file paths
func Test_dirsOutsideVarLog(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  []string
	}{
		{
			name: "Should not include /var/log",
			paths: []string{
				"/var/log/containers/podx_kube-system_pod-xx-88.log",
				"/var/log/pods/kube-system_pod-xx-6rpr5_f69cf85b-x0x0-12345cd3fbd0/pod-xx/0.log",
				"/var/log/containers/pody_kube-system_pod-yy-99.log",
				"/var/log/pods/kube-system_pod-yy-f64g8_1ff336c7-y1y1-12a345c45e6c/pod-yy/1.log",
			},
			want: []string{},
		},
		{
			name: "/u01/data/",
			paths: []string{
				"/var/log/containers/podx_kube-system_pod-xx-88.log",
				"/var/log/pods/kube-system_pod-xx-6rpr5_f69cf85b-x0x0-12345cd3fbd0/pod-xx/0.log",
				"/u01/data/docker/containers/82e/82e-json.log",
				"/var/log/containers/pody_kube-system_pod-yy-99.log",
				"/var/log/pods/kube-system_pod-yy-f64g8_1ff336c7-y1y1-12a345c45e6c/pod-yy/1.log",
				"/u01/data/docker/containers/92a/92a-json.log",
			},
			want: []string{"/u01/data/docker/containers/"},
		},
		{
			name: "multiple extra",
			paths: []string{
				"/var/log/containers/podx_kube-system_pod-xx-88.log",
				"/u0/x/pods/kube-system_pod-xx-6rpr5_f69cf85b-x0x0-12345cd3fbd0/pod-xx/0.log",
				"/u0/y/containers/82e/82e-json.log",
				"/u01/data/containers/82e/82e-json.log",
				"/var/log/containers/pody_kube-system_pod-yy-99.log",
				"/u0/x/pods/kube-system_pod-yy-f64g8_1ff336c7-y1y1-12a345c45e6c/pod-yy/1.log",
				"/u0/y/containers/92a/92a-json.log",
				"/u01/data/containers/92a/92a-json.log",
			},
			want: []string{"/u0/", "/u01/data/containers/"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dirsOutsideVarLog(tt.paths); !equalStringSet(got, tt.want) {
				t.Errorf("commonPaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_isParentDir tests isParentDir function
// GIVEN two file paths
// WHEN the isParentDir function is called
// THEN isParentDir func return true if the dir is container directory of the path
func Test_isParentDir(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "/u01/data/containers",
			path: "/u01/data/containers/",
			want: true,
		}, {
			name: "/u01/data/containers/",
			path: "/u01/data/containers/",
			want: true,
		}, {
			name: "/u01/data/cont",
			path: "/u01/data/containers/",
			want: false,
		}, {
			name: "/u01/da",
			path: "/u01/data",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isParentDir(tt.path, tt.name); got != tt.want {
				t.Errorf("isParentDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_addFluentdExtraVolumeMounts tests addFluentdExtraVolumeMounts function
// GIVEN a Verrazzano and a set of file paths
// WHEN the addFluentdExtraVolumeMounts function is called
// THEN extra volume mounts are added to the fluentd component
func Test_addFluentdExtraVolumeMounts(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		vz    *vzapi.Verrazzano
		want  []string
	}{
		{
			name: "/u01/data/containers/",
			files: []string{
				"/var/log/containers/podx_kube-system_pod-xx-88.log",
				"/u0/x/pods/kube-system_pod-xx-6rpr5_f69cf85b-x0x0-12345cd3fbd0/pod-xx/0.log",
				"/u0/y/containers/82e/82e-json.log",
				"/u01/data/containers/82e/82e-json.log",
				"/var/log/containers/pody_kube-system_pod-yy-99.log",
				"/u0/x/pods/kube-system_pod-yy-f64g8_1ff336c7-y1y1-12a345c45e6c/pod-yy/1.log",
				"/u0/y/containers/92a/92a-json.log",
				"/u01/data/containers/92a/92a-json.log",
			},
			vz:   &vzapi.Verrazzano{},
			want: []string{"/u0/", "/u01/data/containers/"},
		}, {
			name: "/u01/data",
			files: []string{
				"/var/log/containers/podx_kube-system_pod-xx-88.log",
				"/u0/x/pods/kube-system_pod-xx-6rpr5_f69cf85b-x0x0-12345cd3fbd0/pod-xx/0.log",
				"/u0/y/containers/82e/82e-json.log",
				"/u01/data/containers/82e/82e-json.log",
				"/var/log/containers/pody_kube-system_pod-yy-99.log",
				"/u0/x/pods/kube-system_pod-yy-f64g8_1ff336c7-y1y1-12a345c45e6c/pod-yy/1.log",
				"/u0/y/containers/92a/92a-json.log",
				"/u01/data/containers/92a/92a-json.log",
			},
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					ExtraVolumeMounts: []vzapi.VolumeMount{{
						Source: "/u01/data",
					}},
				}}},
			},
			want: []string{"/u0/", "/u01/data"},
		}, {
			name: "/u01/",
			files: []string{
				"/var/log/containers/podx_kube-system_pod-xx-88.log",
				"/u0/x/pods/kube-system_pod-xx-6rpr5_f69cf85b-x0x0-12345cd3fbd0/pod-xx/0.log",
				"/u0/y/containers/82e/82e-json.log",
				"/u01/data/containers/82e/82e-json.log",
				"/var/log/containers/pody_kube-system_pod-yy-99.log",
				"/u0/x/pods/kube-system_pod-yy-f64g8_1ff336c7-y1y1-12a345c45e6c/pod-yy/1.log",
				"/u0/y/containers/92a/92a-json.log",
				"/u01/data/containers/92a/92a-json.log",
			},
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					ExtraVolumeMounts: []vzapi.VolumeMount{{
						Source: "/u0/x",
					}, {
						Source: "/u01/",
					}},
				}}},
			},
			want: []string{"/u0/", "/u01/", "/u0/x"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := collectVolumeMounts(addFluentdExtraVolumeMounts(tt.files, tt.vz)); !equalStringSet(got, tt.want) {
				t.Errorf("addFluentdExtraVolumeMounts() = %v, want %v", got, tt.want)
			}
		})
	}
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

func collectVolumeMounts(vz *vzapi.Verrazzano) []string {
	var vms []string
	for _, vm := range vz.Spec.Components.Fluentd.ExtraVolumeMounts {
		vms = append(vms, vm.Source)
	}
	return vms
}

func equalStringSet(x, y []string) bool {
	if len(x) != len(y) {
		return false
	}
	for _, a := range x {
		found := false
		for _, b := range y {
			if a == b {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
