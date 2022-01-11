// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"testing"
	"time"
)

// TestReconcilerSetupWithManager test the creation of the metrics trait reconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	scheme := newScheme()
	vzapi.AddToScheme(scheme)
	client := fake.NewFakeClientWithScheme(scheme)
	reconciler := newReconciler(client)

	mocker := gomock.NewController(t)
	manager := mocks.NewMockManager(mocker)
	manager.EXPECT().GetConfig().Return(&rest.Config{}).AnyTimes()
	manager.EXPECT().GetScheme().Return(scheme).AnyTimes()
	manager.EXPECT().GetLogger().Return(log.NullLogger{}).AnyTimes()
	manager.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	manager.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()

	err := reconciler.SetupWithManager(manager)
	assert.NoError(err, "Expected no error when setting up reconciler")
	mocker.Finish()
}

// TestGetMetricsTemplate tests the retrieval process of the metrics template
// GIVEN a metrics binding
// WHEN the function receives the binding
// THEN return the metrics template without error
func TestGetMetricsTemplate(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMetricsBinding := metricsBinding.DeepCopy()

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: testMetricsTemplateNamespace, Name: testMetricsTemplateName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, template *vzapi.MetricsTemplate) error {
			template.SetNamespace(metricsTemplate.Namespace)
			return nil
		})

	template, err := reconciler.getMetricsTemplate(localMetricsBinding)
	assert.NoError(err, "Expected no error getting the MetricsTemplate from the MetricsBinding")
	assert.NotNil(template)
}

// TestGetPromConfigMap tests the retrieval process of the Prometheus ConfigMap
// GIVEN a metrics binding
// WHEN the function receives the binding
// THEN then the Prometheus ConfigMap should be returned with no error
func TestGetPromConfigMap(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMetricsBinding := metricsBinding.DeepCopy()

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: constants.VerrazzanoSystemNamespace, Name: testConfigMapName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, cm *k8score.ConfigMap) error {
			cm.Data = configMap.Data
			return nil
		})

	template, err := reconciler.getPromConfigMap(localMetricsBinding)
	assert.NoError(err, "Expected no error getting the MetricsTemplate from the MetricsBinding")
	assert.NotNil(template)
}

// TestAddFinalizer tests the addition of a finalizer to the requested resource
// GIVEN a resource that is being created
// WHEN the function is called
// THEN the finalizer should be added
func TestAddFinalizer(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMetricsBinding := metricsBinding.DeepCopy()

	mock.EXPECT().Update(gomock.Any(), gomock.Not(gomock.Nil())).Return(nil)

	err := reconciler.addFinalizerIfRequired(context.TODO(), localMetricsBinding)
	assert.Equal([]string{finalizerName}, localMetricsBinding.GetFinalizers())
	assert.NoError(err, "Expected no error adding finalizer to the Deployment")
}

// TestRemoveFinalizer tests the removal of a finalizer on the requested resource
// GIVEN a resource that is being deleted has the metricsbinding finalizer
// WHEN the function is called
// THEN the finalizer should be removed
func TestRemoveFinalizer(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMetricsBinding := metricsBinding.DeepCopy()
	localMetricsBinding.SetDeletionTimestamp(&k8smeta.Time{Time: time.Now()})
	localMetricsBinding.SetFinalizers([]string{finalizerName})

	mock.EXPECT().Update(gomock.Any(), gomock.Not(gomock.Nil())).Return(nil)

	err := reconciler.removeFinalizerIfRequired(context.TODO(), localMetricsBinding)
	assert.Equal([]string{}, localMetricsBinding.GetFinalizers())
	assert.NoError(err, "Expected no error adding finalizer to the Deployment")
}

