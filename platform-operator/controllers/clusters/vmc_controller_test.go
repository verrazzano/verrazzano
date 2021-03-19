// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"testing"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	k8net "k8s.io/api/networking/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/rest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	clustersapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const apiVersion = "clusters.verrazzano.io/v1alpha1"
const kind = "VerrazzanoManagedCluster"

const testServerData = "https://testurl"

const (
	token              = "tokenData"
	managedClusterData = "cluster1"
)

type AssertFn func(configMap *corev1.ConfigMap) error

// TestCreateVMC tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied
// THEN ensure all the objects are created
func TestCreateVMC(t *testing.T) {
	namespace := constants.VerrazzanoMultiClusterNamespace
	name := "test"
	promData := "prometheus:\n" +
		"  authpasswd: nRXlxXgMwN\n" +
		"  host: prometheus.vmi.system.default.152.67.141.181.xip.io\n" +
		"  cacrt: |\n" +
		"    -----BEGIN CERTIFICATE-----\n" +
		"    MIIBiDCCAS6gAwIBAgIBADAKBggqhkjOPQQDAjA7MRwwGgYDVQQKExNkeW5hbWlj\n" +
		"    -----END CERTIFICATE-----"
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, name)
	expectSyncServiceAccount(t, mock, name)
	expectSyncRoleBinding(t, mock, name)
	expectSyncAgent(t, mock, name)
	expectSyncRegistration(t, mock, name)
	expectSyncElasticsearch(t, mock, name)
	expectSyncManifest(t, mock, name)
	expectSyncPrometheusScraper(mock, name, "", promData, func(configMap *corev1.ConfigMap) error {
		asserts.Len(configMap.Data, 2, "no data found")
		asserts.NotEmpty(configMap.Data["ca-test"], "No cert entry found")
		prometheusYaml := configMap.Data["prometheus.yml"]

		scrapeConfig, err := getScrapeConfig(prometheusYaml, name)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")
		asserts.Equal("prometheus.vmi.system.default.152.67.141.181.xip.io",
			scrapeConfig.Search("static_configs", "0", "targets", "0").Data(), "No host entry found")
		asserts.NotEmpty(scrapeConfig.Search("basic_auth", "password").Data(), "No password")
		asserts.Equal(prometheusConfigBasePath+"ca-test",
			scrapeConfig.Search("tls_config", "ca_file").Data(), "Wrong cert path")
		return nil
	})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestCreateVMC tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource for an OCI DNS cluster
// WHEN a VerrazzanoManagedCluster resource has been applied
// THEN ensure all the objects are created
func TestCreateVMCOCIDNS(t *testing.T) {
	namespace := "verrazzano-mc"
	name := "test"
	// OCI DNS cluster does not include a CA cert since the CA is public
	promData := "prometheus:\n" +
		"  authpasswd: nRXlxXgMwN\n" +
		"  host: prometheus.vmi.system.default.152.67.141.181.xip.io\n"
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, name)
	expectSyncServiceAccount(t, mock, name)
	expectSyncRoleBinding(t, mock, name)
	expectSyncAgent(t, mock, name)
	expectSyncRegistration(t, mock, name)
	expectSyncElasticsearch(t, mock, name)
	expectSyncManifest(t, mock, name)
	expectSyncPrometheusScraper(mock, name, "", promData, func(configMap *corev1.ConfigMap) error {
		asserts.Len(configMap.Data, 2, "no data found")
		asserts.Empty(configMap.Data["ca-test"], "Cert entry found")
		prometheusYaml := configMap.Data["prometheus.yml"]
		scrapeConfig, err := getScrapeConfig(prometheusYaml, name)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")
		asserts.Equal("prometheus.vmi.system.default.152.67.141.181.xip.io",
			scrapeConfig.Search("static_configs", "0", "targets", "0").Data(), "No host entry found")
		asserts.NotEmpty(scrapeConfig.Search("basic_auth", "password").Data(), "No password")
		asserts.Empty(scrapeConfig.Search("tls_config", "ca_file").Data(), "Wrong cert path")

		return nil
	})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestCreateVMCWithExistingScrapeConfiguration tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied and prometheus is already configured with a scrape config for the cluster
