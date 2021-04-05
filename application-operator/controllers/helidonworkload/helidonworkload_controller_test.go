// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonworkload

import (
	"bufio"
	"context"
	"io/ioutil"
	"strings"
	"testing"

	oamapi "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/loggingscope"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

const namespace = "unit-test-namespace"

// TestReconcilerSetupWithManager test the creation of the VerrazzanoHelidonWorkload reconciler.
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

// TestReconcileWorkloadNotFound tests reconciling a VerrazzanoHelidonWorkload when the workload
// cannot be fetched. This happens when the workload has been deleted by the OAM runtime.
// GIVEN a VerrazzanoHelidonWorkload resource has been deleted
// WHEN the controller Reconcile function is called and we attempt to fetch the workload
// THEN return success from the controller as there is nothing more to do
func TestReconcileWorkloadNotFound(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileFetchWorkloadError tests reconciling a VerrazzanoHelidonWorkload when the workload
// cannot be fetched due to an unexpected error.
// GIVEN a VerrazzanoHelidonWorkload resource has been created
// WHEN the controller Reconcile function is called and we attempt to fetch the workload and get an error
// THEN return the error
func TestReconcileFetchWorkloadError(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			return k8serrors.NewBadRequest("An error has occurred")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.Equal("An error has occurred", err.Error())
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateCoherence tests the basic happy path of reconciling a VerrazzanoHelidonWorkload. We
// expect to write out a Deployment and Service but we aren't adding logging or any other scopes or traits.
// GIVEN a VerrazzanoHelidonWorkload resource is created
// WHEN the controller Reconcile function is called
// THEN expect a Deployment and Service to be written
func TestReconcileCreateHelidon(t *testing.T) {
	assert := asserts.New(t)
	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}
	helidonTestContainerPort := v1.ContainerPort{
		ContainerPort: 8080,
		Name:          "http",
	}
	helidonTestContainer := v1.Container{
		Name:  "hello-helidon-container-new",
		Image: "ghcr.io/verrazzano/example-helidon-greet-app-v1:0.1.10-3-20201016220428-56fb4d4",
		Ports: []v1.ContainerPort{
			helidonTestContainerPort,
		},
	}
	deploymentTemplate := &vzapi.DeploymentTemplate{
		Metadata: metav1.ObjectMeta{
			Name:      "hello-helidon-deployment-new",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "hello-helidon-deploy-new",
			},
		},
		PodSpec: v1.PodSpec{
			Containers: []v1.Container{
				helidonTestContainer,
			},
		},
	}
	// expect call to fetch existing deployment
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "hello-helidon-deployment-new"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
	// expect a call to fetch the application configuration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-app-config"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appconf *oamapi.ApplicationConfiguration) error {
			appconf.Namespace = name.Namespace
			appconf.Name = name.Name
			appconf.APIVersion = oamapi.SchemeGroupVersion.String()
			appconf.Kind = oamapi.ApplicationConfigurationKind
			appconf.Spec.Components = []oamapi.ApplicationConfigurationComponent{{ComponentName: "unit-test-component"}}
			return nil
		})
	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			workload.Spec.DeploymentTemplate = *deploymentTemplate
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoHelidonWorkload"
			workload.Namespace = namespace
			return nil
		})
	// expect a call to fetch the application configuration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-app-config"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appconf *oamapi.ApplicationConfiguration) error {
			appconf.Namespace = name.Namespace
			appconf.Name = name.Name
			appconf.APIVersion = oamapi.SchemeGroupVersion.String()
			appconf.Kind = oamapi.ApplicationConfigurationKind
			appconf.Spec.Components = []oamapi.ApplicationConfigurationComponent{{ComponentName: "unit-test-component"}}
			return nil
		})
	// expect a call to create the Deployment
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deploy *appsv1.Deployment, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(deploymentAPIVersion, deploy.APIVersion)
			assert.Equal(deploymentKind, deploy.Kind)
			// make sure the OAM component and app name labels were copied
			assert.Equal(map[string]string{"app": "hello-helidon-deploy-new", "app.oam.dev/component": "unit-test-component", "app.oam.dev/name": "unit-test-app-config"}, deploy.GetLabels())
			assert.Equal([]v1.Container{
				helidonTestContainer,
			}, deploy.Spec.Template.Spec.Containers)
			return nil
		})
	// expect a call to create the Service
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, service *v1.Service, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(serviceAPIVersion, service.APIVersion)
			assert.Equal(serviceKind, service.Kind)
			return nil
		})
	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoHelidonWorkload) error {
			assert.Len(workload.Status.Resources, 2)
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateVerrazzanoHelidonWorkloadWithLoggingScope tests the basic happy path of reconciling a VerrazzanoHelidonWorkload that has a logging scope.
// We expect to write out a Deployment, Service and Configmap.
// GIVEN a VerrazzanoHelidonWorkload resource is created
// AND that the workload has a logging scope applied
// WHEN the controller Reconcile function is called
// THEN expect a Deployment, Service and Configmap to be written
func TestReconcileCreateVerrazzanoHelidonWorkloadWithLoggingScope(t *testing.T) {
	assert := asserts.New(t)
	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	testNamespace := "test-namespace"
	loggingSecretName := "test-secret-name"

	fluentdImage := "unit-test-image:latest"
	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := loggingscope.DefaultFluentdImage
	loggingscope.DefaultFluentdImage = fluentdImage
	defer func() { loggingscope.DefaultFluentdImage = initialDefaultFluentdImage }()

	params := map[string]string{
		"##APPCONF_NAME##":          "test-appconf",
		"##APPCONF_NAMESPACE##":     testNamespace,
		"##COMPONENT_NAME##":        "test-component",
		"##SCOPE_NAME##":            "test-scope",
		"##SCOPE_NAMESPACE##":       testNamespace,
		"##INGEST_URL##":            "http://test-ingest-host:9200",
		"##INGEST_SECRET_NAME##":    loggingSecretName,
		"##FLUENTD_IMAGE##":         "test-fluentd-image-name",
		"##WORKLOAD_APIVER##":       "oam.verrazzano.io/v1alpha1",
		"##WORKLOAD_KIND##":         "VerrazzanoHelidonWorkload",
		"##WORKLOAD_NAME##":         "test-workload-name",
		"##WORKLOAD_NAMESPACE##":    testNamespace,
		"##DEPLOYMENT_NAME##":       "test-deployment",
		"##CONTAINER_NAME##":        "test-container",
		"##CONTAINER_IMAGE##":       "test-container-image",
		"##CONTAINER_PORT_NAME##":   "http",
		"##CONTAINER_PORT_NUMBER##": "8080",
		"##LOGGING_SCOPE_NAME##":    "test-logging-scope",
		"##INGRESS_TRAIT_NAME##":    "test-ingress-trait",
		"##INGRESS_TRAIT_PATH##":    "/test-ingress-path",
	}
	// expect call to fetch existing deployment
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-deployment"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
	// expect a call to fetch the application configuration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-appconf"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appconf *oamapi.ApplicationConfiguration) error {
			assert.NoError(updateObjectFromYAMLTemplate(appconf, "test/templates/helidon_appconf_with_ingress_and_logging.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			assert.NoError(updateObjectFromYAMLTemplate(workload, "test/templates/helidon_workload.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the application configuration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-appconf"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appconf *oamapi.ApplicationConfiguration) error {
			assert.NoError(updateObjectFromYAMLTemplate(appconf, "test/templates/helidon_appconf_with_ingress_and_logging.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the logging scope
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-logging-scope"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, scope *vzapi.LoggingScope) error {
			assert.NoError(updateObjectFromYAMLTemplate(scope, "test/templates/logging_scope.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the Fluentd config and return a not found error
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "fluentd-config-helidon-test-deployment"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.GroupResource{Group: "", Resource: "configmap"}, "fluentd-config-helidon-test-deployment")).
		Times(1)

	// expect a call to get the Elasticsearch secret in app namespace - return not found
	testLoggingSecretFullName := types.NamespacedName{Namespace: testNamespace, Name: loggingSecretName}
	cli.EXPECT().
		Get(gomock.Any(), testLoggingSecretFullName, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.ParseGroupResource("v1.Secret"), loggingSecretName))

	// expect a call to create an empty Elasticsearch secret in app namespace (default behavior, so
	// that Fluentd volume mount works)
	cli.EXPECT().
		Create(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *v1.Secret, options *client.CreateOptions) error {
			asserts.Equal(t, testNamespace, sec.Namespace)
			asserts.Equal(t, loggingSecretName, sec.Name)
			asserts.Nil(t, sec.Data)
			asserts.Equal(t, client.CreateOptions{}, *options)
			return nil
		})

	// expect a call to create a Configmap
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, config *v1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(testNamespace, config.Namespace)
			assert.Equal("fluentd-config-helidon-test-deployment", config.Name)
			assert.Len(config.Data, 1)
			assert.Contains(config.Data["fluentd.conf"], "label")
			return nil
		}).Times(1)

	// expect a call to create the Deployment
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deploy *appsv1.Deployment, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(deploymentAPIVersion, deploy.APIVersion)
			assert.Equal(deploymentKind, deploy.Kind)
			// make sure the OAM component and app name labels were copied
			assert.Equal(map[string]string{"app.oam.dev/component": "test-component", "app.oam.dev/name": "test-appconf"}, deploy.GetLabels())
			assert.Equal(params["##CONTAINER_NAME##"], deploy.Spec.Template.Spec.Containers[0].Name)
			assert.Len(deploy.Spec.Template.Spec.Containers, 2, "Expect 4 containers: app+sidecar")

			// The app container should be unmodified for the Helidon use case.
			c, found := findContainer(deploy.Spec.Template.Spec.Containers, "test-container")
			assert.True(found, "Expected to find app container test-container")
			assert.Equal(c.Image, "test-container-image")
			assert.Len(c.Ports, 1)
			assert.Equal(c.Ports[0].Name, "http")
			assert.Equal(c.Ports[0].ContainerPort, int32(8080))
			assert.Nil(c.VolumeMounts, "Expected app container to have no volume mounts")

			// The side car should have env vars setup correctly and 4 mount volumes.
			c, found = findContainer(deploy.Spec.Template.Spec.Containers, "fluentd")
			assert.True(found, "Expected to find sidecar container test-container")
			assert.Equal(fluentdImage, c.Image)
			assert.Len(c.Env, 10, "Expect sidecar container to have 10 env vars")
			assertEnvVar(t, c.Env, "WORKLOAD_NAME", "test-deployment")
			assertEnvVarFromField(t, c.Env, "APP_CONF_NAME", "metadata.labels['app.oam.dev/name']")
			assertEnvVarFromField(t, c.Env, "COMPONENT_NAME", "metadata.labels['app.oam.dev/component']")
			assertEnvVar(t, c.Env, "ELASTICSEARCH_URL", "http://test-ingest-host:9200")
			assertEnvVarFromSecret(t, c.Env, "ELASTICSEARCH_USER", "test-secret-name", "username")
			assertEnvVarFromSecret(t, c.Env, "ELASTICSEARCH_PASSWORD", "test-secret-name", "password")
			assert.Len(c.VolumeMounts, 4, "Expect sidecar container to have 4 volume mounts")
			assertVolumeMount(t, c.VolumeMounts, "fluentd-config-volume", "/fluentd/etc/fluentd.conf", "fluentd.conf", true)
			assertVolumeMount(t, c.VolumeMounts, "secret-volume", "/fluentd/secret", "", true)
			assertVolumeMount(t, c.VolumeMounts, "varlog", "/var/log", "", true)
			assertVolumeMount(t, c.VolumeMounts, "datadockercontainers", "/u01/data/docker/containers", "", true)
			return nil
		})
	// expect a call to create the Service
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, service *v1.Service, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(serviceAPIVersion, service.APIVersion)
			assert.Equal(serviceKind, service.Kind)
			return nil
		})
	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoHelidonWorkload) error {
			assert.Len(workload.Status.Resources, 2)
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, "test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateVerrazzanoHelidonWorkloadWithMultipleContainersAndLoggingScope tests correct sidecar setup for a workload with multiple containers.
// GIVEN a VerrazzanoHelidonWorkload resource is created
// AND that the workload has multiple containers
// AND that the workload has a logging scope applied
// WHEN the controller Reconcile function is called
// THEN expect a Deployment, Service and Configmap to be written
// AND expect that each application container has an associated logging sidecar container
func TestReconcileCreateVerrazzanoHelidonWorkloadWithMultipleContainersAndLoggingScope(t *testing.T) {
	assert := asserts.New(t)
	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	testNamespace := "test-namespace"
	loggingSecretName := "test-secret-name"

	fluentdImage := "unit-test-image:latest"
	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := loggingscope.DefaultFluentdImage
	loggingscope.DefaultFluentdImage = fluentdImage
	defer func() { loggingscope.DefaultFluentdImage = initialDefaultFluentdImage }()

	params := map[string]string{
		"##APPCONF_NAME##":            "test-appconf",
		"##APPCONF_NAMESPACE##":       testNamespace,
		"##COMPONENT_NAME##":          "test-component",
		"##SCOPE_NAME##":              "test-scope",
		"##SCOPE_NAMESPACE##":         testNamespace,
		"##INGEST_URL##":              "http://test-ingest-host:9200",
		"##INGEST_SECRET_NAME##":      loggingSecretName,
		"##FLUENTD_IMAGE##":           "test-fluentd-image-name",
		"##WORKLOAD_APIVER##":         "oam.verrazzano.io/v1alpha1",
		"##WORKLOAD_KIND##":           "VerrazzanoHelidonWorkload",
		"##WORKLOAD_NAME##":           "test-workload-name",
		"##WORKLOAD_NAMESPACE##":      testNamespace,
		"##DEPLOYMENT_NAME##":         "test-deployment",
		"##CONTAINER_NAME_1##":        "test-container-1",
		"##CONTAINER_IMAGE_1##":       "test-container-image-1",
		"##CONTAINER_PORT_NAME_1##":   "http1",
		"##CONTAINER_PORT_NUMBER_1##": "8081",
		"##CONTAINER_NAME_2##":        "test-container-2",
		"##CONTAINER_IMAGE_2##":       "test-container-image-2",
		"##CONTAINER_PORT_NAME_2##":   "http2",
		"##CONTAINER_PORT_NUMBER_2##": "8082",
		"##LOGGING_SCOPE_NAME##":      "test-logging-scope",
		"##INGRESS_TRAIT_NAME##":      "test-ingress-trait",
		"##INGRESS_TRAIT_PATH##":      "/test-ingress-path",
	}
	// expect call to fetch existing deployment
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-deployment"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
	// expect a call to fetch the application configuration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-appconf"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appconf *oamapi.ApplicationConfiguration) error {
			assert.NoError(updateObjectFromYAMLTemplate(appconf, "test/templates/helidon_appconf_with_ingress_and_logging.yaml", params))
			return nil
		}).Times(2)
	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			assert.NoError(updateObjectFromYAMLTemplate(workload, "test/templates/helidon_workload_multi_container.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the logging scope
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-logging-scope"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, scope *vzapi.LoggingScope) error {
			assert.NoError(updateObjectFromYAMLTemplate(scope, "test/templates/logging_scope.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the Fluentd config and return a not found error
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "fluentd-config-helidon-test-deployment"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.GroupResource{Group: "", Resource: "configmap"}, "fluentd-config-helidon-test-deployment")).
		Times(1)
	// expect a call to get the Elasticsearch secret in app namespace - return not found
	testLoggingSecretFullName := types.NamespacedName{Namespace: testNamespace, Name: loggingSecretName}
	cli.EXPECT().
		Get(gomock.Any(), testLoggingSecretFullName, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.ParseGroupResource("v1.Secret"), loggingSecretName))

	// expect a call to create an empty Elasticsearch secret in app namespace (default behavior, so
	// that Fluentd volume mount works)
	cli.EXPECT().
		Create(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *v1.Secret, options *client.CreateOptions) error {
			asserts.Equal(t, testNamespace, sec.Namespace)
			asserts.Equal(t, loggingSecretName, sec.Name)
			asserts.Nil(t, sec.Data)
			asserts.Equal(t, client.CreateOptions{}, *options)
			return nil
		})
	// expect a call to create a Configmap
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, config *v1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(testNamespace, config.Namespace)
			assert.Equal("fluentd-config-helidon-test-deployment", config.Name)
			assert.Len(config.Data, 1)
			assert.Contains(config.Data["fluentd.conf"], "label")
			return nil
		}).Times(1)

	// expect a call to create the Deployment
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deploy *appsv1.Deployment, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(deploymentAPIVersion, deploy.APIVersion)
			assert.Equal(deploymentKind, deploy.Kind)
			// make sure the OAM component and app name labels were copied
			assert.Equal(map[string]string{"app.oam.dev/component": "test-component", "app.oam.dev/name": "test-appconf"}, deploy.GetLabels())
			assert.Equal(params["##CONTAINER_NAME_1##"], deploy.Spec.Template.Spec.Containers[0].Name)

			// There should be 4 containers because a sidecar will be added for each original container.
			assert.Len(deploy.Spec.Template.Spec.Containers, 3, "Expect 3 containers.  Two app and one sidecar.")

			// The first app container should be unmodified.
			c, found := findContainer(deploy.Spec.Template.Spec.Containers, "test-container-1")
			assert.True(found, "Expected to find app container test-container")
			assert.Equal(c.Image, "test-container-image-1")
			assert.Len(c.Ports, 1)
			assert.Equal(c.Ports[0].Name, "http1")
			assert.Equal(c.Ports[0].ContainerPort, int32(8081))
			assert.Nil(c.VolumeMounts, "Expected app container to have no volume mounts")

			// The second app container should be unmodified.
			c, found = findContainer(deploy.Spec.Template.Spec.Containers, "test-container-2")
			assert.True(found, "Expected to find app container test-container")
			assert.Equal(c.Image, "test-container-image-2")
			assert.Len(c.Ports, 1)
			assert.Equal(c.Ports[0].Name, "http2")
			assert.Equal(c.Ports[0].ContainerPort, int32(8082))
			assert.Nil(c.VolumeMounts, "Expected app container to have no volume mounts")

			// The sidecar should have the correct env vars.
			c, found = findContainer(deploy.Spec.Template.Spec.Containers, "fluentd")
			assert.True(found, "Expected to find sidecar container fluentd")
			assert.Equal(c.Name, "fluentd")
			assert.Equal(fluentdImage, c.Image)
			assert.Len(c.Env, 10, "Expect sidecar container to have 10 env vars")
			assertEnvVar(t, c.Env, "WORKLOAD_NAME", "test-deployment")
			assertEnvVarFromField(t, c.Env, "APP_CONF_NAME", "metadata.labels['app.oam.dev/name']")
			assertEnvVarFromField(t, c.Env, "COMPONENT_NAME", "metadata.labels['app.oam.dev/component']")
			assertEnvVar(t, c.Env, "ELASTICSEARCH_URL", "http://test-ingest-host:9200")
			assertEnvVarFromSecret(t, c.Env, "ELASTICSEARCH_USER", loggingSecretName, "username")
			assertEnvVarFromSecret(t, c.Env, "ELASTICSEARCH_PASSWORD", loggingSecretName, "password")
			assert.Len(c.VolumeMounts, 4, "Expect sidecar container to have 4 volume mounts")
			assertVolumeMount(t, c.VolumeMounts, "fluentd-config-volume", "/fluentd/etc/fluentd.conf", "fluentd.conf", true)
			assertVolumeMount(t, c.VolumeMounts, "secret-volume", "/fluentd/secret", "", true)
			assertVolumeMount(t, c.VolumeMounts, "varlog", "/var/log", "", true)
			assertVolumeMount(t, c.VolumeMounts, "datadockercontainers", "/u01/data/docker/containers", "", true)

			return nil
		})
	// expect a call to create the Service
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, service *v1.Service, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(serviceAPIVersion, service.APIVersion)
			assert.Equal(serviceKind, service.Kind)
			return nil
		})
	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoHelidonWorkload) error {
			assert.Len(workload.Status.Resources, 2)
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, "test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileAlreadyExistsUpgrade tests reconciling a VerrazzanoHelidonWorkload when the Deployment CR already
// exists and the upgrade version specified in the labels differs from the current upgrade version.
// This should result in the latest Fluentd image being pulled from the env.
// GIVEN a VerrazzanoHelidonWorkload resource that already exists and the current upgrade version differs from the existing upgrade version
// WHEN the controller Reconcile function is called
// THEN the Fluentd image should be retrieved from the env and the new update version should be set on the workload status
func TestReconcileAlreadyExistsUpgrade(t *testing.T) {
	assert := asserts.New(t)
	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	testNamespace := "test-namespace"
	loggingSecretName := "test-secret-name"

	appConfigName := "test-appconf"
	componentName := "test-component"
	existingUpgradeVersion := "existing-upgrade-version"
	newUpgradeVersion := "new-upgrade-version"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName, constants.LabelUpgradeVersion: newUpgradeVersion}

	fluentdImage := "unit-test-image:latest"
	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := loggingscope.DefaultFluentdImage
	loggingscope.DefaultFluentdImage = fluentdImage
	defer func() { loggingscope.DefaultFluentdImage = initialDefaultFluentdImage }()

	params := map[string]string{
		"##APPCONF_NAME##":          appConfigName,
		"##APPCONF_NAMESPACE##":     testNamespace,
		"##COMPONENT_NAME##":        componentName,
		"##SCOPE_NAME##":            "test-scope",
		"##SCOPE_NAMESPACE##":       testNamespace,
		"##INGEST_URL##":            "http://test-ingest-host:9200",
		"##INGEST_SECRET_NAME##":    loggingSecretName,
		"##FLUENTD_IMAGE##":         "test-fluentd-image-name",
		"##WORKLOAD_APIVER##":       "oam.verrazzano.io/v1alpha1",
		"##WORKLOAD_KIND##":         "VerrazzanoHelidonWorkload",
		"##WORKLOAD_NAME##":         "test-workload-name",
		"##WORKLOAD_NAMESPACE##":    testNamespace,
		"##DEPLOYMENT_NAME##":       "test-deployment",
		"##CONTAINER_NAME##":        "test-container",
		"##CONTAINER_IMAGE##":       "test-container-image",
		"##CONTAINER_PORT_NAME##":   "http",
		"##CONTAINER_PORT_NUMBER##": "8080",
		"##LOGGING_SCOPE_NAME##":    "test-logging-scope",
		"##INGRESS_TRAIT_NAME##":    "test-ingress-trait",
		"##INGRESS_TRAIT_PATH##":    "/test-ingress-path",
	}
	// expect call to fetch existing deployment
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-deployment"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
			return nil
		})
	// expect a call to fetch the application configuration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-appconf"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appconf *oamapi.ApplicationConfiguration) error {
			assert.NoError(updateObjectFromYAMLTemplate(appconf, "test/templates/helidon_appconf_with_ingress_and_logging.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			assert.NoError(updateObjectFromYAMLTemplate(workload, "test/templates/helidon_workload.yaml", params))
			workload.ObjectMeta.Labels = labels
			workload.Status.CurrentUpgradeVersion = existingUpgradeVersion
			return nil
		}).Times(1)
	// expect a call to fetch the application configuration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-appconf"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appconf *oamapi.ApplicationConfiguration) error {
			assert.NoError(updateObjectFromYAMLTemplate(appconf, "test/templates/helidon_appconf_with_ingress_and_logging.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the logging scope
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-logging-scope"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, scope *vzapi.LoggingScope) error {
			assert.NoError(updateObjectFromYAMLTemplate(scope, "test/templates/logging_scope.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the Fluentd config and return a not found error
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "fluentd-config-helidon-test-deployment"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.GroupResource{Group: "", Resource: "configmap"}, "fluentd-config-helidon-test-deployment")).
		Times(1)

	// expect a call to get the Elasticsearch secret in app namespace - return not found
	testLoggingSecretFullName := types.NamespacedName{Namespace: testNamespace, Name: loggingSecretName}
	cli.EXPECT().
		Get(gomock.Any(), testLoggingSecretFullName, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.ParseGroupResource("v1.Secret"), loggingSecretName))

	// expect a call to create an empty Elasticsearch secret in app namespace (default behavior, so
	// that Fluentd volume mount works)
	cli.EXPECT().
		Create(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *v1.Secret, options *client.CreateOptions) error {
			asserts.Equal(t, testNamespace, sec.Namespace)
			asserts.Equal(t, loggingSecretName, sec.Name)
			asserts.Nil(t, sec.Data)
			asserts.Equal(t, client.CreateOptions{}, *options)
			return nil
		})
	// expect a call to create a Configmap
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, config *v1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(testNamespace, config.Namespace)
			assert.Equal("fluentd-config-helidon-test-deployment", config.Name)
			assert.Len(config.Data, 1)
			assert.Contains(config.Data["fluentd.conf"], "label")
			return nil
		}).Times(1)

	// expect a call to create the Deployment
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deploy *appsv1.Deployment, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(deploymentAPIVersion, deploy.APIVersion)
			assert.Equal(deploymentKind, deploy.Kind)
			// make sure the OAM component and app name labels were copied
			assert.Equal(labels, deploy.GetLabels())
			assert.Equal(params["##CONTAINER_NAME##"], deploy.Spec.Template.Spec.Containers[0].Name)
			assert.Len(deploy.Spec.Template.Spec.Containers, 2, "Expect 4 containers: app+sidecar")

			// The app container should be unmodified for the Helidon use case.
			c, found := findContainer(deploy.Spec.Template.Spec.Containers, "test-container")
			assert.True(found, "Expected to find app container test-container")
			assert.Equal(c.Image, "test-container-image")
			assert.Len(c.Ports, 1)
			assert.Equal(c.Ports[0].Name, "http")
			assert.Equal(c.Ports[0].ContainerPort, int32(8080))
			assert.Nil(c.VolumeMounts, "Expected app container to have no volume mounts")

			// The side car should have env vars setup correctly and 4 mount volumes.
			c, found = findContainer(deploy.Spec.Template.Spec.Containers, "fluentd")
			assert.True(found, "Expected to find sidecar container test-container")
			assert.Equal(fluentdImage, c.Image)
			assert.Len(c.Env, 10, "Expect sidecar container to have 10 env vars")
			assertEnvVar(t, c.Env, "WORKLOAD_NAME", "test-deployment")
			assertEnvVarFromField(t, c.Env, "APP_CONF_NAME", "metadata.labels['app.oam.dev/name']")
			assertEnvVarFromField(t, c.Env, "COMPONENT_NAME", "metadata.labels['app.oam.dev/component']")
			assertEnvVar(t, c.Env, "ELASTICSEARCH_URL", "http://test-ingest-host:9200")
			assertEnvVarFromSecret(t, c.Env, "ELASTICSEARCH_USER", "test-secret-name", "username")
			assertEnvVarFromSecret(t, c.Env, "ELASTICSEARCH_PASSWORD", "test-secret-name", "password")
			assert.Len(c.VolumeMounts, 4, "Expect sidecar container to have 4 volume mounts")
			assertVolumeMount(t, c.VolumeMounts, "fluentd-config-volume", "/fluentd/etc/fluentd.conf", "fluentd.conf", true)
			assertVolumeMount(t, c.VolumeMounts, "secret-volume", "/fluentd/secret", "", true)
			assertVolumeMount(t, c.VolumeMounts, "varlog", "/var/log", "", true)
			assertVolumeMount(t, c.VolumeMounts, "datadockercontainers", "/u01/data/docker/containers", "", true)
			return nil
		})
	// expect a call to create the Service
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, service *v1.Service, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(serviceAPIVersion, service.APIVersion)
			assert.Equal(serviceKind, service.Kind)
			return nil
		})
	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoHelidonWorkload) error {
			assert.Len(workload.Status.Resources, 2)
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, "test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileAlreadyExistsNoUpgrade tests reconciling a VerrazzanoHelidonWorkload when the Deployment CR already
// exists and the upgrade version specified in the labels matches the current upgrade version.
// This should result in the existing Fluentd image being reused.
// GIVEN a VerrazzanoHelidonWorkload resource that already exists and the current upgrade version matches the existing upgrade version
// WHEN the controller Reconcile function is called
// THEN the existing Fluentd image should be reused
func TestReconcileAlreadyExistsNoUpgrade(t *testing.T) {
	assert := asserts.New(t)
	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	testNamespace := "test-namespace"
	loggingSecretName := "test-secret-name"

	appConfigName := "test-appconf"
	componentName := "test-component"
	existingUpgradeVersion := "existing-upgrade-version"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName, constants.LabelUpgradeVersion: existingUpgradeVersion}

	existingFluentdImage := "unit-test-image:latest"
	fluentdImage := "unit-test-image:latest"
	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := loggingscope.DefaultFluentdImage
	loggingscope.DefaultFluentdImage = fluentdImage
	defer func() { loggingscope.DefaultFluentdImage = initialDefaultFluentdImage }()

	containers := []corev1.Container{{Name: loggingscope.FluentdContainerName, Image: existingFluentdImage}}

	params := map[string]string{
		"##APPCONF_NAME##":          appConfigName,
		"##APPCONF_NAMESPACE##":     testNamespace,
		"##COMPONENT_NAME##":        componentName,
		"##SCOPE_NAME##":            "test-scope",
		"##SCOPE_NAMESPACE##":       testNamespace,
		"##INGEST_URL##":            "http://test-ingest-host:9200",
		"##INGEST_SECRET_NAME##":    loggingSecretName,
		"##FLUENTD_IMAGE##":         "test-fluentd-image-name",
		"##WORKLOAD_APIVER##":       "oam.verrazzano.io/v1alpha1",
		"##WORKLOAD_KIND##":         "VerrazzanoHelidonWorkload",
		"##WORKLOAD_NAME##":         "test-workload-name",
		"##WORKLOAD_NAMESPACE##":    testNamespace,
		"##DEPLOYMENT_NAME##":       "test-deployment",
		"##CONTAINER_NAME##":        "test-container",
		"##CONTAINER_IMAGE##":       "test-container-image",
		"##CONTAINER_PORT_NAME##":   "http",
		"##CONTAINER_PORT_NUMBER##": "8080",
		"##LOGGING_SCOPE_NAME##":    "test-logging-scope",
		"##INGRESS_TRAIT_NAME##":    "test-ingress-trait",
		"##INGRESS_TRAIT_PATH##":    "/test-ingress-path",
	}
	// expect call to fetch existing deployment
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-deployment"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
			deployment.Spec.Template.Spec.Containers = containers
			return nil
		})
	// expect a call to fetch the application configuration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-appconf"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appconf *oamapi.ApplicationConfiguration) error {
			assert.NoError(updateObjectFromYAMLTemplate(appconf, "test/templates/helidon_appconf_with_ingress_and_logging.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			assert.NoError(updateObjectFromYAMLTemplate(workload, "test/templates/helidon_workload.yaml", params))
			workload.ObjectMeta.Labels = labels
			workload.Status.CurrentUpgradeVersion = existingUpgradeVersion
			return nil
		}).Times(1)
	// expect a call to fetch the application configuration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-appconf"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appconf *oamapi.ApplicationConfiguration) error {
			assert.NoError(updateObjectFromYAMLTemplate(appconf, "test/templates/helidon_appconf_with_ingress_and_logging.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the logging scope
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-logging-scope"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, scope *vzapi.LoggingScope) error {
			assert.NoError(updateObjectFromYAMLTemplate(scope, "test/templates/logging_scope.yaml", params))
			return nil
		}).Times(1)
	// expect a call to fetch the Fluentd config and return a not found error
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "fluentd-config-helidon-test-deployment"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.GroupResource{Group: "", Resource: "configmap"}, "fluentd-config-helidon-test-deployment")).
		Times(1)

	// expect a call to get the Elasticsearch secret in app namespace - return not found
	testLoggingSecretFullName := types.NamespacedName{Namespace: testNamespace, Name: loggingSecretName}
	cli.EXPECT().
		Get(gomock.Any(), testLoggingSecretFullName, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.ParseGroupResource("v1.Secret"), loggingSecretName))

	// expect a call to create an empty Elasticsearch secret in app namespace (default behavior, so
	// that Fluentd volume mount works)
	cli.EXPECT().
		Create(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *v1.Secret, options *client.CreateOptions) error {
			asserts.Equal(t, testNamespace, sec.Namespace)
			asserts.Equal(t, loggingSecretName, sec.Name)
			asserts.Nil(t, sec.Data)
			asserts.Equal(t, client.CreateOptions{}, *options)
			return nil
		})

	// expect a call to create a Configmap
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, config *v1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(testNamespace, config.Namespace)
			assert.Equal("fluentd-config-helidon-test-deployment", config.Name)
			assert.Len(config.Data, 1)
			assert.Contains(config.Data["fluentd.conf"], "label")
			return nil
		}).Times(1)

	// expect a call to create the Deployment
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deploy *appsv1.Deployment, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(deploymentAPIVersion, deploy.APIVersion)
			assert.Equal(deploymentKind, deploy.Kind)
			// make sure the OAM component and app name labels were copied
			assert.Equal(labels, deploy.GetLabels())
			assert.Equal(params["##CONTAINER_NAME##"], deploy.Spec.Template.Spec.Containers[0].Name)
			assert.Len(deploy.Spec.Template.Spec.Containers, 2, "Expect 4 containers: app+sidecar")

			// The app container should be unmodified for the Helidon use case.
			c, found := findContainer(deploy.Spec.Template.Spec.Containers, "test-container")
			assert.True(found, "Expected to find app container test-container")
			assert.Equal(c.Image, "test-container-image")
			assert.Len(c.Ports, 1)
			assert.Equal(c.Ports[0].Name, "http")
			assert.Equal(c.Ports[0].ContainerPort, int32(8080))
			assert.Nil(c.VolumeMounts, "Expected app container to have no volume mounts")

			// The side car should have env vars setup correctly and 4 mount volumes.
			c, found = findContainer(deploy.Spec.Template.Spec.Containers, "fluentd")
			assert.True(found, "Expected to find sidecar container test-container")
			assert.Equal(existingFluentdImage, c.Image)
			assert.Len(c.Env, 10, "Expect sidecar container to have 10 env vars")
			assertEnvVar(t, c.Env, "WORKLOAD_NAME", "test-deployment")
			assertEnvVarFromField(t, c.Env, "APP_CONF_NAME", "metadata.labels['app.oam.dev/name']")
			assertEnvVarFromField(t, c.Env, "COMPONENT_NAME", "metadata.labels['app.oam.dev/component']")
			assertEnvVar(t, c.Env, "ELASTICSEARCH_URL", "http://test-ingest-host:9200")
			assertEnvVarFromSecret(t, c.Env, "ELASTICSEARCH_USER", "test-secret-name", "username")
			assertEnvVarFromSecret(t, c.Env, "ELASTICSEARCH_PASSWORD", "test-secret-name", "password")
			assert.Len(c.VolumeMounts, 4, "Expect sidecar container to have 4 volume mounts")
			assertVolumeMount(t, c.VolumeMounts, "fluentd-config-volume", "/fluentd/etc/fluentd.conf", "fluentd.conf", true)
			assertVolumeMount(t, c.VolumeMounts, "secret-volume", "/fluentd/secret", "", true)
			assertVolumeMount(t, c.VolumeMounts, "varlog", "/var/log", "", true)
			assertVolumeMount(t, c.VolumeMounts, "datadockercontainers", "/u01/data/docker/containers", "", true)
			return nil
		})
	// expect a call to create the Service
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, service *v1.Service, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(serviceAPIVersion, service.APIVersion)
			assert.Equal(serviceKind, service.Kind)
			return nil
		})
	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoHelidonWorkload) error {
			assert.Len(workload.Status.Resources, 2)
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, "test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	vzapi.AddToScheme(scheme)
	return scheme
}

