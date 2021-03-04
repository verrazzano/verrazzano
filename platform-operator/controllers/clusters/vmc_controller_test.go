// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	promconfig "github.com/prometheus/prometheus/config"
	vzk8s "github.com/verrazzano/verrazzano/platform-operator/internal/k8s"
	"gopkg.in/yaml.v2"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/rest"
	"testing"
	"time"

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

const kubeAdminData = `
apiEndpoints:
  oke-xyz:
    advertiseAddress: 1.2.3.4
    bindPort: 6443
`
const (
	tokenKey = "token"
	token    = "tokenData"
)

// TestCreateVMC tests the Reconcile method for the following use case
// GIVEN a request to reconcile an VerrazzanoManagedCluster resource
// WHEN a VerrazzanoManagedCluster resource has been applied
// THEN ensure all the objects are created
func TestCreateVMC(t *testing.T) {
	namespace := "verrazzano-mc"
	name := "test"
	saSecretName := "saSecret"
	promData := "prometheus:\n" +
		"  authpasswd: nRXlxXgMwN\n" +
		"  host: prometheus.vmi.system.default.152.67.141.181.xip.io\n" +
		"  cacrt: |\n" +
		"    -----BEGIN CERTIFICATE-----\n" +
		"    MIIBiDCCAS6gAwIBAgIBADAKBggqhkjOPQQDAjA7MRwwGgYDVQQKExNkeW5hbWlj\n" +
		"    -----END CERTIFICATE-----"
	clusterName := "cluster1"
	labels := map[string]string{"label1": "test"}
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	// Expect a call to get the VerrazzanoManagedCluster resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
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
			vmc.ClusterName = clusterName
			return nil
		})

	setupStandardCreationMockInteractions(mock, namespace, name, asserts, saSecretName, promData, clusterName)

	// following are calls from setupPrometheusScraper

	// Expect a call to get the prometheus secret - return return it
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getPrometheusSecretName()}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				"managed-cluster.yaml": []byte(promData),
			}
			return nil
		})

	// Expect a call to get the prometheus configmap and return a new one in this case (not testing existing entries)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "verrazzano-system", Name: "vmi-system-prometheus-config"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			configMap.Data = map[string]string{}
			return nil
		})

	// Expect a call to Update the configmap
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.UpdateOption) error {
			asserts.Len(configMap.Data, 2, "no data found")
			asserts.NotNil(configMap.Data["ca-cluster1"], "No cert entry found")
			prometheusYaml := configMap.Data["prometheus.yml"]
			asserts.NotNil(prometheusYaml, "No prometheus config yaml found")
			asserts.Contains(prometheusYaml, "prometheus.vmi.system.default.152.67.141.181.xip.io", "No host entry found")
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
	saSecretName := "saSecret"
	promData := "prometheus:\n" +
		"  authpasswd: nRXlxXgMwN\n" +
		"  host: prometheus.vmi.system.default.152.67.141.181.xip.io\n" +
		"  cacrt: |\n" +
		"    -----BEGIN CERTIFICATE-----\n" +
		"    MIIBiDCCAS6gAwIBAgIBADAKBggqhkjOPQQDAjA7MRwwGgYDVQQKExNkeW5hbWlj\n" +
		"    -----END CERTIFICATE-----"
	clusterName := "cluster1"
	labels := map[string]string{"label1": "test"}
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer setConfigFunc(getConfigFunc)
	setConfigFunc(fakeGetConfig)

	// Expect a call to get the VerrazzanoManagedCluster resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
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
			vmc.ClusterName = clusterName
			return nil
		})

	setupStandardCreationMockInteractions(mock, namespace, name, asserts, saSecretName, promData, clusterName)

	// following are calls from setupPrometheusScraper

	// Expect a call to get the prometheus secret - return return it
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getPrometheusSecretName()}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				"managed-cluster.yaml": []byte(promData),
			}
			return nil
		})

	// Expect a call to get the prometheus configmap and return a new one in this case (not testing existing entries)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "verrazzano-system", Name: "vmi-system-prometheus-config"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			// setup a scaled down existing scrape config entry for cluster1
			cfg := &promconfig.Config{ScrapeConfigs: []*promconfig.ScrapeConfig{
				{
					JobName: "cluster1",
					Scheme:  "http",
				},
			}}
			newConfig, _ := yaml.Marshal(cfg)
			configMap.Data = map[string]string{
				"prometheus.yml": string(newConfig),
			}
			return nil
		})

	// Expect a call to Update the configmap
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.UpdateOption) error {
			// check for the modified entry
			asserts.Len(configMap.Data, 2, "no data found")
			asserts.NotNil(configMap.Data["ca-cluster1"], "No cert entry found")
			prometheusYaml := configMap.Data["prometheus.yml"]
			asserts.NotNil(prometheusYaml, "No prometheus config yaml found")
			asserts.Contains(prometheusYaml, "prometheus.vmi.system.default.152.67.141.181.xip.io", "No host entry found")
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

func getPrometheusSecretName() string {
	return "cluster1-secret"
}

// setupStandardCreationMockInteractions performs the mock setup calls that do not change for creation test methods
func setupStandardCreationMockInteractions(mock *mocks.MockClient, namespace string, name string, asserts *assert.Assertions, saSecretName string, promData string, clusterName string) {
	// The following calls are used by syncServiceAccount

	// Expect a call to get the ServiceAccount - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: generateManagedResourceName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "ServiceAccount"}, generateManagedResourceName(name)))

	// Expect a call to create the ServiceAccount - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, serviceAccount *corev1.ServiceAccount, opts ...client.CreateOption) error {
			asserts.Equalf(namespace, serviceAccount.Namespace, "ServiceAccount namespace did not match")
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

	// The following calls are used by syncManagedRoleBinding

	// Expect a call to get the ClusterRoleBinding - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: generateManagedResourceName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "ServiceAccount"}, generateManagedResourceName(name)))

	// Expect a call to create the ClusterRoleBinding - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, binding *rbacv1.ClusterRoleBinding, opts ...client.CreateOption) error {
			asserts.Equalf(generateManagedResourceName(name), binding.Name, "ClusterRoleBinding name did not match")
			asserts.Equalf("verrazzano-managed-cluster", binding.RoleRef.Name, "ClusterRoleBinding roleref did not match")
			asserts.Equalf(generateManagedResourceName(name), binding.Subjects[0].Name, "Subject did not match")
			asserts.Equalf(namespace, binding.Subjects[0].Namespace, "Subject namespace did not match")
			return nil
		})

	// The following calls are used by syncRegistrationSecret

	// Expect a call to get the ServiceAccount, return one with the secret name set
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: generateManagedResourceName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, sa *corev1.ServiceAccount) error {
			sa.Secrets = []corev1.ObjectReference{{
				Name: saSecretName,
			}}
			return nil
		})

	// Expect a call to get the service token secret, return the secret with the token
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: saSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				tokenKey: []byte(token),
			}
			return nil
		})

	// Expect a call to get the kubeadmin configmap
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: vzk8s.KubeSystem, Name: vzk8s.KubeAdminConfig}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, cm *corev1.ConfigMap) error {
			cm.Data = map[string]string{
				vzk8s.ClusterStatusKey: kubeAdminData,
			}
			return nil
		})

	// Expect a call to get the registration secret - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: GetRegistrationSecretName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Secret"}, GetRegistrationSecretName(name)))

	// Expect a call to create the registration secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			clusterName, _ := secret.Data[managedClusterNameKey]
			asserts.Equalf(name, string(clusterName), "Incorrect cluster name in cluster secret ")
			return nil
		})

	// Expect a call to update the VerrazzanoManagedCluster registration secret name - return success
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			asserts.Equal(vmc.Spec.ClusterRegistrationSecret, GetRegistrationSecretName(name), "Registration name did not match")
			return nil
		})

	// The following calls are used by sync_manifest
	//
	// Expect a call to get the registration secret
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: GetRegistrationSecretName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Data = map[string][]byte{
				kubeconfigKey:         []byte(promData),
				managedClusterNameKey: []byte(clusterName),
			}
			return nil
		})

	// Expect a call to get the manifest secret - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: GetManifestSecretName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Secret"}, GetManifestSecretName(name)))

	// Expect a call to create the manifest secret
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			data, _ := secret.Data[yamlKey]
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
				DeletionTimestamp: &metav1.Time{},
				Labels:            labels}
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