// THEN ensure all the objects are created
func TestCreateVMCWithExistingScrapeConfiguration(t *testing.T) {
	namespace := "verrazzano-mc"
	name := "test"
	promData := "prometheus:\n" +
		"  authpasswd: nRXlxXgMwN\n" +
		"  host: prometheus.vmi.system.default.152.67.141.181.xip.io\n" +
		"  cacrt: |\n" +
		"    -----BEGIN CERTIFICATE-----\n" +
		"    MIIBiDCCAS6gAwIBAgIBADAKBggqhkjOPQQDAjA7MRwwGgYDVQQKExNkeW5hbWlj\n" +
		"    -----END CERTIFICATE-----"
	prometheusYaml := `global:
  scrape_interval: 20s
  scrape_timeout: 10s
  evaluation_interval: 30s
scrape_configs:
- job_name: cluster1
  scrape_interval: 20s
  scrape_timeout: 15s
  scheme: http`
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, name)
	expectSyncServiceAccount(t, mock, name)
	expectSyncRoleBinding(t, mock, name)
	expectSyncAgent(t, mock, name)
	expectSyncRegistration(t, mock, name)
	expectSyncElasticsearch(t, mock, name)
	expectSyncManifest(t, mock, name)
	expectSyncPrometheusScraper(mock, name, prometheusYaml, promData, func(configMap *corev1.ConfigMap) error {
		// check for the modified entry
		asserts.Len(configMap.Data, 2, "no data found")
		asserts.NotEmpty(configMap.Data["ca-test"], "No cert entry found")
		prometheusYaml := configMap.Data["prometheus.yml"]
		scrapeConfig, err := getScrapeConfig(prometheusYaml, name)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")
		asserts.Equal("prometheus.vmi.system.default.152.67.141.181.xip.io",
			scrapeConfig.Search("static_configs", "0", "targets", "0").Data(), "No host entry found")
		asserts.NotEmpty(scrapeConfig.Search("basic_auth", "password").Data(), "No password")
		asserts.Equal(prometheusConfigBasePath+"ca-test",
			scrapeConfig.Search("tls_config", "ca_file").Data(), "Wrong cert path")

		return nil
	})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestReplaceExistingScrapeConfiguration tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied and prometheus is already configured with a scrape configuration for the same cluster
// THEN ensure all the objects are created (existing configuration is replaced)
func TestReplaceExistingScrapeConfiguration(t *testing.T) {
	namespace := "verrazzano-mc"
	name := "test"
	promData := "prometheus:\n" +
		"  authpasswd: nRXlxXgMwN\n" +
		"  host: prometheus.vmi.system.default.152.67.141.181.xip.io\n" +
		"  cacrt: |\n" +
		"    -----BEGIN CERTIFICATE-----\n" +
		"    MIIBiDCCAS6gAwIBAgIBADAKBggqhkjOPQQDAjA7MRwwGgYDVQQKExNkeW5hbWlj\n" +
		"    -----END CERTIFICATE-----"
	prometheusYaml := `global:
  scrape_interval: 20s
  scrape_timeout: 10s
  evaluation_interval: 30s
scrape_configs:
- job_name: test
  scrape_interval: 20s
  scrape_timeout: 15s
  scheme: http`
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, name)
	expectSyncServiceAccount(t, mock, name)
	expectSyncRoleBinding(t, mock, name)
	expectSyncAgent(t, mock, name)
	expectSyncRegistration(t, mock, name)
	expectSyncElasticsearch(t, mock, name)
	expectSyncManifest(t, mock, name)
	expectSyncPrometheusScraper(mock, name, prometheusYaml, promData, func(configMap *corev1.ConfigMap) error {
		asserts.Len(configMap.Data, 2, "no data found")
		asserts.NotNil(configMap.Data["ca-test"], "No cert entry found")
		prometheusYaml := configMap.Data["prometheus.yml"]
		scrapeConfig, err := getScrapeConfig(prometheusYaml, name)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")
		asserts.Equal("test", scrapeConfig.Path("job_name").Data(), "wrong job name")
		asserts.Equal("prometheus.vmi.system.default.152.67.141.181.xip.io",
			scrapeConfig.Search("static_configs", "0", "targets", "0").Data(), "No host entry found")
		asserts.NotEmpty(scrapeConfig.Search("basic_auth", "password").Data(), "No password")
		asserts.Equal(prometheusConfigBasePath+"ca-test",
			scrapeConfig.Search("tls_config", "ca_file").Data(), "Wrong cert path")
		asserts.Equal("https", scrapeConfig.Path("scheme").Data(), "wrong scheme")
		return nil
	})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestDeleteVMC tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been deleted
