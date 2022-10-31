// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	"github.com/verrazzano/verrazzano/pkg/metricsutils"
	"github.com/verrazzano/verrazzano/pkg/test/mockmatchers"
	clustersapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconstants "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const apiVersion = "clusters.verrazzano.io/v1alpha1"
const kind = "VerrazzanoManagedCluster"

const (
	token                    = "tokenData"
	testManagedCluster       = "test"
	rancherAgentRegistry     = "ghcr.io"
	rancherAgentImage        = rancherAgentRegistry + "/verrazzano/rancher-agent:v1.0.0"
	unitTestRancherClusterID = "unit-test-rancher-cluster-id"
)

var (
	loginURLParts    = strings.Split(loginPath, "?")
	loginURIPath     = loginURLParts[0]
	loginQueryString = loginURLParts[1]
)

const rancherManifestYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cattle-cluster-agent
  namespace: cattle-system
spec:
  template:
    spec:
	  containers:
	    - name: cluster-register
		  image: ` + rancherAgentImage + `
		  imagePullPolicy: IfNotPresent
`

type AssertFn func(configMap *corev1.ConfigMap) error
type secretAssertFn func(secret *corev1.Secret) error

// TestCreateVMC tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied in a cluster where Rancher is enabled
// THEN ensure all the objects are created correctly (USES feature flag rancherBasedKubeconfigEnabled)
func TestCreateVMCRancherEnabled(t *testing.T) {
	// with feature flag disabled (which triggers different asserts/mocks from enabled)
	doTestCreateVMC(t, true)

	// with feature flag enabled
	rancherBasedKubeConfigEnabled = true
	defer func() { rancherBasedKubeConfigEnabled = false }()
	doTestCreateVMC(t, true)
}

// TestCreateVMC tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied in a cluster where Rancher is DISABLED
// THEN ensure all the objects are created correctly
func TestCreateVMCRancherDisabled(t *testing.T) {
	doTestCreateVMC(t, false)
}

func doTestCreateVMC(t *testing.T, rancherEnabled bool) {
	namespace := constants.VerrazzanoMultiClusterNamespace
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	mockRequestSender := mocks.NewMockRequestSender(mocker)
	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = mockRequestSender

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	caSecretExistsInVMC := true
	expectVmcGetAndUpdate(t, mock, testManagedCluster, caSecretExistsInVMC, false)
	expectSyncServiceAccount(t, mock, testManagedCluster, true)
	expectSyncRoleBinding(t, mock, testManagedCluster, true)
	// Agent secret sync checks depend on whether Rancher is enabled
	expectSyncAgent(t, mock, testManagedCluster, rancherEnabled, false)
	expectSyncRegistration(t, mock, testManagedCluster, false)
	expectSyncManifest(t, mock, mockStatus, mockRequestSender, testManagedCluster, false, rancherManifestYAML)
	expectSyncCACertRancherK8sCalls(t, mock, mockRequestSender, false)
	expectSyncPrometheusScraper(mock, testManagedCluster, "", "", true, getCaCrt(), func(configMap *corev1.ConfigMap) error {
		asserts.Len(configMap.Data, 2, "no data found")
		asserts.NotEmpty(configMap.Data["ca-test"], "No cert entry found")
		prometheusYaml := configMap.Data["prometheus.yml"]
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")

		scrapeConfig, err := getScrapeConfig(prometheusYaml, testManagedCluster)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		validateScrapeConfig(t, scrapeConfig, prometheusConfigBasePath, true)
		return nil
	}, func(secret *corev1.Secret) error {
		scrapeConfigYaml := secret.Data[constants.PromAdditionalScrapeConfigsSecretKey]
		scrapeConfigs, err := metricsutils.ParseScrapeConfig(string(scrapeConfigYaml))
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		scrapeConfig := getJob(scrapeConfigs.Children(), testManagedCluster)
		validateScrapeConfig(t, scrapeConfig, managedCertsBasePath, true)
		return nil
	}, func(secret *corev1.Secret) error {
		asserts.NotEmpty(secret.Data["ca-test"], "Expected to find a managed cluster TLS cert")
		return nil
	})

	// expect status updated with condition Ready=true
	expectStatusUpdateReadyCondition(asserts, mock, mockStatus, corev1.ConditionTrue, "", false)

	// Create and make the request
	request := newRequest(namespace, testManagedCluster)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(vpoconstants.ReconcileLoopRequeueInterval), result.RequeueAfter)
}

// TestCreateVMC tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied on a Verrazzano install configured with external ES
// THEN ensure all the objects are created
func TestCreateVMCWithExternalES(t *testing.T) {
	namespace := constants.VerrazzanoMultiClusterNamespace
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	mockRequestSender := mocks.NewMockRequestSender(mocker)
	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = mockRequestSender

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, testManagedCluster, true, false)
	expectSyncServiceAccount(t, mock, testManagedCluster, true)
	expectSyncRoleBinding(t, mock, testManagedCluster, true)
	expectSyncAgent(t, mock, testManagedCluster, false, true)
	expectSyncRegistration(t, mock, testManagedCluster, true)
	expectSyncManifest(t, mock, mockStatus, mockRequestSender, testManagedCluster, false, rancherManifestYAML)
	expectSyncCACertRancherK8sCalls(t, mock, mockRequestSender, false)
	expectSyncPrometheusScraper(mock, testManagedCluster, "", "", true, getCaCrt(), func(configMap *corev1.ConfigMap) error {
		asserts.Len(configMap.Data, 2, "no data found")
		asserts.NotEmpty(configMap.Data["ca-test"], "No cert entry found")
		prometheusYaml := configMap.Data["prometheus.yml"]
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")

		scrapeConfig, err := getScrapeConfig(prometheusYaml, testManagedCluster)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		validateScrapeConfig(t, scrapeConfig, prometheusConfigBasePath, true)
		return nil
	}, func(secret *corev1.Secret) error {
		scrapeConfigYaml := secret.Data[constants.PromAdditionalScrapeConfigsSecretKey]
		scrapeConfigs, err := metricsutils.ParseScrapeConfig(string(scrapeConfigYaml))
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		scrapeConfig := getJob(scrapeConfigs.Children(), testManagedCluster)
		validateScrapeConfig(t, scrapeConfig, managedCertsBasePath, true)
		return nil
	}, func(secret *corev1.Secret) error {
		asserts.NotEmpty(secret.Data["ca-test"], "Expected to find a managed cluster TLS cert")
		return nil
	})

	// expect status updated with condition Ready=true
	expectStatusUpdateReadyCondition(asserts, mock, mockStatus, corev1.ConditionTrue, "", false)

	// Create and make the request
	request := newRequest(namespace, testManagedCluster)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(vpoconstants.ReconcileLoopRequeueInterval), result.RequeueAfter)
}

// TestCreateVMC tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource for an OCI DNS cluster
// WHEN a VerrazzanoManagedCluster resource has been applied
// THEN ensure all the objects are created
func TestCreateVMCOCIDNS(t *testing.T) {
	namespace := "verrazzano-mc"
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	mockRequestSender := mocks.NewMockRequestSender(mocker)
	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = mockRequestSender

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, testManagedCluster, true, false)
	expectSyncServiceAccount(t, mock, testManagedCluster, true)
	expectSyncRoleBinding(t, mock, testManagedCluster, true)
	expectSyncAgent(t, mock, testManagedCluster, false, true)
	expectSyncRegistration(t, mock, testManagedCluster, false)
	expectSyncManifest(t, mock, mockStatus, mockRequestSender, testManagedCluster, false, rancherManifestYAML)
	expectSyncCACertRancherK8sCalls(t, mock, mockRequestSender, false)
	expectSyncPrometheusScraper(mock, testManagedCluster, "", "", true, "", func(configMap *corev1.ConfigMap) error {
		asserts.Len(configMap.Data, 2, "no data found")
		asserts.Empty(configMap.Data["ca-test"], "Cert entry found")
		prometheusYaml := configMap.Data["prometheus.yml"]
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")

		scrapeConfig, err := getScrapeConfig(prometheusYaml, testManagedCluster)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		validateScrapeConfig(t, scrapeConfig, prometheusConfigBasePath, false)
		return nil
	}, func(secret *corev1.Secret) error {
		scrapeConfigYaml := secret.Data[constants.PromAdditionalScrapeConfigsSecretKey]
		scrapeConfigs, err := metricsutils.ParseScrapeConfig(string(scrapeConfigYaml))
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		scrapeConfig := getJob(scrapeConfigs.Children(), testManagedCluster)
		validateScrapeConfig(t, scrapeConfig, managedCertsBasePath, false)
		return nil
	}, func(secret *corev1.Secret) error {
		asserts.Empty(secret.Data["ca-test"])
		return nil
	})

	// expect status updated with condition Ready=true
	expectStatusUpdateReadyCondition(asserts, mock, mockStatus, corev1.ConditionTrue, "", false)

	// Create and make the request
	request := newRequest(namespace, testManagedCluster)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(vpoconstants.ReconcileLoopRequeueInterval), result.RequeueAfter)
}

// TestCreateVMCNoCACert tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied with no CA Cert
// THEN ensure all the objects are created
func TestCreateVMCNoCACert(t *testing.T) {
	namespace := constants.VerrazzanoMultiClusterNamespace
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	mockRequestSender := mocks.NewMockRequestSender(mocker)
	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = mockRequestSender

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, testManagedCluster, false, false)
	expectSyncServiceAccount(t, mock, testManagedCluster, true)
	expectSyncRoleBinding(t, mock, testManagedCluster, true)
	expectSyncAgent(t, mock, testManagedCluster, false, true)
	expectSyncRegistration(t, mock, testManagedCluster, true)
	expectSyncManifest(t, mock, mockStatus, mockRequestSender, testManagedCluster, false, rancherManifestYAML)
	expectRancherConfigK8sCalls(t, mock)
	expectRancherGetAdminTokenHTTPCall(t, mockRequestSender)
	expectSyncCACertRancherHTTPCalls(t, mockRequestSender, "")
	expectSyncPrometheusScraper(mock, testManagedCluster, "", "", false, getCaCrt(), func(configMap *corev1.ConfigMap) error {
		asserts.Len(configMap.Data, 2, "no data found")
		prometheusYaml := configMap.Data["prometheus.yml"]
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")

		scrapeConfig, err := getScrapeConfig(prometheusYaml, testManagedCluster)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		validateScrapeConfig(t, scrapeConfig, prometheusConfigBasePath, false)
		return nil
	}, func(secret *corev1.Secret) error {
		scrapeConfigYaml := secret.Data[constants.PromAdditionalScrapeConfigsSecretKey]
		scrapeConfigs, err := metricsutils.ParseScrapeConfig(string(scrapeConfigYaml))
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		scrapeConfig := getJob(scrapeConfigs.Children(), testManagedCluster)
		validateScrapeConfig(t, scrapeConfig, managedCertsBasePath, false)
		return nil
	}, func(secret *corev1.Secret) error {
		asserts.Empty(secret.Data["ca-test"])
		return nil
	})

	// expect status updated with condition Ready=true and ManagedCARetrieved condition is not set because we don't provide
	// a non-zero length managed ca cert
	expectStatusUpdateReadyCondition(asserts, mock, mockStatus, corev1.ConditionTrue, "", false)

	// Create and make the request
	request := newRequest(namespace, testManagedCluster)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(vpoconstants.ReconcileLoopRequeueInterval), result.RequeueAfter)
}

// TestCreateVMCFetchCACertFromManagedCluster tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied and the caSecret field is empty
// THEN ensure that we fetch the CA cert secret from the managed cluster and populate the caSecret field
func TestCreateVMCFetchCACertFromManagedCluster(t *testing.T) {
	namespace := constants.VerrazzanoMultiClusterNamespace
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	mockRequestSender := mocks.NewMockRequestSender(mocker)
	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = mockRequestSender

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, testManagedCluster, false, false)
	expectSyncServiceAccount(t, mock, testManagedCluster, true)
	expectSyncRoleBinding(t, mock, testManagedCluster, true)
	expectSyncAgent(t, mock, testManagedCluster, true, false)
	expectSyncRegistration(t, mock, testManagedCluster, true)
	expectSyncManifest(t, mock, mockStatus, mockRequestSender, testManagedCluster, false, rancherManifestYAML)
	expectSyncCACertRancherHTTPCalls(t, mockRequestSender, `{"data":{"ca.crt":"base64-ca-cert"}}`)
	expectSyncCACertRancherK8sCalls(t, mock, mockRequestSender, true)
	expectSyncPrometheusScraper(mock, testManagedCluster, "", "", true, getCaCrt(), func(configMap *corev1.ConfigMap) error {
		asserts.Len(configMap.Data, 2, "no data found")
		prometheusYaml := configMap.Data["prometheus.yml"]
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")

		scrapeConfig, err := getScrapeConfig(prometheusYaml, testManagedCluster)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		validateScrapeConfig(t, scrapeConfig, prometheusConfigBasePath, false)
		return nil
	}, func(secret *corev1.Secret) error {
		scrapeConfigYaml := secret.Data[constants.PromAdditionalScrapeConfigsSecretKey]
		scrapeConfigs, err := metricsutils.ParseScrapeConfig(string(scrapeConfigYaml))
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		scrapeConfig := getJob(scrapeConfigs.Children(), testManagedCluster)
		validateScrapeConfig(t, scrapeConfig, managedCertsBasePath, false)
		return nil
	}, func(secret *corev1.Secret) error {
		asserts.NotEmpty(secret.Data["ca-test"], "Expected to find a managed cluster TLS cert")
		return nil
	})

	// expect status updated with condition Ready=true and ManagedCARetrieved
	expectStatusUpdateReadyCondition(asserts, mock, mockStatus, corev1.ConditionTrue, "", true)

	// Create and make the request
	request := newRequest(namespace, testManagedCluster)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(vpoconstants.ReconcileLoopRequeueInterval), result.RequeueAfter)
}

// TestCreateVMCWithExistingScrapeConfiguration tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied and prometheus is already configured with a scrape config for the cluster
// THEN ensure all the objects are created
func TestCreateVMCWithExistingScrapeConfiguration(t *testing.T) {
	namespace := "verrazzano-mc"
	jobs := `  - ` + constants.PrometheusJobNameKey + `: cluster1
    scrape_interval: 20s
    scrape_timeout: 15s
    scheme: http`
	prometheusYaml := `global:
  scrape_interval: 20s
  scrape_timeout: 10s
  evaluation_interval: 30s