// TestCreateScrapeConfig tests the creation of the scrape config job
// GIVEN a directive to create a new config job
// WHEN the function is called
// THEN the scrape config should be created
func TestCreateScrapeConfig(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMetricsBinding := metricsBinding.DeepCopy()
	localMetricsTemplate := metricsTemplate.DeepCopy()

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")

	scrapeConfigTemplate, err := os.ReadFile("./testdata/scrape-config-template.yaml")
	assert.NoError(err, "Expected no error reading the scrape config test template")

	localMetricsTemplate.Spec.PrometheusConfig.ScrapeConfigTemplate = string(scrapeConfigTemplate)

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: testDeploymentNamespace, Name: testDeploymentName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, workload *unstructured.Unstructured) error {
			workload.SetUID(testUIDName)
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: testMetricsTemplateNamespace, Name: testMetricsTemplateName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, template *vzapi.MetricsTemplate) error {
			template.SetNamespace(metricsTemplate.Namespace)
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: constants.VerrazzanoSystemNamespace, Name: testConfigMapName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, cm *k8score.ConfigMap) error {
			cm.Data = configMap.Data
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Name: testDeploymentNamespace}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, namespace *k8score.Namespace) error {
			namespace.Labels = map[string]string{"istio-injection": "enabled"}
			namespace.Name = testDeploymentNamespace
			return nil
		})

	configMap, err = reconciler.createOrUpdateScrapeConfig(localMetricsBinding)
	assert.NoError(err, "Expected no error creating the scrape config")
	assert.True(strings.Contains(configMap.Data["prometheus.yml"], formatJobName(createJobName(localMetricsBinding))))
}

// TestUpdateScrapeConfig tests the updating of the scrape config job
// GIVEN a directive to update a config job
// WHEN the function is called
// THEN the scrape config should be updated
func TestUpdateScrapeConfig(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMetricsBinding := metricsBinding.DeepCopy()
	localMetricsTemplate := metricsTemplate.DeepCopy()

	localMetricsBinding.Spec.Workload = vzapi.Workload{
		Namespace: testExistsDeploymentNamespace,
		Name:      testExistsDeploymentName,
		TypeMeta: k8smeta.TypeMeta{
			Kind:       deploymentKind,
			APIVersion: deploymentGroup + "/" + deploymentVersion,
		},
	}
	localMetricsBinding.Namespace = testExistsDeploymentNamespace

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")

	scrapeConfigTemplate, err := os.ReadFile("./testdata/scrape-config-template.yaml")
	assert.NoError(err, "Expected no error reading the scrape config test template")

	localMetricsTemplate.Spec.PrometheusConfig.ScrapeConfigTemplate = string(scrapeConfigTemplate)

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: testMetricsTemplateNamespace, Name: testMetricsTemplateName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, template *vzapi.MetricsTemplate) error {
			template.SetNamespace(metricsTemplate.Namespace)
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: constants.VerrazzanoSystemNamespace, Name: testConfigMapName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, cm *k8score.ConfigMap) error {
			cm.Data = configMap.Data
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: testExistsDeploymentNamespace, Name: testExistsDeploymentName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, dep *unstructured.Unstructured) error {
			dep.SetLabels(deployment.GetLabels())
			dep.SetUID(testUIDName)
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Name: testDeploymentNamespace}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, namespace *k8score.Namespace) error {
			namespace.Labels = map[string]string{"istio-injection": "enabled"}
			namespace.Name = testDeploymentNamespace
			return nil
		})

	assert.True(strings.Contains(configMap.Data["prometheus.yml"], formatJobName(createJobName(localMetricsBinding))))
	configMap, err = reconciler.createOrUpdateScrapeConfig(localMetricsBinding)
	assert.NoError(err, "Expected no error updating the scrape config")
	assert.True(strings.Contains(configMap.Data["prometheus.yml"], formatJobName(createJobName(localMetricsBinding))))
}