// THEN ensure the object is not processed
func TestDeleteVMC(t *testing.T) {
	namespace := "verrazzano-install"
	name := "test"
	labels := map[string]string{"label1": "test"}
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the VerrazzanoManagedCluster resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, vmc *clustersapi.VerrazzanoManagedCluster) error {
			vmc.TypeMeta = metav1.TypeMeta{
				APIVersion: apiVersion,
				Kind:       kind}
			vmc.ObjectMeta = metav1.ObjectMeta{
				Namespace:         name.Namespace,
				Name:              name.Name,
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
				Labels:            labels,
				Finalizers:        []string{finalizerName}}
			return nil
		})

	// Expect a call to get the prometheus configmap and return one with two entries, including this cluster
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "verrazzano-system", Name: "vmi-system-prometheus-config"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			// setup a scaled down existing scrape config entry for cluster1
			configMap.Data = map[string]string{
				"prometheus.yml": `global:
  scrape_interval: 20s
  scrape_timeout: 10s
  evaluation_interval: 30s
scrape_configs:
- job_name: test
  scrape_interval: 20s
  scrape_timeout: 15s
  scheme: http
- job_name: test2
  scrape_interval: 20s
  scrape_timeout: 15s
  scheme: http`,
				"ca-test": "-----BEGIN CERTIFICATE-----\n" +
					"    MIIBiDCCAS6gAwIBAgIBADAKBggqhkjOPQQDAjA7MRwwGgYDVQQKExNkeW5hbWlj\n" +
					"    -----END CERTIFICATE-----",
			}

			return nil
		})

	// Expect a call to Update the configmap
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.UpdateOption) error {
			// check for the modified entry
			asserts.Len(configMap.Data, 1, "no data found")
			asserts.Empty(configMap.Data["ca-test"], "cert entry found")
			prometheusYaml := configMap.Data["prometheus.yml"]
			scrapeConfig, err := getScrapeConfig(prometheusYaml, "test2")
			if err != nil {
				asserts.Fail("failed due to error %v", err)
			}

			// Expect a call to update the VerrazzanoManagedCluster finalizer
			mock.EXPECT().
				Update(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
					asserts.True(len(vmc.ObjectMeta.Finalizers) == 0, "Wrong number of finalizers")
					return nil
				})

			asserts.NotNil(prometheusYaml, "No prometheus config yaml found")
			asserts.NotNil(scrapeConfig, "No scrape configs found")
			asserts.Equal("test2", scrapeConfig.Path("job_name").Data(), "Expected scrape config not found")

			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clustersapi.AddToScheme(scheme)
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

// newVMCReconciler creates a new reconciler for testing
// c - The Kerberos client to inject into the reconciler
func newVMCReconciler(c client.Client) VerrazzanoManagedClusterReconciler {
	scheme := newScheme()
	reconciler := VerrazzanoManagedClusterReconciler{
		Client: c,
		Scheme: scheme}
	return reconciler
}

func fakeGetConfig() (*rest.Config, error) {
	conf := rest.Config{
		TLSClientConfig: rest.TLSClientConfig{
			CAData: []byte("fakeCA"),
		},
	}
	return &conf, nil
}

// Expect syncRoleBinding related calls
func expectSyncRoleBinding(t *testing.T, mock *mocks.MockClient, name string) {
	asserts := assert.New(t)

	// Expect a call to get the ClusterRoleBinding - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: generateManagedResourceName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "ServiceAccount"}, generateManagedResourceName(name)))

	// Expect a call to create the ClusterRoleBinding - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, binding *rbacv1.ClusterRoleBinding, opts ...client.CreateOption) error {
			asserts.Equalf(generateManagedResourceName(name), binding.Name, "ClusterRoleBinding name did not match")
			asserts.Equalf(constants.MCClusterRole, binding.RoleRef.Name, "ClusterRoleBinding roleref did not match")
			asserts.Equalf(generateManagedResourceName(name), binding.Subjects[0].Name, "Subject did not match")
			asserts.Equalf(constants.VerrazzanoMultiClusterNamespace, binding.Subjects[0].Namespace, "Subject namespace did not match")
			return nil
		})
}