scrape_configs:
` + jobs
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	mockRequestSender := mocks.NewMockRequestSender(mocker)
	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = mockRequestSender

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, testManagedCluster, true, false)
	expectSyncServiceAccount(t, mock, testManagedCluster, true)
	expectSyncRoleBinding(t, mock, testManagedCluster, true)
	expectSyncAgent(t, mock, testManagedCluster, false, false)
	expectSyncRegistration(t, mock, testManagedCluster, false)
	expectSyncManifest(t, mock, mockStatus, mockRequestSender, testManagedCluster, false, rancherManifestYAML)
	expectSyncCACertRancherK8sCalls(t, mock, mockRequestSender, false)
	expectSyncPrometheusScraper(mock, testManagedCluster, prometheusYaml, jobs, true, getCaCrt(), func(configMap *corev1.ConfigMap) error {

		// check for the modified entry
		asserts.Len(configMap.Data, 2, "no data found")
		asserts.NotEmpty(configMap.Data["ca-test"], "No cert entry found")
		prometheusYaml := configMap.Data["prometheus.yml"]
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")

		scrapeConfig, err := getScrapeConfig(prometheusYaml, testManagedCluster)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		validateScrapeConfig(t, scrapeConfig, prometheusConfigBasePath, true)
		return nil
	}, func(secret *corev1.Secret) error {
		scrapeConfigYaml := secret.Data[constants.PromAdditionalScrapeConfigsSecretKey]
		scrapeConfigs, err := metricsutils.ParseScrapeConfig(string(scrapeConfigYaml))
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		scrapeConfig := getJob(scrapeConfigs.Children(), testManagedCluster)
		validateScrapeConfig(t, scrapeConfig, managedCertsBasePath, true)
		return nil
	}, func(secret *corev1.Secret) error {
		asserts.NotEmpty(secret.Data["ca-test"], "Expected to find a managed cluster TLS cert")
		return nil
	})

	// expect status updated with condition Ready=true
	expectStatusUpdateReadyCondition(asserts, mock, mockStatus, corev1.ConditionTrue, "", false)

	// Create and make the request
	request := newRequest(namespace, testManagedCluster)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(vpoconstants.ReconcileLoopRequeueInterval), result.RequeueAfter)
}

// TestReplaceExistingScrapeConfiguration tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied and prometheus is already configured with a scrape configuration for the same cluster
// THEN ensure all the objects are created (existing configuration is replaced)
func TestReplaceExistingScrapeConfiguration(t *testing.T) {
	namespace := "verrazzano-mc"
	jobs := `  - ` + constants.PrometheusJobNameKey + `: test
    scrape_interval: 20s
    scrape_timeout: 15s
    scheme: http`
	prometheusYaml := `global:
  scrape_interval: 20s
  scrape_timeout: 10s
  evaluation_interval: 30s