// newReconciler creates a new reconciler for testing
// c - The K8s client to inject into the reconciler
func newReconciler(c client.Client) Reconciler {
	scheme := newScheme()
	metricsReconciler := &metricstrait.Reconciler{Client: c, Scheme: scheme, Scraper: "verrazzano-system/vmi-system-prometheus-0"}
	return Reconciler{
		Client:  c,
		Log:     ctrl.Log.WithName("test"),
		Scheme:  scheme,
		Metrics: metricsReconciler,
	}
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// readTemplate reads a string template from a file and replaces values in the template from param maps
// template - The filename of a template
// params - a vararg of param maps
func readTemplate(template string, params ...map[string]string) (string, error) {
	bytes, err := ioutil.ReadFile("../../" + template)
	if err != nil {
		bytes, err = ioutil.ReadFile("../" + template)
		if err != nil {
			bytes, err = ioutil.ReadFile(template)
			if err != nil {
				return "", err
			}
		}
	}
	content := string(bytes)
	for _, p := range params {
		for k, v := range p {
			content = strings.ReplaceAll(content, k, v)
		}
	}
	return content, nil
}

// removeHeaderLines removes the top N lines from the text.
func removeHeaderLines(text string, lines int) string {
	line := 0
	output := ""
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		if line >= lines {
			output += scanner.Text()
			output += "\n"
		}
		line++
	}
	return output
}