// Expect syncServiceAccount related calls
func expectSyncServiceAccount(t *testing.T, mock *mocks.MockClient, name string) {
	asserts := assert.New(t)

	// Expect a call to get the ServiceAccount - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: generateManagedResourceName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "ServiceAccount"}, generateManagedResourceName(name)))

	// Expect a call to create the ServiceAccount - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, serviceAccount *corev1.ServiceAccount, opts ...client.CreateOption) error {
			asserts.Equalf(constants.VerrazzanoMultiClusterNamespace, serviceAccount.Namespace, "ServiceAccount namespace did not match")
			asserts.Equalf(generateManagedResourceName(name), serviceAccount.Name, "ServiceAccount name did not match")
			return nil
		})

	// Expect a call to update the VerrazzanoManagedCluster service account name - return success
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(vmc.Spec.ServiceAccount, generateManagedResourceName(name), "ServiceAccount name did not match")
			return nil
		})
}

// Expect syncAgent related calls
func expectSyncAgent(t *testing.T, mock *mocks.MockClient, name string) {
	saSecretName := "saSecret"

	// Expect a call to get the ServiceAccount, return one with the secret name set
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: generateManagedResourceName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, sa *corev1.ServiceAccount) error {
			sa.Secrets = []corev1.ObjectReference{{
				Name: saSecretName,
			}}
			return nil
		})

	// Expect a call to get the service token secret, return the secret with the token
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: saSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				TokenKey: []byte(token),
			}
			return nil
		})

	// Expect a call to get the verrazzano-admin-cluster configmap
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: constants.AdminClusterConfigMapName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, cm *corev1.ConfigMap) error {
			cm.Data = map[string]string{
				constants.ServerDataKey: testServerData,
			}
			return nil
		})

	// Expect a call to get the Agent secret - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetAgentSecretName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoMultiClusterNamespace, Resource: "Secret"}, GetAgentSecretName(name)))

	// Expect a call to create the Agent secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			return nil
		})
}

// Expect syncRegistration related calls
func expectSyncRegistration(t *testing.T, mock *mocks.MockClient, name string) {
	// Expect a call to get the registration secret - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetRegistrationSecretName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoMultiClusterNamespace, Resource: "Secret"}, GetRegistrationSecretName(name)))

	// Expect a call to create the registration secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			secret.Data = map[string][]byte{
				ManagedClusterNameKey: []byte(managedClusterData),
			}
			return nil
		})
}

// Expect syncElasticSearch related calls
func expectSyncElasticsearch(t *testing.T, mock *mocks.MockClient, name string) {
	asserts := assert.New(t)
	caData := "ca"
	userData := "user"
	passwordData := "pw"
	hostdata := "testhost"
	urlData := "https://testhost:443"

	// Expect a call to get the tls ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: vmiIngest}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *k8net.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			ingress.Spec.Rules = []k8net.IngressRule{{
				Host: hostdata,
			}}
			return nil
		})

	// Expect a call to get the Verrazzano secret, return the secret with the fields set
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.Verrazzano}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				UsernameKey: []byte(userData),
				PasswordKey: []byte(passwordData),
			}
			return nil
		})

	// Expect a call to get the system-tls secret, return the secret with the fields set
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.SystemTLS}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				CaCrtKey: []byte(caData),
			}
			return nil
		})

	// Expect a call to get the Elasticsearch secret - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetElasticsearchSecretName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoMultiClusterNamespace, Resource: "Secret"}, GetElasticsearchSecretName(name)))

	// Expect a call to create the Elasticsearch secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			ca := secret.Data[CaBundleKey]
			asserts.Equalf(caData, string(ca), "Incorrect cadata in Elasticsearch secret ")
			user := secret.Data[UsernameKey]
			asserts.Equalf(userData, string(user), "Incorrect user in Elasticsearch secret ")
			pw := secret.Data[PasswordKey]
			asserts.Equalf(passwordData, string(pw), "Incorrect password in Elasticsearch secret ")
			url := secret.Data[URLKey]
			asserts.Equalf(urlData, string(url), "Incorrect URL in Elasticsearch secret ")
			return nil
		})
}

