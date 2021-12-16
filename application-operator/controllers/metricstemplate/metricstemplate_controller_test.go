// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstemplate

import (
	"context"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// TestGetResourceFromUID test the retrieval of resources from UID
// GIVEN a resource type and UID
// WHEN the function is called
// THEN populate the template object with found object
func TestGetResourceFromUID(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	configMap := k8score.ConfigMap{
		ObjectMeta: k8smetav1.ObjectMeta{UID: types.UID(testCMUID)},
	}
	expectListObject(&configMap, mock)

	metricsTemplate := v1alpha1.MetricsTemplate{
		ObjectMeta: k8smetav1.ObjectMeta{UID: types.UID(testMTUID)},
	}
	expectListObject(&metricsTemplate, mock)

	newConfigMap := configMap.DeepCopy()
	newMetricsTemplate := metricsTemplate.DeepCopy()
	err := reconciler.getResourceFromUID(context.TODO(), newConfigMap, testCMUID)
	assert.NoError(err, "Expected no error when getting ConfigMap from UID")
	assert.Equal(configMap, *newConfigMap)
	err = reconciler.getResourceFromUID(context.TODO(), newMetricsTemplate, testMTUID)
	assert.NoError(err, "Expected no error when getting MetricsTemplate from UID")
	assert.Equal(metricsTemplate, *newMetricsTemplate)
}

// TestGetRequestedResource test the retrieval of request resource
// GIVEN a resource type and namespaced name
// WHEN the function is called
// THEN populate the template object with found object
func TestGetRequestedResource(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	unstructuredDeployment := unstructured.Unstructured{}
	unstructuredDeployment.SetUID(testExistsDeploymentUID)

	namespacedName := types.NamespacedName{Namespace: testDeploymentNamespace, Name: testDeploymentName}
	mock.EXPECT().Get(gomock.Any(), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, name types.NamespacedName, object *unstructured.Unstructured) error {
			object.Object = unstructuredDeployment.Object
			return nil
		})

	newUnstructuredDeployment, err := reconciler.getRequestedResource(namespacedName)
	assert.NoError(err, "Expected no error retrieving the Deployment")
	assert.Equal(unstructuredDeployment, *newUnstructuredDeployment)
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

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")
	deployment := k8sapps.Deployment{
		ObjectMeta: k8smetav1.ObjectMeta{
			Namespace: testDeploymentNamespace,
			Name:      testDeploymentName,
			Labels: map[string]string{
				"app.verrazzano.io/metrics-template-uid": testMTUID,
			},
			UID: testDeploymentUID,
		},
	}
	unstructuredDeploymentMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
	assert.NoError(err, "Expected no error creating unstructured Deployment")
	unstructuredDeployment := unstructured.Unstructured{Object: unstructuredDeploymentMap}

	scrapeConfigTemplate, err := os.ReadFile("./testdata/scrape-config-template.yaml")
	assert.NoError(err, "Expected no error reading the scrape config test template")
	metricsTemplate := v1alpha1.MetricsTemplate{
		ObjectMeta: k8smetav1.ObjectMeta{UID: types.UID(testMTUID)},
		Spec: v1alpha1.MetricsTemplateSpec{
			PrometheusConfig: v1alpha1.PrometheusConfig{
				ScrapeConfigTemplate: string(scrapeConfigTemplate),
			},
		},
	}
	expectListObject(&metricsTemplate, mock)

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Name: testDeploymentNamespace, Namespace: constants.DefaultNamespace}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, namespaceUnstructured *unstructured.Unstructured) error {
			namespace := k8score.Namespace{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: testDeploymentNamespace,
					Labels: map[string]string{
						"istio-injection": "enabled",
					},
				},
			}
			namespaceUnstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&namespace)
			if err != nil {
				return err
			}
			namespaceUnstructured.Object = namespaceUnstructuredMap

			return nil
		})

	namespacedName := types.NamespacedName{Namespace: deployment.Namespace, Name: deployment.Name}
	err = reconciler.createOrUpdateScrapeConfig(configMap, namespacedName, &unstructuredDeployment)
	assert.NoError(err, "Expected no error creating the scrape config")
	assert.True(strings.Contains(configMap.Data["prometheus.yml"], formatJobName(createJobName(namespacedName, deployment.GetUID()))))
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

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")
	deployment := k8sapps.Deployment{
		ObjectMeta: k8smetav1.ObjectMeta{
			Namespace: testExistsDeploymentNamespace,
			Name:      testExistsDeploymentName,
			Labels: map[string]string{
				"app.verrazzano.io/metrics-template-uid": testMTUID,
			},
			UID: testExistsDeploymentUID,
		},
	}
	unstructuredDeploymentMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
	assert.NoError(err, "Expected no error creating unstructured Deployment")
	unstructuredDeployment := unstructured.Unstructured{Object: unstructuredDeploymentMap}

	scrapeConfigTemplate, err := os.ReadFile("./testdata/scrape-config-template.yaml")
	assert.NoError(err, "Expected no error reading the scrape config test template")
	metricsTemplate := v1alpha1.MetricsTemplate{
		ObjectMeta: k8smetav1.ObjectMeta{UID: types.UID(testMTUID)},
		Spec: v1alpha1.MetricsTemplateSpec{
			PrometheusConfig: v1alpha1.PrometheusConfig{
				ScrapeConfigTemplate: string(scrapeConfigTemplate),
			},
		},
	}
	expectListObject(&metricsTemplate, mock)

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(client.ObjectKey{Name: testExistsDeploymentNamespace, Namespace: constants.DefaultNamespace}), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, namespaceUnstructured *unstructured.Unstructured) error {
			namespace := k8score.Namespace{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: testExistsDeploymentNamespace,
					Labels: map[string]string{
						"istio-injection": "enabled",
					},
				},
			}
			namespaceUnstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&namespace)
			if err != nil {
				return err
			}
			namespaceUnstructured.Object = namespaceUnstructuredMap

			return nil
		})

	namespacedName := types.NamespacedName{Namespace: deployment.Namespace, Name: deployment.Name}
	assert.True(strings.Contains(configMap.Data["prometheus.yml"], formatJobName(createJobName(namespacedName, deployment.GetUID()))))
	err = reconciler.createOrUpdateScrapeConfig(configMap, namespacedName, &unstructuredDeployment)
	assert.NoError(err, "Expected no error updating the scrape config")
	assert.True(strings.Contains(configMap.Data["prometheus.yml"], formatJobName(createJobName(namespacedName, deployment.GetUID()))))
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

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")
	unstructuredDeployment := unstructured.Unstructured{}
	unstructuredDeployment.SetUID(testExistsDeploymentUID)

	namespacedName := types.NamespacedName{Namespace: testExistsDeploymentNamespace, Name: testExistsDeploymentName}
	assert.True(strings.Contains(configMap.Data["prometheus.yml"], formatJobName(createJobName(namespacedName, testExistsDeploymentUID))))
	err = reconciler.deleteScrapeConfig(configMap, namespacedName, &unstructuredDeployment)
	assert.NoError(err, "Expected no error deleting the scrape config")
	assert.False(strings.Contains(configMap.Data["prometheus.yml"], formatJobName(createJobName(namespacedName, testExistsDeploymentUID))))
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

	unstructuredDeployment := unstructured.Unstructured{}
	unstructuredDeployment.SetUID(testExistsDeploymentUID)
	unstructuredDeployment.SetLabels(map[string]string{"app.verrazzano.io/metrics-prometheus-configmap-uid": testCMUID})

	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Expected no error creating the ConfigMap from the test file")
	configMap.SetUID(testCMUID)
	expectListObject(configMap, mock)

	mock.EXPECT().Update(gomock.Any(), gomock.Not(gomock.Nil)).Return(nil)

	err = reconciler.mutatePrometheusScrapeConfig(context.TODO(), &unstructuredDeployment, reconciler.deleteScrapeConfig)
	assert.NoError(err, "Expected no error mutating the scrape config")
}