// TestDeleteScrapeConfig tests the deletion of the scrape config job
// GIVEN a directive to delete a config job
// WHEN the function is called
// THEN the scrape config should be deleted
func TestDeleteScrapeConfig(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMetricsBinding := metricsBinding.DeepCopy()

	localMetricsBinding.OwnerReferences = []k8smeta.OwnerReference{
		{
			Kind:       deploymentKind,
			APIVersion: strings.Join([]string{deploymentGroup, deploymentVersion}, "/"),
			Name:       testExistsDeploymentName,
		},
	}
	localMetricsBinding.Spec.Workload = vzapi.Workload{
		Namespace: testExistsDeploymentNamespace,
		Name:      testExistsDeploymentName,
		TypeMeta: k8smeta.TypeMeta{
			Kind:       deploymentKind,
			APIVersion: deploymentGroup + "/" + deploymentVersion,
		},
	}
	localMetricsBinding.Namespace = testExistsDeploymentNamespace

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: constants.VerrazzanoSystemNamespace, Name: testConfigMapName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, cm *k8score.ConfigMap) error {
			cm.Data = configMap.Data
			return nil
		})

	assert.True(strings.Contains(configMap.Data["prometheus.yml"], formatJobName(createJobName(localMetricsBinding))))
	configMap, err = reconciler.deleteScrapeConfig(localMetricsBinding)
	assert.NoError(err, "Expected no error deleting the scrape config")
	assert.False(strings.Contains(configMap.Data["prometheus.yml"], formatJobName(createJobName(localMetricsBinding))))
}

// TestMutatePrometheusScrapeConfig tests the overarching mutation process given a resource
// GIVEN a request
// WHEN the controller receives the mutation request
// THEN verify the mutation process returns no error
func TestMutatePrometheusScrapeConfig(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMetricsBinding := metricsBinding.DeepCopy()
	localMetricsBinding.OwnerReferences = []k8smeta.OwnerReference{
		{
			Kind:       deploymentKind,
			APIVersion: strings.Join([]string{deploymentGroup, deploymentVersion}, "/"),
			Name:       testExistsDeploymentName,
		},
	}
	localMetricsBinding.Spec.Workload = vzapi.Workload{
		Namespace: testExistsDeploymentNamespace,
		Name:      testExistsDeploymentName,
		TypeMeta: k8smeta.TypeMeta{
			Kind:       deploymentKind,
			APIVersion: deploymentGroup + "/" + deploymentVersion,
		},
	}

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: constants.VerrazzanoSystemNamespace, Name: testConfigMapName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, cm *k8score.ConfigMap) error {
			cm.Data = configMap.Data
			return nil
		})

	mock.EXPECT().Update(gomock.Any(), gomock.Not(gomock.Nil)).Return(nil)

	err = reconciler.mutatePrometheusScrapeConfig(context.TODO(), localMetricsBinding, reconciler.deleteScrapeConfig)
	assert.NoError(err, "Expected no error mutating the scrape config")
}

// TestReconcileBindingCreateOrUpdate tests the reconciliation for create or update
// GIVEN an object and a request
// WHEN the reconciler processes the request
// THEN verify the process returns no error
func TestReconcileBindingCreateOrUpdate(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMetricsBinding := metricsBinding.DeepCopy()

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")

	mock.EXPECT().Update(gomock.Any(), localMetricsBinding).DoAndReturn(
		func(ctx context.Context, binding *vzapi.MetricsBinding) error {
			binding.Finalizers = append(metricsBinding.GetFinalizers(), finalizerName)
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: testMetricsTemplateNamespace, Name: testMetricsTemplateName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, template *vzapi.MetricsTemplate) error {
			template.SetNamespace(metricsTemplate.Namespace)
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: constants.VerrazzanoSystemNamespace, Name: testConfigMapName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, cm *k8score.ConfigMap) error {
			cm.Data = configMap.Data
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: testDeploymentNamespace, Name: testDeploymentName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, dep *unstructured.Unstructured) error {
			dep.SetLabels(deployment.GetLabels())
			dep.SetUID(testUIDName)
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Name: testDeploymentNamespace}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, namespace *k8score.Namespace) error {
			namespace.Labels = map[string]string{"istio-injection": "enabled"}
			namespace.Name = testDeploymentNamespace
			return nil
		})

	mock.EXPECT().Update(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).Return(nil)

	controllerResult, err := reconciler.reconcileBindingCreateOrUpdate(context.TODO(), localMetricsBinding)
	assert.NoError(err, "Expected no error reconciling the Deployment")
	assert.Equal(controllerResult, ctrl.Result{})
}