// updateUnstructuredFromYAMLTemplate updates an unstructured from a populated YAML template file.
// uns - The unstructured to update
// template - The template file
// params - The param maps to merge into the template
func updateUnstructuredFromYAMLTemplate(uns *unstructured.Unstructured, template string, params ...map[string]string) error {
	str, err := readTemplate(template, params...)
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

// updateObjectFromYAMLTemplate updates an object from a populated YAML template file.
// uns - The unstructured to update
// template - The template file
// params - The param maps to merge into the template
func updateObjectFromYAMLTemplate(obj interface{}, template string, params ...map[string]string) error {
	uns := unstructured.Unstructured{}
	err := updateUnstructuredFromYAMLTemplate(&uns, template, params...)
	if err != nil {
		return err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(uns.Object, obj)
	if err != nil {
		return err
	}
	return nil
}

// findContainer finds a container in a slice by name.
func findContainer(containers []v1.Container, name string) (*v1.Container, bool) {
	for i, c := range containers {
		if c.Name == name {
			return &containers[i], true
		}
	}
	return nil, false
}

// assertEnvVar asserts the existence and correct value of an env var in a slice.
func assertEnvVar(t *testing.T, vars []v1.EnvVar, name string, value string) {
	for _, v := range vars {
		if v.Name == name {
			asserts.Equal(t, value, v.Value, "Expect var %s to have required value", name)
			return
		}
	}
	asserts.Fail(t, "Expect var to exist", "name: %s", name)
}

// assertVolumeMount asserts the existance and correct content of a volume mount in a slice.
func assertVolumeMount(t *testing.T, mounts []v1.VolumeMount, name string, path string, subPath string, readOnly bool) {
	for _, m := range mounts {
		if m.Name == name {
			asserts.Equal(t, path, m.MountPath, "Expect mountPath to be correct")
			asserts.Equal(t, subPath, m.SubPath, "Expect subPath to be correct")
			asserts.Equal(t, readOnly, m.ReadOnly, "Expect readOnly to be correct")
			return
		}
	}
	asserts.Fail(t, "Expect volume mount to exist", "name: %s", name)
}

func assertEnvVarFromField(t *testing.T, vars []v1.EnvVar, name string, fieldPath string) {
	for _, v := range vars {
		if v.Name == name {
			asserts.Equal(t, fieldPath, v.ValueFrom.FieldRef.FieldPath, "Expect var field path")
			return
		}
	}
	asserts.Fail(t, "Expect var to exist", "name: %s", name)
}

func assertEnvVarFromSecret(t *testing.T, vars []v1.EnvVar, name string, secretName string, secretKey string) {
	for _, v := range vars {
		if v.Name == name {
			asserts.Equal(t, secretName, v.ValueFrom.SecretKeyRef.LocalObjectReference.Name, "Expect var secret ref to have correct name")
			asserts.Equal(t, secretKey, v.ValueFrom.SecretKeyRef.Key, "Expect var secret ref to have correct key")
			return
		}
	}
	asserts.Fail(t, "Expect var to exist", "name: %s", name)
}