// Expect syncManifest related calls
func expectSyncManifest(t *testing.T, mock *mocks.MockClient, name string) {
	asserts := assert.New(t)
	clusterName := "cluster1"
	caData := "ca"
	userData := "user"
	passwordData := "pw"
	kubeconfigData := "fakekubeconfig"
	urlData := "https://testhost:443"

	// Expect a call to get the Agent secret
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetAgentSecretName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				KubeconfigKey: []byte(kubeconfigData),
			}
			return nil
		})

	// Expect a call to get the registration secret
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetRegistrationSecretName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				ManagedClusterNameKey: []byte(clusterName),
			}
			return nil
		})

	// Expect a call to get the Elasticsearch secret
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetElasticsearchSecretName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				CaCrtKey:    []byte(caData),
				UsernameKey: []byte(userData),
				PasswordKey: []byte(passwordData),
				URLKey:      []byte(urlData),
			}
			return nil
		})

	// Expect a call to get the manifest secret - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetManifestSecretName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoMultiClusterNamespace, Resource: "Secret"}, GetManifestSecretName(name)))

	// Expect a call to create the manifest secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			data := secret.Data[YamlKey]
			asserts.NotZero(len(data), "Expected yaml data in manifest secret")
			return nil
		})

	// Expect a call to update the VerrazzanoManagedCluster kubeconfig secret name - return success
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(vmc.Spec.ManagedClusterManifestSecret, GetManifestSecretName(name), "Manifest secret name did not match")
			return nil
		})
}

func expectVmcGetAndUpdate(t *testing.T, mock *mocks.MockClient, name string) {
	asserts := assert.New(t)
	labels := map[string]string{"label1": "test"}

	// Expect a call to get the VerrazzanoManagedCluster resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, vmc *clustersapi.VerrazzanoManagedCluster) error {
			vmc.TypeMeta = metav1.TypeMeta{
				APIVersion: apiVersion,
				Kind:       kind}
			vmc.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    labels}
			vmc.Spec = clustersapi.VerrazzanoManagedClusterSpec{
				PrometheusSecret: getPrometheusSecretName(),
			}
			return nil
		})

	// Expect a call to update the VerrazzanoManagedCluster finalizer
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.True(len(vmc.ObjectMeta.Finalizers) == 1, "Wrong number of finalizers")
			asserts.Equal(finalizerName, vmc.ObjectMeta.Finalizers[0], "wrong finalizer")
			return nil
		})

}

func expectSyncPrometheusScraper(mock *mocks.MockClient, vmcName string, prometheusYaml string, prometheusSecretData string, f AssertFn) {
	// Expect a call to get the prometheus secret - return it
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: getPrometheusSecretName()}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				getClusterYamlKey(vmcName): []byte(prometheusSecretData),
			}
			return nil
		})

	// Expect a call to get the prometheus configmap and return one with an existing entry
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-prometheus-config"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			// setup a scaled down existing scrape config entry for cluster1
			configMap.Data = map[string]string{
				"prometheus.yml": prometheusYaml,
			}
			return nil
		})

	mock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.UpdateOption) error {
			return f(configMap)
		})

}

// getScrapeConfig gets a representation of the vmc scrape configuration from the provided yaml
func getScrapeConfig(prometheusYaml string, name string) (*gabs.Container, error) {
	cfg, err := parsePrometheusConfig(prometheusYaml)
	if err != nil {
		return nil, err
	}
	scrapeConfigs := cfg.Path(scrapeConfigsKey).Children()
	var scrapeConfig *gabs.Container
	for _, scrapeConfig = range scrapeConfigs {
		jobName := scrapeConfig.Search(jobNameKey).Data()
		if jobName == name {
			break
		}
	}
	return scrapeConfig, nil
}

// getPrometheusSecretName returns the prometheus secret name
func getPrometheusSecretName() string {
	return "test-secret"
}