// TestReconcileBindingDelete tests the reconciliation for a deletion
// GIVEN an object and a request
// WHEN the reconciler processes the request
// THEN verify the process returns no error
func TestReconcileBindingDelete(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMetricsBinding := metricsBinding.DeepCopy()
	localMetricsBinding.OwnerReferences = []k8smeta.OwnerReference{
		{
			Kind:       deploymentKind,
			APIVersion: strings.Join([]string{deploymentGroup, deploymentVersion}, "/"),
			Name:       testExistsDeploymentName,
		},
	}
	localMetricsBinding.Spec.Workload = vzapi.Workload{
		Namespace: testExistsDeploymentNamespace,
		Name:      testExistsDeploymentName,
		TypeMeta: k8smeta.TypeMeta{
			Kind:       deploymentKind,
			APIVersion: deploymentGroup + "/" + deploymentVersion,
		},
	}

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: constants.VerrazzanoSystemNamespace, Name: testConfigMapName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, cm *k8score.ConfigMap) error {
			cm.Data = configMap.Data
			return nil
		})

	mock.EXPECT().Update(gomock.Any(), gomock.Not(gomock.Nil())).Return(nil)

	controllerResult, err := reconciler.reconcileBindingDelete(context.TODO(), localMetricsBinding)
	assert.NoError(err, "Expected no error reconciling the Deployment")
	assert.Equal(controllerResult, ctrl.Result{})
}

// TestCreateDeployment tests the creation process of the metrics template
// GIVEN a request
// WHEN the controller reconciles the request
// THEN verify the reconciliation has occurred no error is returned
func TestCreateDeployment(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMetricsBinding := metricsBinding.DeepCopy()

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")

	mock.EXPECT().Update(gomock.Any(), localMetricsBinding).DoAndReturn(
		func(ctx context.Context, binding *vzapi.MetricsBinding) error {
			binding.Finalizers = append(binding.GetFinalizers(), finalizerName)
			return nil
		})

	mock.EXPECT().Update(gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: testMetricsBindingNamespace, Name: testMetricsBindingName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, binding *vzapi.MetricsBinding) error {
			binding.Spec = localMetricsBinding.Spec
			binding.OwnerReferences = localMetricsBinding.OwnerReferences
			binding.ObjectMeta = localMetricsBinding.ObjectMeta
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: testMetricsTemplateNamespace, Name: testMetricsTemplateName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, template *vzapi.MetricsTemplate) error {
			template.SetNamespace(metricsTemplate.Namespace)
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: constants.VerrazzanoSystemNamespace, Name: testConfigMapName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, cm *k8score.ConfigMap) error {
			cm.Data = configMap.Data
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: testDeploymentNamespace, Name: testDeploymentName}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, dep *unstructured.Unstructured) error {
			dep.SetLabels(deployment.GetLabels())
			dep.SetUID(testUIDName)
			return nil
		})

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Name: testDeploymentNamespace}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, namespace *k8score.Namespace) error {
			namespace.Labels = map[string]string{"istio-injection": "enabled"}
			namespace.Name = testDeploymentNamespace
			return nil
		})

	namespacedName := types.NamespacedName{Namespace: testMetricsBindingNamespace, Name: testMetricsBindingName}
	request := ctrl.Request{NamespacedName: namespacedName}

	result, err := reconciler.Reconcile(request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	//_ = clientgoscheme.AddToScheme(scheme)
	k8sapps.AddToScheme(scheme)
	//	vzapi.AddToScheme(scheme)
	k8score.AddToScheme(scheme)
	//	certapiv1alpha2.AddToScheme(scheme)
	k8net.AddToScheme(scheme)
	return scheme
}

// newReconciler creates a new reconciler for testing
// c - The Kerberos client to inject into the reconciler
func newReconciler(c client.Client) Reconciler {
	log := ctrl.Log.WithName("test")
	scheme := newScheme()
	reconciler := Reconciler{
		Client:  c,
		Log:     log,
		Scheme:  scheme,
		Scraper: "istio-system/prometheus",
	}
	return reconciler
}