scrape_configs:
` + jobs
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	mockRequestSender := mocks.NewMockRequestSender(mocker)
	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = mockRequestSender

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, testManagedCluster, true, false)
	expectSyncServiceAccount(t, mock, testManagedCluster, true)
	expectSyncRoleBinding(t, mock, testManagedCluster, true)
	expectSyncAgent(t, mock, testManagedCluster, false, true)
	expectSyncRegistration(t, mock, testManagedCluster, false)
	expectSyncManifest(t, mock, mockStatus, mockRequestSender, testManagedCluster, false, rancherManifestYAML)
	expectSyncCACertRancherK8sCalls(t, mock, mockRequestSender, false)
	expectSyncPrometheusScraper(mock, testManagedCluster, prometheusYaml, jobs, true, getCaCrt(), func(configMap *corev1.ConfigMap) error {

		asserts.Len(configMap.Data, 2, "no data found")
		asserts.NotNil(configMap.Data["ca-test"], "No cert entry found")
		prometheusYaml := configMap.Data["prometheus.yml"]
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")

		scrapeConfig, err := getScrapeConfig(prometheusYaml, testManagedCluster)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		validateScrapeConfig(t, scrapeConfig, prometheusConfigBasePath, true)
		return nil
	}, func(secret *corev1.Secret) error {
		scrapeConfigYaml := secret.Data[constants.PromAdditionalScrapeConfigsSecretKey]
		scrapeConfigs, err := metricsutils.ParseScrapeConfig(string(scrapeConfigYaml))
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		scrapeConfig := getJob(scrapeConfigs.Children(), testManagedCluster)
		validateScrapeConfig(t, scrapeConfig, managedCertsBasePath, true)
		return nil
	}, func(secret *corev1.Secret) error {
		asserts.NotEmpty(secret.Data["ca-test"], "Expected to find a managed cluster TLS cert")
		return nil
	})

	// expect status updated with condition Ready=true
	expectStatusUpdateReadyCondition(asserts, mock, mockStatus, corev1.ConditionTrue, "", false)

	// Create and make the request
	request := newRequest(namespace, testManagedCluster)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(vpoconstants.ReconcileLoopRequeueInterval), result.RequeueAfter)
}

// TestCreateVMC tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied
// AND the cluster has already been registered with Rancher
// THEN ensure all the objects are created
func TestCreateVMCClusterAlreadyRegistered(t *testing.T) {
	namespace := constants.VerrazzanoMultiClusterNamespace
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	mockRequestSender := mocks.NewMockRequestSender(mocker)
	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = mockRequestSender

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, testManagedCluster, true, true)
	expectSyncServiceAccount(t, mock, testManagedCluster, true)
	expectSyncRoleBinding(t, mock, testManagedCluster, true)
	expectSyncAgent(t, mock, testManagedCluster, false, true)
	expectSyncRegistration(t, mock, testManagedCluster, false)
	expectSyncManifest(t, mock, mockStatus, mockRequestSender, testManagedCluster, true, rancherManifestYAML)
	expectSyncCACertRancherK8sCalls(t, mock, mockRequestSender, false)
	expectSyncPrometheusScraper(mock, testManagedCluster, "", "", true, getCaCrt(), func(configMap *corev1.ConfigMap) error {
		asserts.Len(configMap.Data, 2, "no data found")
		asserts.NotEmpty(configMap.Data["ca-test"], "No cert entry found")
		prometheusYaml := configMap.Data["prometheus.yml"]
		asserts.NotEmpty(prometheusYaml, "No prometheus config yaml found")

		scrapeConfig, err := getScrapeConfig(prometheusYaml, testManagedCluster)
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		validateScrapeConfig(t, scrapeConfig, prometheusConfigBasePath, true)
		return nil
	}, func(secret *corev1.Secret) error {
		scrapeConfigYaml := secret.Data[constants.PromAdditionalScrapeConfigsSecretKey]
		scrapeConfigs, err := metricsutils.ParseScrapeConfig(string(scrapeConfigYaml))
		if err != nil {
			asserts.Fail("failed due to error %v", err)
		}
		scrapeConfig := getJob(scrapeConfigs.Children(), testManagedCluster)
		validateScrapeConfig(t, scrapeConfig, managedCertsBasePath, true)
		return nil
	}, func(secret *corev1.Secret) error {
		asserts.NotEmpty(secret.Data["ca-test"], "Expected to find a managed cluster TLS cert")
		return nil
	})

	// expect status updated with condition Ready=true
	expectStatusUpdateReadyCondition(asserts, mock, mockStatus, corev1.ConditionTrue, "", false)

	// Create and make the request
	request := newRequest(namespace, testManagedCluster)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(vpoconstants.ReconcileLoopRequeueInterval), result.RequeueAfter)
}

// TestCreateVMCSyncSvcAccountFailed tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN syncing of service account fails
// THEN ensure that the VMC status is updated to Ready=false with an appropriate message
func TestCreateVMCSyncSvcAccountFailed(t *testing.T) {
	namespace := constants.VerrazzanoMultiClusterNamespace
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, testManagedCluster, true, false)
	expectSyncServiceAccount(t, mock, testManagedCluster, false)

	// expect status updated with condition Ready=true
	expectStatusUpdateReadyCondition(asserts, mock, mockStatus, corev1.ConditionFalse, "failing syncServiceAccount", false)

	// Create and make the request
	request := newRequest(namespace, testManagedCluster)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results - there should have been an error returned for failing to sync svc account
	mocker.Finish()
	asserts.Nil(err)
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
}

// TestCreateVMCSyncRoleBindingFailed tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN syncing of role binding fails
// THEN ensure that the VMC status is updated to Ready=false with an appropriate message
func TestCreateVMCSyncRoleBindingFailed(t *testing.T) {
	namespace := constants.VerrazzanoMultiClusterNamespace
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	expectVmcGetAndUpdate(t, mock, name, true, false)
	expectSyncServiceAccount(t, mock, name, true)
	expectSyncRoleBinding(t, mock, name, false)

	// expect status updated with condition Ready=true
	expectStatusUpdateReadyCondition(asserts, mock, mockStatus, corev1.ConditionFalse, "failing syncRoleBinding", false)

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results - there should have been an error returned
	mocker.Finish()
	asserts.Nil(err)
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
}

// TestDeleteVMC tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been deleted
// THEN ensure the object is not processed
func TestDeleteVMC(t *testing.T) {
	namespace := "verrazzano-install"
	labels := map[string]string{"label1": "test"}
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the VerrazzanoManagedCluster resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: testManagedCluster}, gomock.Not(gomock.Nil())).
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
			vmc.Status = clustersapi.VerrazzanoManagedClusterStatus{
				PrometheusHost: getPrometheusHost(),
			}

			return nil
		})

	jobs := `  - ` + constants.PrometheusJobNameKey + `: test
    scrape_interval: 20s
    scrape_timeout: 15s
    scheme: http
  - ` + constants.PrometheusJobNameKey + `: test2
    scrape_interval: 20s
    scrape_timeout: 15s
    scheme: http`

	// Expect a call to get the additional scrape config secret - return it configured with the two scrape jobs
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: vpoconstants.VerrazzanoMonitoringNamespace, Name: constants.PromAdditionalScrapeConfigsSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				constants.PromAdditionalScrapeConfigsSecretKey: []byte(jobs),
			}
			return nil
		})

	// Expect a call to get the additional scrape config secret (we call controllerruntime.CreateOrUpdate so it fetches again) - return it
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: vpoconstants.VerrazzanoMonitoringNamespace, Name: constants.PromAdditionalScrapeConfigsSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				constants.PromAdditionalScrapeConfigsSecretKey: []byte(jobs),
			}
			return nil
		})

	// Expect a call to update the additional scrape config secret
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.UpdateOption) error {
			// validate that the scrape config for the managed cluster is no longer present
			scrapeConfigs, err := metricsutils.ParseScrapeConfig(string(secret.Data[constants.PromAdditionalScrapeConfigsSecretKey]))
			if err != nil {
				return err
			}
			asserts.Len(scrapeConfigs.Children(), 1, "Expected only one scrape config")
			scrapeJobName := scrapeConfigs.Children()[0].Search(constants.PrometheusJobNameKey).Data()
			asserts.Equal("test2", scrapeJobName)
			return nil
		})

	// Expect a call to get the managed cluster TLS certs secret - return it configured with two managed cluster certs
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: vpoconstants.VerrazzanoMonitoringNamespace, Name: vpoconstants.PromManagedClusterCACertsSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				"ca-test":  []byte("ca-cert-1"),
				"ca-test2": []byte("ca-cert-1"),
			}
			return nil
		})

	// Expect a call to update the managed cluster TLS certs secret
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.UpdateOption) error {
			// validate that the cert for the cluster being deleted is no longer present
			asserts.Len(secret.Data, 1, "Expected only one managed cluster cert")
			asserts.Contains(secret.Data, "ca-test2", "Expected to find cert for managed cluster not being deleted")
			return nil
		})

	// Expect a call to update the VerrazzanoManagedCluster finalizer
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.True(len(vmc.ObjectMeta.Finalizers) == 0, "Wrong number of finalizers")
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, testManagedCluster)
	reconciler := newVMCReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestSyncManifestSecretFailRancherRegistration tests syncing the manifest secret
// when Rancher registration fails
// GIVEN a call to sync the manifest secret
// WHEN Rancher registration fails
// THEN the manifest secret is still created and syncManifestSecret returns no error
func TestSyncManifestSecretFailRancherRegistration(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	clusterName := "cluster1"
	caData := "ca"
	userData := "user"
	passwordData := "pw"
	kubeconfigData := "fakekubeconfig"
	urlData := "https://testhost:443"

	// Expect a call to get the Agent secret
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetAgentSecretName(clusterName)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				mcconstants.KubeconfigKey: []byte(kubeconfigData),
			}
			return nil
		})

	// Expect a call to get the registration secret
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetRegistrationSecretName(clusterName)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				mcconstants.ManagedClusterNameKey:   []byte(clusterName),
				mcconstants.CaCrtKey:                []byte(caData),
				mcconstants.RegistrationUsernameKey: []byte(userData),
				mcconstants.RegistrationPasswordKey: []byte(passwordData),
				mcconstants.ESURLKey:                []byte(urlData),
			}
			return nil
		})

	// Expect a call to get the manifest secret - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetManifestSecretName(clusterName)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoMultiClusterNamespace, Resource: "Secret"}, GetManifestSecretName(clusterName)))

	// Expect a call to get the Rancher ingress and return no spec rules, which will cause registration to fail
	mock.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: rancherNamespace, Name: rancherIngressName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, nsName types.NamespacedName, ingress *networkingv1.Ingress) error {
			return nil
		})

	mock.EXPECT().Status().Return(mockStatus)
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersapi.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(clustersapi.RegistrationFailed, vmc.Status.RancherRegistration.Status)
			asserts.Equal("Failed to create Rancher API client: Failed, Rancher ingress cattle-system/rancher is missing host names", vmc.Status.RancherRegistration.Message)
			return nil
		})

	// Expect a call to create the manifest secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			data := secret.Data[mcconstants.YamlKey]
			asserts.NotZero(len(data), "Expected yaml data in manifest secret")
			return nil
		})

	// Expect a call to update the VerrazzanoManagedCluster kubeconfig secret testManagedCluster - return success
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(vmc.Spec.ManagedClusterManifestSecret, GetManifestSecretName(clusterName), "Manifest secret testManagedCluster did not match")
			return nil
		})

	// Create a reconciler and call the function to sync the manifest secret - the call to register the cluster with Rancher will
	// fail but the result of syncManifestSecret should be success
	vmc := clustersapi.VerrazzanoManagedCluster{ObjectMeta: metav1.ObjectMeta{Name: clusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}}
	reconciler := newVMCReconciler(mock)
	reconciler.log = vzlog.DefaultLogger()

	vzVMCWaitingForClusterID, err := reconciler.syncManifestSecret(context.TODO(), &vmc)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.False(vzVMCWaitingForClusterID)
}

// TestSyncManifestSecretEmptyRancherManifest tests syncing the manifest secret
// when Rancher returns an empty registration manifest YAML string.
// GIVEN a call to sync the manifest secret
// WHEN Rancher returns an empty manifest
// THEN the status is set to failed on the VMC with an appropriate message and the syncManifestSecret call returns an error
func TestSyncManifestSecretEmptyRancherManifest(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	mockRequestSender := mocks.NewMockRequestSender(mocker)
	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = mockRequestSender

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	caData := "ca"
	userData := "user"
	passwordData := "pw"
	kubeconfigData := "fakekubeconfig"
	urlData := "https://testhost:443"

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetAgentSecretName(testManagedCluster)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				mcconstants.KubeconfigKey: []byte(kubeconfigData),
			}
			return nil
		})

	// Expect a call to get the registration secret
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetRegistrationSecretName(testManagedCluster)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				mcconstants.ManagedClusterNameKey:   []byte(testManagedCluster),
				mcconstants.CaCrtKey:                []byte(caData),
				mcconstants.RegistrationUsernameKey: []byte(userData),
				mcconstants.RegistrationPasswordKey: []byte(passwordData),
				mcconstants.ESURLKey:                []byte(urlData),
			}
			return nil
		})

	// Expect all the calls needed to register the cluster with Rancher - note we are passing an empty string for the Rancher manifest YAML
	// that will be returned when calling the Rancher API
	expectRegisterClusterWithRancher(t, mock, mockRequestSender, testManagedCluster, false, "")

	// Expect the Rancher registration status to be set appropriately
	mock.EXPECT().Status().Return(mockStatus)
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersapi.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(clustersapi.RegistrationFailed, vmc.Status.RancherRegistration.Status)
			asserts.Equal(unitTestRancherClusterID, vmc.Status.RancherRegistration.ClusterID)
			asserts.Equal("Empty Rancher manifest YAML", vmc.Status.RancherRegistration.Message)
			return nil
		})

	// Create a reconciler and call the function to sync the manifest secret
	vmc := clustersapi.VerrazzanoManagedCluster{ObjectMeta: metav1.ObjectMeta{Name: testManagedCluster, Namespace: constants.VerrazzanoMultiClusterNamespace}}
	reconciler := newVMCReconciler(mock)
	reconciler.log = vzlog.DefaultLogger()

	vzVMCWaitingForClusterID, err := reconciler.syncManifestSecret(context.TODO(), &vmc)

	// Validate the results
	mocker.Finish()
	asserts.ErrorContains(err, "Failed retrieving Rancher manifest, YAML is an empty string")
	asserts.False(vzVMCWaitingForClusterID)
}

// TestRegisterClusterWithRancherK8sErrorCases tests errors cases using the Kubernetes
// client when registering with Rancher.
func TestRegisterClusterWithRancherK8sErrorCases(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// GIVEN a call to register a managed cluster with Rancher
	// WHEN the call to get the ingress host name returns no ingress rules
	// THEN the registration call returns an error

	// Expect a call to get the ingress host name but there are no ingress rules
	mock.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: rancherNamespace, Name: rancherIngressName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, nsName types.NamespacedName, ingress *networkingv1.Ingress) error {
			return nil
		})

	rc, err := newRancherConfig(mock, vzlog.DefaultLogger())

	mocker.Finish()
	asserts.Error(err)
	asserts.Nil(rc)

	// GIVEN a call to register a managed cluster with Rancher
	// WHEN the call to get the Rancher root CA cert secret fails
	// THEN the registration call returns an error
	mocker = gomock.NewController(t)
	mock = mocks.NewMockClient(mocker)

	// Expect a call to get the ingress host name
	mock.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: rancherNamespace, Name: rancherIngressName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, nsName types.NamespacedName, ingress *networkingv1.Ingress) error {
			rule := networkingv1.IngressRule{Host: "rancher.unit-test.com"}
			ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
			return nil
		})

	// Expect a call to get the secret with the Rancher root CA cert but the call fails
	mock.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: rancherNamespace, Name: rancherTLSSecret}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, nsName types.NamespacedName, secret *corev1.Secret) error {
			return errors.NewResourceExpired("something bad happened")
		})

	rc, err = newRancherConfig(mock, vzlog.DefaultLogger())

	mocker.Finish()
	asserts.Error(err)
	asserts.Nil(rc)
}

// TestRegisterClusterWithRancherHTTPErrorCases tests errors cases using the HTTP
// client when registering with Rancher.
func TestRegisterClusterWithRancherHTTPErrorCases(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockRequestSender := mocks.NewMockRequestSender(mocker)

	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = mockRequestSender

	// GIVEN a call to register a managed cluster with Rancher
	// WHEN the call to get the Rancher admin token fails
	// THEN the registration call returns an error

	// Expect all of the Kubernetes calls
	expectRancherConfigK8sCalls(t, mock)

	// Expect an HTTP request to fetch the admin token from Rancher but the call fails
	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(loginURIPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte{}))
			resp := &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	rc, err := newRancherConfig(mock, vzlog.DefaultLogger())

	mocker.Finish()
	asserts.Error(err)
	asserts.Nil(rc)

	// GIVEN a call to register a managed cluster with Rancher
	// WHEN the call to import the cluster into Rancher fails
	// THEN the registration call returns an error
	mocker = gomock.NewController(t)
	mock = mocks.NewMockClient(mocker)
	mockRequestSender = mocks.NewMockRequestSender(mocker)
	rancherHTTPClient = mockRequestSender

	// Expect all of the Kubernetes calls
	expectRancherConfigK8sCalls(t, mock)

	// Expect an HTTP request to fetch the admin token from Rancher
	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(loginURIPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	// Expect an HTTP request to import the cluster to Rancher but the call fails
	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(clusterPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte{}))
			resp := &http.Response{
				StatusCode: http.StatusConflict,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	rc, err = newRancherConfig(mock, vzlog.DefaultLogger())
	asserts.NoError(err)

	regYAML, _, err := registerManagedClusterWithRancher(rc, testManagedCluster, "", vzlog.DefaultLogger())

	mocker.Finish()
	asserts.Error(err)
	asserts.Empty(regYAML)

	// GIVEN a call to register a managed cluster with Rancher
	// WHEN the call to create the Rancher registration token fails
	// THEN the registration call returns an error
	mocker = gomock.NewController(t)
	mock = mocks.NewMockClient(mocker)
	mockRequestSender = mocks.NewMockRequestSender(mocker)
	rancherHTTPClient = mockRequestSender

	// Expect all of the Kubernetes calls
	expectRancherConfigK8sCalls(t, mock)

	// Expect an HTTP request to fetch the admin token from Rancher
	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(loginURIPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	// Expect an HTTP request to import the cluster to Rancher
	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(clusterPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"id":"some-cluster"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
			}
			return resp, nil
		})

	// Expect an HTTP request to create the registration token in Rancher but the call fails
	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(clusterRegTokenPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte{}))
			resp := &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	rc, err = newRancherConfig(mock, vzlog.DefaultLogger())
	asserts.NoError(err)

	regYAML, _, err = registerManagedClusterWithRancher(rc, testManagedCluster, "", vzlog.DefaultLogger())

	mocker.Finish()
	asserts.Error(err)
	asserts.Empty(regYAML)

	// GIVEN a call to register a managed cluster with Rancher
	// WHEN the call to get the Rancher manifest YAML fails
	// THEN the registration call returns an error
	mocker = gomock.NewController(t)
	mock = mocks.NewMockClient(mocker)
	mockRequestSender = mocks.NewMockRequestSender(mocker)
	rancherHTTPClient = mockRequestSender

	// Expect all of the Kubernetes calls
	expectRancherConfigK8sCalls(t, mock)

	// Expect an HTTP request to fetch the admin token from Rancher
	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(loginURIPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	// Expect an HTTP request to import the cluster to Rancher
	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(clusterPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"id":"some-cluster"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
			}
			return resp, nil
		})

	// Expect an HTTP request to create the registration token in Rancher
	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(clusterRegTokenPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"manifest-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	// Expect an HTTP request to fetch the manifest YAML from Rancher but the call fails
	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte{}))
			resp := &http.Response{
				StatusCode: http.StatusUnsupportedMediaType,
				Body:       r,
				Request:    &http.Request{Method: http.MethodGet},
			}
			return resp, nil
		})

	rc, err = newRancherConfig(mock, vzlog.DefaultLogger())
	asserts.NoError(err)

	regYAML, _, err = registerManagedClusterWithRancher(rc, testManagedCluster, "", vzlog.DefaultLogger())

	mocker.Finish()
	asserts.Error(err)
	asserts.Empty(regYAML)
}

// GIVEN a call to register a managed cluster with Rancher
// WHEN the call to get the admin token from Rancher fails
// AND the error is retryable
// THEN ensure that the request is retried
func TestRegisterClusterWithRancherRetryRequest(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockRequestSender := mocks.NewMockRequestSender(mocker)

	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = mockRequestSender

	// replace the retry configuration so all of the retries happen very quickly
	retrySteps := 3
	savedRetry := defaultRetry
	defer func() {
		defaultRetry = savedRetry
	}()
	defaultRetry = wait.Backoff{
		Steps:    retrySteps,
		Duration: 1 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.1,
	}

	// Expect all of the Kubernetes calls
	expectRancherConfigK8sCalls(t, mock)

	// Expect an HTTP request to fetch the admin token from Rancher - return an error response and
	// the request should be retried for a total of "retrySteps" # of times
	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(loginURIPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte{}))
			resp := &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		}).Times(retrySteps)

	_, err := newRancherConfig(mock, vzlog.DefaultLogger())

	mocker.Finish()
	asserts.Error(err)
}

// TestUpateStatus tests the updateStatus function
func TestUpateStatus(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// GIVEN a VMC with a status state unset and the last agent connect time set
	// WHEN the updateStatus function is called
	// THEN the status state is updated to pending
	mock.EXPECT().Status().Return(mockStatus)
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersapi.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(clustersapi.StatePending, vmc.Status.State)
			return nil
		})

	vmc := clustersapi.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testManagedCluster,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Status: clustersapi.VerrazzanoManagedClusterStatus{
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
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersapi.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(clustersapi.StateActive, vmc.Status.State)
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
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersapi.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(clustersapi.StateInactive, vmc.Status.State)
			return nil
		})

	err = reconciler.updateStatus(context.TODO(), &vmc)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = clustersapi.AddToScheme(scheme)
	return scheme
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// testManagedCluster - The testManagedCluster to use in the request
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
func expectSyncRoleBinding(t *testing.T, mock *mocks.MockClient, name string, succeed bool) {
	asserts := assert.New(t)

	// Expect a call to get the RoleBinding - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: generateManagedResourceName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "RoleBinding"}, generateManagedResourceName(name)))

	// Expect a call to create the RoleBinding - return success or failure based on the succeed argument
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, binding *rbacv1.RoleBinding, opts ...client.CreateOption) error {
			if succeed {
				asserts.Equalf(generateManagedResourceName(name), binding.Name, "RoleBinding testManagedCluster did not match")
				asserts.Equalf(vpoconstants.MCClusterRole, binding.RoleRef.Name, "RoleBinding roleref did not match")
				asserts.Equalf(generateManagedResourceName(name), binding.Subjects[0].Name, "Subject did not match")
				asserts.Equalf(constants.VerrazzanoMultiClusterNamespace, binding.Subjects[0].Namespace, "Subject namespace did not match")
				return nil
			}
			return errors.NewInternalError(fmt.Errorf("failing syncRoleBinding"))
		})
}

// Expect syncServiceAccount related calls
func expectSyncServiceAccount(t *testing.T, mock *mocks.MockClient, name string, succeed bool) {
	asserts := assert.New(t)

	// Expect a call to get the ServiceAccount - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: generateManagedResourceName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "ServiceAccount"}, generateManagedResourceName(name)))

	// Expect a call to create the ServiceAccount - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, serviceAccount *corev1.ServiceAccount, opts ...client.CreateOption) error {
			asserts.Equalf(constants.VerrazzanoMultiClusterNamespace, serviceAccount.Namespace, "ServiceAccount namespace did not match")
			asserts.Equalf(generateManagedResourceName(name), serviceAccount.Name, "ServiceAccount testManagedCluster did not match")
			return nil
		})

	// Expect a call to get the Token Secret - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: generateManagedResourceName(name) + "-token"}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, generateManagedResourceName(name)+"-token"))

	// Expect a call to create the Token Secret - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			asserts.Equalf(constants.VerrazzanoMultiClusterNamespace, secret.Namespace, "Secret namespace did not match")
			asserts.Equalf(generateManagedResourceName(name)+"-token", secret.Name, "Secret testManagedCluster did not match")
			return nil
		})

	// Expect a call to update the VerrazzanoManagedCluster service account name - return success or
	// failure depending on the succeed argument
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			if succeed {
				asserts.Equal(vmc.Spec.ServiceAccount, generateManagedResourceName(name), "ServiceAccount testManagedCluster did not match")
				return nil
			}
			return errors.NewInternalError(fmt.Errorf("failing syncServiceAccount"))
		})
}

// Expect syncAgent related calls
func expectSyncAgent(t *testing.T, mock *mocks.MockClient, name string, rancherEnabled bool, noSASecret bool) {
	saSecretName := "saSecret"
	rancherURL := "http://rancher-url"
	rancherCAData := "rancherCAData"
	userAPIServerURL := "https://testurl"
	if noSASecret {
		// Expect a call to get the ServiceAccount, return without the secret set
		mock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: generateManagedResourceName(name)}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, sa *corev1.ServiceAccount) error {
				sa.Name = name.Name
				return nil
			})
		// Set the secret name to the service account token created by the VMC controller
		saSecretName = generateManagedResourceName(name) + "-token"

	} else {
		// Expect a call to get the ServiceAccount, return one with the secret testManagedCluster set
		mock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: generateManagedResourceName(name)}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, sa *corev1.ServiceAccount) error {
				sa.Secrets = []corev1.ObjectReference{{
					Name: saSecretName,
				}}
				return nil
			})
	}

	// ONLY if the rancherBasedKubeconfig feature flag is enabled - Expect a call to list Verrazzanos
	// and return a Verrazzano that has Rancher URL in status only if rancherEnabled is true
	if rancherBasedKubeConfigEnabled {
		mock.EXPECT().
			List(gomock.Any(), &vzapi.VerrazzanoList{}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, list *vzapi.VerrazzanoList, opts ...client.ListOption) error {
				var status vzapi.VerrazzanoStatus
				if rancherEnabled {
					status = vzapi.VerrazzanoStatus{
						VerrazzanoInstance: &vzapi.InstanceInfo{RancherURL: &rancherURL},
					}
				}
				vz := vzapi.Verrazzano{
					Spec:   vzapi.VerrazzanoSpec{},
					Status: status,
				}
				list.Items = append(list.Items, vz)
				return nil
			})
	}
	// Expect a call to get the service token secret, return the secret with the token
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: saSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				mcconstants.TokenKey: []byte(token),
			}
			return nil
		})

	if rancherEnabled && rancherBasedKubeConfigEnabled {
		// Expect a call to get the tls-ca-additional secret, return the secret as not found
		mock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Namespace: constants.RancherSystemNamespace, Name: constants.AdditionalTLS}, gomock.Not(gomock.Nil())).
			Return(errors.NewNotFound(schema.GroupResource{Group: constants.RancherSystemNamespace, Resource: "Secret"}, constants.AdditionalTLS))

		// Expect a call to get the Rancher ingress tls secret, return the secret with the fields set
		mock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Namespace: constants.RancherSystemNamespace, Name: rancherTLSSecret}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
				secret.Data = map[string][]byte{
					mcconstants.CaCrtKey: []byte(rancherCAData),
				}
				return nil
			})
	} else {
		// Expect a call to get the verrazzano-admin-cluster configmap
		mock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: vpoconstants.AdminClusterConfigMapName}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, cm *corev1.ConfigMap) error {
				cm.Data = map[string]string{
					vpoconstants.ServerDataKey: userAPIServerURL,
				}
				return nil
			})
	}

	// Expect a call to get the Agent secret - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetAgentSecretName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoMultiClusterNamespace, Resource: "Secret"}, GetAgentSecretName(name)))

	// Expect a call to create the Agent secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			adminKubeconfig := string(secret.Data[mcconstants.KubeconfigKey])
			if rancherEnabled && rancherBasedKubeConfigEnabled {
				assert.Contains(t, adminKubeconfig, "server: "+rancherURL)
			} else {
				assert.Contains(t, adminKubeconfig, "server: "+userAPIServerURL)
			}
			return nil
		})
}

// Expect syncRegistration related calls
func expectSyncRegistration(t *testing.T, mock *mocks.MockClient, name string, externalES bool) {
	const vzEsURLData = "https://vz-testhost:443"
	const vzUserData = "vz-user"
	const vzPasswordData = "vz-pw"
	const vzCaData = "vz-ca"

	const externalEsURLData = "https://external-testhost:443"
	const externalUserData = "external-user"
	const externalPasswordData = "external-pw"
	const externalCaData = "external-ca"

	fluentdESURL := "http://verrazzano-authproxy-elasticsearch:8775"
	fluentdESSecret := "verrazzano"
	esSecret := constants.VerrazzanoESInternal
	if externalES {
		fluentdESURL = externalEsURLData
		fluentdESSecret = "some-external-es-secret"
		esSecret = fluentdESSecret
	}

	asserts := assert.New(t)

	// Expect a call to get the registration secret - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetRegistrationSecretName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoMultiClusterNamespace, Resource: "Secret"}, GetRegistrationSecretName(name)))

	// Expect a call to list Verrazzanos - return the Verrazzano configured with fluentd
	mock.EXPECT().
		List(gomock.Any(), &vzapi.VerrazzanoList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, list *vzapi.VerrazzanoList, opts ...client.ListOption) error {
			vz := vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{
							ElasticsearchURL:    fluentdESURL,
							ElasticsearchSecret: fluentdESSecret,
						},
					},
				},
			}
			list.Items = append(list.Items, vz)
			return nil
		})

	// Expect a call to get the tls ingress and return the ingress.
	if !externalES {
		mock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: vmiIngest}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *networkingv1.Ingress) error {
				ingress.TypeMeta = metav1.TypeMeta{
					APIVersion: "networking.k8s.io/v1",
					Kind:       "ingress"}
				ingress.ObjectMeta = metav1.ObjectMeta{
					Namespace: name.Namespace,
					Name:      name.Name}
				ingress.Spec.Rules = []networkingv1.IngressRule{{
					Host: "vz-testhost",
				}}
				return nil
			})
	}

	// Expect a call to get the opensearch secret, return the secret with the fields set
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: esSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			if externalES {
				secret.Data = map[string][]byte{
					mcconstants.VerrazzanoUsernameKey: []byte(externalUserData),
					mcconstants.VerrazzanoPasswordKey: []byte(externalPasswordData),
					mcconstants.FluentdESCaBundleKey:  []byte(externalCaData),
				}
			} else {
				secret.Data = map[string][]byte{
					mcconstants.VerrazzanoUsernameKey: []byte(vzUserData),
					mcconstants.VerrazzanoPasswordKey: []byte(vzPasswordData),
				}
			}
			return nil
		})

	// Expect a call to get the tls secret, return the secret with the fields set
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: "verrazzano-tls"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				mcconstants.CaCrtKey: []byte(vzCaData),
			}
			return nil
		})

	// Expect a call to get the tls-ca-additional secret, return the secret as not found
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.RancherSystemNamespace, Name: constants.AdditionalTLS}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.RancherSystemNamespace, Resource: "Secret"}, constants.AdditionalTLS))

	// Expect a call to get the keycloak ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "keycloak", Name: "keycloak"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *networkingv1.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			ingress.Spec.Rules = []networkingv1.IngressRule{{
				Host: "keycloak",
			}}
			return nil
		})

	// Expect a call to create the registration secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			asserts.Equalf(testManagedCluster, string(secret.Data[mcconstants.ManagedClusterNameKey]), "Incorrect cluster testManagedCluster in Registration secret ")
			asserts.Equalf("https://keycloak", string(secret.Data[mcconstants.KeycloakURLKey]), "Incorrect admin ca bundle in Registration secret ")
			asserts.Equalf(vzCaData, string(secret.Data[mcconstants.AdminCaBundleKey]), "Incorrect admin ca bundle in Registration secret ")
			if externalES {
				asserts.Equalf(externalEsURLData, string(secret.Data[mcconstants.ESURLKey]), "Incorrect ES URL in Registration secret ")
				asserts.Equalf(externalCaData, string(secret.Data[mcconstants.ESCaBundleKey]), "Incorrect ES ca bundle in Registration secret ")
				asserts.Equalf(externalUserData, string(secret.Data[mcconstants.RegistrationUsernameKey]), "Incorrect ES user in Registration secret ")
				asserts.Equalf(externalPasswordData, string(secret.Data[mcconstants.RegistrationPasswordKey]), "Incorrect ES password in Registration secret ")
			} else {
				asserts.Equalf(vzEsURLData, string(secret.Data[mcconstants.ESURLKey]), "Incorrect ES URL in Registration secret ")
				asserts.Equalf(vzCaData, string(secret.Data[mcconstants.ESCaBundleKey]), "Incorrect ES ca bundle in Registration secret ")
				asserts.Equalf(vzUserData, string(secret.Data[mcconstants.RegistrationUsernameKey]), "Incorrect ES user in Registration secret ")
				asserts.Equalf(vzPasswordData, string(secret.Data[mcconstants.RegistrationPasswordKey]), "Incorrect ES password in Registration secret ")
			}
			return nil
		})
}

// Expect syncManifest related calls
func expectSyncManifest(t *testing.T, mock *mocks.MockClient, mockStatus *mocks.MockStatusWriter, mockRequestSender *mocks.MockRequestSender,
	name string, clusterAlreadyRegistered bool, expectedRancherYAML string) {

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
				mcconstants.KubeconfigKey: []byte(kubeconfigData),
			}
			return nil
		})

	// Expect a call to get the registration secret
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetRegistrationSecretName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				mcconstants.ManagedClusterNameKey:   []byte(clusterName),
				mcconstants.CaCrtKey:                []byte(caData),
				mcconstants.RegistrationUsernameKey: []byte(userData),
				mcconstants.RegistrationPasswordKey: []byte(passwordData),
				mcconstants.ESURLKey:                []byte(urlData),
			}
			return nil
		})

	// Expect a call to get the manifest secret - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: GetManifestSecretName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoMultiClusterNamespace, Resource: "Secret"}, GetManifestSecretName(name)))

	// Expect all the calls needed to register the cluster with Rancher
	expectRegisterClusterWithRancher(t, mock, mockRequestSender, name, clusterAlreadyRegistered, expectedRancherYAML)

	mock.EXPECT().Status().Return(mockStatus)
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersapi.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(clustersapi.RegistrationCompleted, vmc.Status.RancherRegistration.Status)
			asserts.Equal(unitTestRancherClusterID, vmc.Status.RancherRegistration.ClusterID)
			asserts.Equal("Registration of managed cluster completed successfully", vmc.Status.RancherRegistration.Message)
			return nil
		})

	// Expect a call to create the manifest secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			data := secret.Data[mcconstants.YamlKey]
			asserts.NotZero(len(data), "Expected yaml data in manifest secret")

			// YAML should contain the Rancher manifest things
			yamlString := string(data)
			asserts.True(strings.Contains(yamlString, expectedRancherYAML), "Manifest YAML does not contain the correct Rancher resources")

			return nil
		})

	// Expect a call to update the VerrazzanoManagedCluster kubeconfig secret testManagedCluster - return success
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(vmc.Spec.ManagedClusterManifestSecret, GetManifestSecretName(name), "Manifest secret testManagedCluster did not match")
			return nil
		})
}

func expectVmcGetAndUpdate(t *testing.T, mock *mocks.MockClient, name string, caSecretExists bool, rancherClusterAlreadyRegistered bool) {
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
			if caSecretExists {
				vmc.Spec = clustersapi.VerrazzanoManagedClusterSpec{
					CASecret: getCASecretName(name.Name),
				}
			}
			vmc.Status = clustersapi.VerrazzanoManagedClusterStatus{
				PrometheusHost: getPrometheusHost(),
			}
			if rancherClusterAlreadyRegistered {
				vmc.Status.RancherRegistration = clustersapi.RancherRegistration{
					Status:    clustersapi.RegistrationCompleted,
					ClusterID: unitTestRancherClusterID,
				}
			}
			return nil
		})

	// Expect a call to update the VerrazzanoManagedCluster finalizer
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.True(len(vmc.ObjectMeta.Finalizers) == 1, "Wrong number of finalizers")
			asserts.Equal(finalizerName, vmc.ObjectMeta.Finalizers[0], "wrong finalizer")
			return nil
		})

}

func expectSyncPrometheusScraper(mock *mocks.MockClient, vmcName string, prometheusYaml string, jobs string, caSecretExists bool, cacrtSecretData string,
	f AssertFn, additionalScrapeConfigsAssertFunc secretAssertFn, managedClusterCertsAssertFunc secretAssertFn) {
	const internalSecretPassword = "nRXlxXgMwN" //nolint:gosec //#gosec G101

	if caSecretExists {
		// Expect a call to get the prometheus secret - return it
		mock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: getCASecretName(vmcName)}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {

				secret.Data = map[string][]byte{
					"cacrt": []byte(cacrtSecretData),
				}
				return nil
			})
	}

	// Expect a call to get the Verrazzano Prometheus internal secret - return it
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VerrazzanoPromInternal}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				mcconstants.VerrazzanoPasswordKey: []byte(internalSecretPassword),
			}
			return nil
		})

	// Expect a call to get the additional scrape config secret - return it
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: vpoconstants.VerrazzanoMonitoringNamespace, Name: constants.PromAdditionalScrapeConfigsSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				constants.PromAdditionalScrapeConfigsSecretKey: []byte(jobs),
			}
			return nil
		})

	// Expect a call to get the additional scrape config secret (we call controllerruntime.CreateOrUpdate so it fetches again) - return it
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: vpoconstants.VerrazzanoMonitoringNamespace, Name: constants.PromAdditionalScrapeConfigsSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				constants.PromAdditionalScrapeConfigsSecretKey: []byte(jobs),
			}
			return nil
		})

	// Expect a call to update the additional scrape config secret
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.UpdateOption) error {
			return additionalScrapeConfigsAssertFunc(secret)
		})

	// Expect a call to get the managed cluster TLS certs secret - return NotFound error
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: vpoconstants.VerrazzanoMonitoringNamespace, Name: vpoconstants.PromManagedClusterCACertsSecretName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, vpoconstants.PromManagedClusterCACertsSecretName))

	// Expect a call to update the managed cluster TLS certs secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.UpdateOption) error {
			return managedClusterCertsAssertFunc(secret)
		})

}

// expectRegisterClusterWithRancher asserts all of the expected calls on the Kubernetes client mock and the HTTP client mock
func expectRegisterClusterWithRancher(t *testing.T,
	k8sMock *mocks.MockClient,
	requestSenderMock *mocks.MockRequestSender,
	clusterName string,
	clusterAlreadyRegistered bool,
	expectedRancherYAML string) {

	expectRancherConfigK8sCalls(t, k8sMock)
	expectRegisterClusterWithRancherHTTPCalls(t, requestSenderMock, clusterName, clusterAlreadyRegistered, expectedRancherYAML)
}

// expectRegisterClusterWithRancherHTTPCalls asserts all of the expected calls on the HTTP client mock
func expectRegisterClusterWithRancherHTTPCalls(t *testing.T, requestSenderMock *mocks.MockRequestSender, clusterName string, clusterAlreadyRegistered bool, rancherManifestYAML string) {
	asserts := assert.New(t)

	expectRancherGetAdminTokenHTTPCall(t, requestSenderMock)

	if !clusterAlreadyRegistered {
		// Cluster is not already registered - we now only expect import to happen if the cluster is NOT already registered
		// Expect an HTTP request to import the cluster to Rancher
		requestSenderMock.EXPECT().
			Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(clusterPath)).
			DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
				asserts.Equal(clusterPath, req.URL.Path)

				var resp *http.Response
				r := io.NopCloser(bytes.NewReader([]byte(`{"id":"` + unitTestRancherClusterID + `"}`)))
				resp = &http.Response{
					StatusCode: http.StatusCreated,
					Body:       r,
				}
				return resp, nil
			})
	}

	manifestToken := "unit-test-manifest-token"

	// Expect an HTTP request to create the registration token in Rancher
	requestSenderMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(clusterRegTokenPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			asserts.Equal(clusterRegTokenPath, req.URL.Path)

			// assert that the cluster ID in the request body is what we expect
			body, err := io.ReadAll(req.Body)
			asserts.NoError(err)
			jsonString, err := gabs.ParseJSON(body)
			asserts.NoError(err)
			clusterID, ok := jsonString.Path("clusterId").Data().(string)
			asserts.True(ok)
			asserts.Equal(unitTestRancherClusterID, clusterID)

			// return a response with the manifest token
			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"` + manifestToken + `"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	// Expect an HTTP request to fetch the manifest YAML from Rancher
	requestSenderMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			asserts.Equal(manifestPath+manifestToken+"_"+unitTestRancherClusterID+".yaml", req.URL.Path)

			r := io.NopCloser(bytes.NewReader([]byte(rancherManifestYAML)))
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
				Request:    &http.Request{Method: http.MethodGet},
			}
			return resp, nil
		})
}

func expectRancherGetAdminTokenHTTPCall(t *testing.T, requestSenderMock *mocks.MockRequestSender) {
	asserts := assert.New(t)

	// Expect an HTTP request to fetch the admin token from Rancher
	requestSenderMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(loginURIPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			asserts.Equal(loginQueryString, req.URL.RawQuery)

			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

}

// expectSyncCACertRancherHTTPCalls asserts all of the expected calls on the HTTP client mock when sync'ing the managed cluster
// CA cert secret
func expectSyncCACertRancherHTTPCalls(t *testing.T, requestSenderMock *mocks.MockRequestSender, caCertSecretData string) {
	// Expect an HTTP request to fetch the managed cluster info from Rancher
	fetchClusterPath := fmt.Sprintf("/v3/clusters/%s", unitTestRancherClusterID)
	requestSenderMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(fetchClusterPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {

			r := io.NopCloser(bytes.NewReader([]byte(`{"state":"active","agentImage":"test-image:1.0.0"}`)))
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
				Request:    &http.Request{Method: http.MethodGet},
			}
			return resp, nil
		})

	// Expect an HTTP request to fetch the Rancher TLS additional CA secret from the managed cluster and return an HTTP 404
	managedClusterAdditionalTLSCAPath := fmt.Sprintf("/k8s/clusters/%s/api/v1/namespaces/cattle-system/secrets/tls-ca-additional", unitTestRancherClusterID)
	requestSenderMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(managedClusterAdditionalTLSCAPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte{}))
			resp := &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       r,
				Request:    &http.Request{Method: http.MethodGet},
			}
			return resp, nil
		})

	// Expect an HTTP request to fetch the Verrazzano system TLS CA secret from the managed cluster and return the secret
	managedClusterSystemCAPath := fmt.Sprintf("/k8s/clusters/%s/api/v1/namespaces/verrazzano-system/secrets/verrazzano-tls", unitTestRancherClusterID)
	requestSenderMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(managedClusterSystemCAPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			statusCode := http.StatusOK
			if len(caCertSecretData) == 0 {
				statusCode = http.StatusNotFound
			}
			r := io.NopCloser(bytes.NewReader([]byte(caCertSecretData)))
			resp := &http.Response{
				StatusCode: statusCode,
				Body:       r,
				Request:    &http.Request{Method: http.MethodGet},
			}
			return resp, nil
		})
}

// expectSyncCACertRancherK8sCalls asserts all of the expected calls on the Kubernetes client mock when sync'ing the managed cluster
// CA cert secret
func expectSyncCACertRancherK8sCalls(t *testing.T, k8sMock *mocks.MockClient, mockRequestSender *mocks.MockRequestSender, shouldSyncCACert bool) {
	asserts := assert.New(t)

	if shouldSyncCACert {
		// Expect K8S calls and admin token call to create new Rancher config
		expectRancherGetAdminTokenHTTPCall(t, mockRequestSender)

		expectRancherConfigK8sCalls(t, k8sMock)

		// Expect a call to get the CA cert secret for the managed cluster - return not found
		k8sMock.EXPECT().
			Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: "ca-secret-test"}), gomock.Not(gomock.Nil())).
			Return(errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoMultiClusterNamespace, Resource: "Secret"}, "ca-secret-test"))

		// Expect a call to create the CA cert secret for the managed cluster
		k8sMock.EXPECT().
			Create(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
				data := secret.Data[caCertSecretKey]
				asserts.NotZero(len(data), "Expected data in CA cert secret")
				return nil
			})

		// Expect a call to update the VMC with ca secret name
		k8sMock.EXPECT().
			Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersapi.VerrazzanoManagedCluster{}), gomock.Any()).
			DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
				asserts.Equal(vmc.Spec.CASecret, getCASecretName(vmc.Name), "CA secret name %s did not match", vmc.Spec.CASecret)
				return nil
			})
	}
}

// getScrapeConfig gets a representation of the vmc scrape configuration from the provided yaml
func getScrapeConfig(prometheusYaml string, name string) (*gabs.Container, error) {
	cfg, err := parsePrometheusConfig(prometheusYaml)
	if err != nil {
		return nil, err
	}
	scrapeConfigs := cfg.Path(scrapeConfigsKey).Children()
	return getJob(scrapeConfigs, name), nil
}

// getJob returns the scrape config job identified by the passed name from the slice of scrape configs
func getJob(scrapeConfigs []*gabs.Container, name string) *gabs.Container {
	var job *gabs.Container
	for _, scrapeConfig := range scrapeConfigs {
		jobName := scrapeConfig.Search(constants.PrometheusJobNameKey).Data()
		if jobName == name {
			job = scrapeConfig
			break
		}
	}
	return job
}

// getPrometheusHost returns the prometheus host for testManagedCluster
func getPrometheusHost() string {
	return "prometheus.vmi.system.default.1.2.3.4.nip.io"
}

// getPrometheusHost returns the prometheus host for testManagedCluster
func getCaCrt() string {
	// this is fake data
	return "    -----BEGIN CERTIFICATE-----\n" +
		"    MIIBiDCCAS6gAwIBAgIBADAKBggqhkjOPQQDAjA7MRwwGgYDVQQKExNkeW5hbWlj\n" +
		"    -----END CERTIFICATE-----"
}

func expectStatusUpdateReadyCondition(asserts *assert.Assertions, mock *mocks.MockClient, mockStatus *mocks.MockStatusWriter, expectReady corev1.ConditionStatus, msg string, assertManagedCACondition bool) {
	mock.EXPECT().Status().Return(mockStatus)
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersapi.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			readyFound := false
			managedCaFound := false
			readyConditionCount := 0
			managedCaConditionCount := 0
			for _, condition := range vmc.Status.Conditions {
				if condition.Type == clustersapi.ConditionReady {
					readyConditionCount++
					if condition.Status == expectReady {
						readyFound = true
						asserts.Contains(condition.Message, msg)
					}
				}
				if condition.Type == clustersapi.ConditionManagedCARetrieved {
					managedCaConditionCount++
					if condition.Status == expectReady {
						managedCaFound = true
						// asserts.Contains(condition.Message, msg)
					}
				}
			}
			asserts.True(readyFound, "Expected Ready condition on VMC not found")
			asserts.Equal(1, readyConditionCount, "Found more than one Ready condition")
			if assertManagedCACondition {
				asserts.True(managedCaFound, "Expected ManagedCARetrieved condition on VMC not found")
				asserts.Equal(1, managedCaConditionCount, "Found more than one ManagedCARetrieved condition")
			}

			return nil
		})
}

// validateScrapeConfig validates that a scrape config job has the expected field names and values
func validateScrapeConfig(t *testing.T, scrapeConfig *gabs.Container, caBasePath string, expectTLSConfig bool) {
	asserts := assert.New(t)
	asserts.NotNil(scrapeConfig)
	asserts.Equal(getPrometheusHost(),
		scrapeConfig.Search("static_configs", "0", "targets", "0").Data(), "No host entry found")
	asserts.NotEmpty(scrapeConfig.Search("basic_auth", "password").Data(), "No password")

	// assert that the verrazzano_cluster label is added in the static config
	asserts.Equal(testManagedCluster, scrapeConfig.Search(
		"static_configs", "0", "labels", "verrazzano_cluster").Data(),
		"Label verrazzano_cluster not set correctly in static_configs")

	// assert that the VMC job relabels verrazzano_cluster label to the right value
	asserts.Equal("verrazzano_cluster", scrapeConfig.Search("metric_relabel_configs", "0",
		"target_label").Data(),
		"metric_relabel_configs entry to post-process verrazzano_cluster label does not have expected target_label value")
	asserts.Equal(testManagedCluster, scrapeConfig.Search("metric_relabel_configs", "0",
		"replacement").Data(),
		"metric_relabel_configs entry to post-process verrazzano_cluster label does not have right replacement value")
	if expectTLSConfig {
		asserts.Equal("https", scrapeConfig.Path("scheme").Data(), "wrong scheme")
		asserts.Equal(caBasePath+"ca-test", scrapeConfig.Search("tls_config", "ca_file").Data(), "Wrong cert path")
	}
}