// TestReconcileTraitCreateOrUpdate tests the reconciliation for create or update
// GIVEN an object and a request
// WHEN the reconciler processes the request
// THEN verify the process returns no error
func TestReconcileTraitCreateOrUpdate(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	deployment := k8sapps.Deployment{
		ObjectMeta: k8smetav1.ObjectMeta{
			Namespace: testExistsDeploymentNamespace,
			Name:      testExistsDeploymentName,
			Labels: map[string]string{
				"app.verrazzano.io/metrics-template-uid": testMTUID,
			},
			UID: testExistsDeploymentUID,
		},
	}
	unstructuredDeploymentMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
	assert.NoError(err, "Expected no error creating unstructured Deployment")
	unstructuredDeployment := unstructured.Unstructured{Object: unstructuredDeploymentMap}

	controllerResult, err := reconciler.reconcileTraitCreateOrUpdate(context.TODO(), &unstructuredDeployment)
	assert.NoError(err, "Expected no error reconciling the Deployment")
	assert.Equal(controllerResult, ctrl.Result{})
}

// TestReconcileTraitDelete tests the reconciliation for a deletion
// GIVEN an object and a request
// WHEN the reconciler processes the request
// THEN verify the process returns no error
func TestReconcileTraitDelete(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	deployment := k8sapps.Deployment{
		ObjectMeta: k8smetav1.ObjectMeta{
			Namespace: testExistsDeploymentNamespace,
			Name:      testExistsDeploymentName,
			Labels: map[string]string{
				"app.verrazzano.io/metrics-template-uid": testMTUID,
			},
			UID: testExistsDeploymentUID,
		},
	}
	unstructuredDeploymentMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
	assert.NoError(err, "Expected no error creating unstructured Deployment")
	unstructuredDeployment := unstructured.Unstructured{Object: unstructuredDeploymentMap}

	controllerResult, err := reconciler.reconcileTraitDelete(context.TODO(), &unstructuredDeployment)
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

	namespacedName := types.NamespacedName{Namespace: testExistsDeploymentNamespace, Name: testDeploymentName}
	request := ctrl.Request{NamespacedName: namespacedName}

	deployment := k8sapps.Deployment{
		ObjectMeta: k8smetav1.ObjectMeta{
			Namespace: testExistsDeploymentNamespace,
			Name:      testExistsDeploymentName,
			Labels: map[string]string{
				"app.verrazzano.io/metrics-template-uid": testMTUID,
			},
			UID: testExistsDeploymentUID,
		},
	}
	unstructuredDeploymentMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
	assert.NoError(err, "Expected no error creating unstructured Deployment")
	unstructuredDeployment := unstructured.Unstructured{Object: unstructuredDeploymentMap}

	mock.EXPECT().Get(gomock.Any(), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, name types.NamespacedName, object *unstructured.Unstructured) error {
			object.Object = unstructuredDeployment.Object
			return nil
		})

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

// expectListObject returns a mock list call of an unstructured list based on the object passed in
func expectListObject(resource runtime.Object, mock *mocks.MockClient) {
	objectKind := resource.GetObjectKind()
	gvk := objectKind.GroupVersionKind()
	unstructuredList := unstructured.UnstructuredList{}
	unstructuredList.SetKind(gvk.Kind + "List")
	unstructuredList.SetAPIVersion(gvk.Version)

	mock.EXPECT().List(gomock.Any(), gomock.Eq(&unstructuredList)).DoAndReturn(
		func(ctx context.Context, object *unstructured.UnstructuredList) error {
			unstructuredObject, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&resource)
			if err != nil {
				return err
			}
			object.Items = []unstructured.Unstructured{{Object: unstructuredObject}}
			return nil
		})
}
