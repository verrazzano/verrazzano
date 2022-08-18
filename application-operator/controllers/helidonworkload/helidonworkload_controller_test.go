// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonworkload

import (
	"context"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/testutil"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"

	oamapi "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/logging"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

const (
	namespace          = "unit-test-namespace"
	testRestartVersion = "new-restart"
)

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
	_ = vzapi.AddToScheme(scheme)
	reconciler = Reconciler{Client: cli, Scheme: scheme}
	mgr.EXPECT().GetControllerOptions().AnyTimes()
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(logr.Discard())
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
	metricsexporter.RequiredInitialization()
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
	metricsexporter.RequiredInitialization()
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			return k8serrors.NewBadRequest("An error has occurred")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.Nil(err)
	assert.True(result.Requeue)
}

// TestReconcileWorkloadMissingData tests reconciling a VerrazzanoHelidonWorkload when the workload
// can be fetched but doesn't contain all required data.
// GIVEN a VerrazzanoHelidonWorkload resource has been created
// WHEN the controller Reconcile function is called and we attempt to validate the workload and get an error
// THEN return the error
func TestReconcileWorkloadMissingData(t *testing.T) {
	metricsexporter.RequiredInitialization()
	assert := asserts.New(t)
	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}
	helidonTestContainerPort := corev1.ContainerPort{
		ContainerPort: 8080,
		Name:          "http",
	}
	helidonTestContainer := corev1.Container{
		Name:  "hello-helidon-container-new",
		Image: "ghcr.io/verrazzano/example-helidon-greet-app-v1:1.0.0-1-20211215184123-0a1b633",
		Ports: []corev1.ContainerPort{
			helidonTestContainerPort,
		},
	}
	deploymentTemplate := &vzapi.DeploymentTemplate{
		Metadata: metav1.ObjectMeta{
			Namespace: namespace,
			Labels: map[string]string{
				"app": "hello-helidon-deploy-new",
			},
		},
		PodSpec: corev1.PodSpec{
			Containers: []corev1.Container{
				helidonTestContainer,
			},
		},
	}

	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			workload.Spec.DeploymentTemplate = *deploymentTemplate
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoHelidonWorkload"
			workload.Namespace = namespace
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.Nil(err)
	assert.True(result.Requeue)
}

// TestReconcileCreateHelidon tests the basic happy path of reconciling a VerrazzanoHelidonWorkload. We
// expect to write out a Deployment and Service but we aren't adding logging or any other scopes or traits.
// GIVEN a VerrazzanoHelidonWorkload resource is created
// WHEN the controller Reconcile function is called
// THEN expect a Deployment and Service to be written
func TestReconcileCreateHelidon(t *testing.T) {
	metricsexporter.RequiredInitialization()
	assert := asserts.New(t)
	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}
	helidonTestContainerPort := corev1.ContainerPort{
		ContainerPort: 8080,
		Name:          "http",
	}
	helidonTestContainer := corev1.Container{
		Name:  "hello-helidon-container-new",
		Image: "ghcr.io/verrazzano/example-helidon-greet-app-v1:1.0.0-1-20211215184123-0a1b633",
		Ports: []corev1.ContainerPort{
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
		PodSpec: corev1.PodSpec{
			Containers: []corev1.Container{
				helidonTestContainer,
			},
		},
		Selector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "hello-helidon",
			},
			MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      "app",
				Operator: "In",
				Values:   []string{"hello-helidon"},
			}},
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
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoHelidonWorkload"
			workload.Namespace = namespace
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
			assert.Equal([]corev1.Container{
				helidonTestContainer,
			}, deploy.Spec.Template.Spec.Containers)
			return nil
		})
	// expect a call to create the Service
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, service *corev1.Service, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(serviceAPIVersion, service.APIVersion)
			assert.Equal(serviceKind, service.Kind)
			return nil
		})
	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoHelidonWorkload, opts ...client.UpdateOption) error {
			assert.Len(workload.Status.Resources, 2)
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateHelidonWithMultipleContainers tests the basic happy path of reconciling a VerrazzanoHelidonWorkload with multiple containers.
// We expect to write out a Deployment and Service but we aren't adding logging or any other scopes or traits.
// GIVEN a VerrazzanoHelidonWorkload resource is created
// AND that the workload has multiple containers
// WHEN the controller Reconcile function is called
// THEN expect a Deployment and Service to be written with multiple containers
func TestReconcileCreateHelidonWithMultipleContainers(t *testing.T) {
	metricsexporter.RequiredInitialization()
	assert := asserts.New(t)
	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}
	helidonTestContainerPort := corev1.ContainerPort{
		ContainerPort: 8080,
		Name:          "http",
	}
	helidonTestContainerPort2 := corev1.ContainerPort{
		ContainerPort: 8081,
		Name:          "udp",
		Protocol:      corev1.ProtocolUDP,
	}
	helidonTestContainer := corev1.Container{
		Name:  "hello-helidon-container-new",
		Image: "ghcr.io/verrazzano/example-helidon-greet-app-v1:1.0.0-1-20211215184123-0a1b633",
		Ports: []corev1.ContainerPort{
			helidonTestContainerPort,
		},
	}
	helidonTestContainer2 := corev1.Container{
		Name:  "hello-helidon-container-new2",
		Image: "ghcr.io/verrazzano/example-helidon-greet-app-v1:1.0.0-1-20211215184123-0a1b633",
		Ports: []corev1.ContainerPort{
			helidonTestContainerPort2,
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
		PodSpec: corev1.PodSpec{
			Containers: []corev1.Container{
				helidonTestContainer,
				helidonTestContainer2,
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
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoHelidonWorkload"
			workload.Namespace = namespace
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
			assert.Equal([]corev1.Container{
				helidonTestContainer,
				helidonTestContainer2,
			}, deploy.Spec.Template.Spec.Containers)

			return nil
		})
	// expect a call to create the Service
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, service *corev1.Service, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(serviceAPIVersion, service.APIVersion)
			assert.Equal(serviceKind, service.Kind)
			assert.Equal(service.Spec.Ports[0].Name, "tcp-"+helidonTestContainer.Name+"-"+strconv.FormatInt(int64(helidonTestContainer.Ports[0].ContainerPort), 10))
			assert.Equal(service.Spec.Ports[0].Port, helidonTestContainer.Ports[0].ContainerPort)
			assert.Equal(service.Spec.Ports[0].TargetPort, intstr.FromInt(int(helidonTestContainer.Ports[0].ContainerPort)))
			assert.Equal(service.Spec.Ports[0].Protocol, corev1.ProtocolTCP)
			assert.Equal(service.Spec.Ports[1].Name, "tcp-"+helidonTestContainer2.Name+"-"+strconv.FormatInt(int64(helidonTestContainer2.Ports[0].ContainerPort), 10))
			assert.Equal(service.Spec.Ports[1].Port, helidonTestContainer2.Ports[0].ContainerPort)
			assert.Equal(service.Spec.Ports[1].TargetPort, intstr.FromInt(int(helidonTestContainer2.Ports[0].ContainerPort)))
			assert.Equal(service.Spec.Ports[1].Protocol, helidonTestContainer2.Ports[0].Protocol)
			return nil
		})
	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoHelidonWorkload, opts ...client.UpdateOption) error {
			assert.Len(workload.Status.Resources, 2)
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
	metricsexporter.RequiredInitialization()
	assert := asserts.New(t)
	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	testNamespace := "test-namespace"
	loggingSecretName := "test-secret-name"

	fluentdImage := "unit-test-image:latest"
	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

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

	// expect a call to create the Deployment
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deploy *appsv1.Deployment, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(deploymentAPIVersion, deploy.APIVersion)
			assert.Equal(deploymentKind, deploy.Kind)
			// make sure the OAM component and app name labels were copied
			assert.Equal(map[string]string{"app.oam.dev/component": "test-component", "app.oam.dev/name": "test-appconf"}, deploy.GetLabels())
			assert.Equal(params["##CONTAINER_NAME##"], deploy.Spec.Template.Spec.Containers[0].Name)
			assert.Len(deploy.Spec.Template.Spec.Containers, 1, "Expect 4 containers: app+sidecar")

			// The app container should be unmodified for the Helidon use case.
			c, found := findContainer(deploy.Spec.Template.Spec.Containers, "test-container")
			assert.True(found, "Expected to find app container test-container")
			assert.Equal(c.Image, "test-container-image")
			assert.Len(c.Ports, 1)
			assert.Equal(c.Ports[0].Name, "http")
			assert.Equal(c.Ports[0].ContainerPort, int32(8080))
			assert.Nil(c.VolumeMounts, "Expected app container to have no volume mounts")
			return nil
		})
	// expect a call to create the Service
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, service *corev1.Service, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(serviceAPIVersion, service.APIVersion)
			assert.Equal(serviceKind, service.Kind)
			return nil
		})
	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoHelidonWorkload, opts ...client.UpdateOption) error {
			assert.Len(workload.Status.Resources, 2)
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, "test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
	metricsexporter.RequiredInitialization()
	assert := asserts.New(t)
	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	testNamespace := "test-namespace"
	loggingSecretName := "test-secret-name"

	fluentdImage := "unit-test-image:latest"
	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

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
		})
	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			assert.NoError(updateObjectFromYAMLTemplate(workload, "test/templates/helidon_workload_multi_container.yaml", params))
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
			assert.Len(deploy.Spec.Template.Spec.Containers, 2, "Expect 2 containers.")

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
			return nil
		})
	// expect a call to create the Service
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, service *corev1.Service, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(serviceAPIVersion, service.APIVersion)
			assert.Equal(serviceKind, service.Kind)
			return nil
		})
	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoHelidonWorkload, opts ...client.UpdateOption) error {
			assert.Len(workload.Status.Resources, 2)
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, "test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	return scheme
}

// newReconciler creates a new reconciler for testing
// c - The K8s client to inject into the reconciler
func newReconciler(c client.Client) Reconciler {
	scheme := newScheme()
	metricsReconciler := &metricstrait.Reconciler{Client: c, Scheme: scheme, Scraper: "verrazzano-system/vmi-system-prometheus-0"}
	return Reconciler{
		Client:  c,
		Log:     zap.S().With("test"),
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
func findContainer(containers []corev1.Container, name string) (*corev1.Container, bool) {
	for i, c := range containers {
		if c.Name == name {
			return &containers[i], true
		}
	}
	return nil, false
}

func getTestDeployment(restartVersion string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{}
	annotateRestartVersion(deployment, restartVersion)
	return deployment
}

func annotateRestartVersion(deployment *appsv1.Deployment, restartVersion string) {
	deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	deployment.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = restartVersion
}

// TestReconcileRestart tests reconciling a VerrazzanoHelidonWorkload when the restart-version specified in the annotations.
// This should result in restart-version written to the Helidon Deployment.
// GIVEN a VerrazzanoHelidonWorkload resource
// WHEN the controller Reconcile function is called and the restart-version is specified
// THEN the restart-version written
func TestReconcileRestart(t *testing.T) {
	metricsexporter.RequiredInitialization()
	assert := asserts.New(t)
	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	testNamespace := "test-namespace"
	loggingSecretName := "test-secret-name"

	appConfigName := "test-appconf"
	componentName := "test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}
	annotations := map[string]string{vzconst.RestartVersionAnnotation: testRestartVersion}

	fluentdImage := "unit-test-image:latest"
	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

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
			workload.ObjectMeta.Annotations = annotations
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
			assert.Len(deploy.Spec.Template.Spec.Containers, 1)

			// The app container should be unmodified for the Helidon use case.
			c, found := findContainer(deploy.Spec.Template.Spec.Containers, "test-container")
			assert.True(found, "Expected to find app container test-container")
			assert.Equal(c.Image, "test-container-image")
			assert.Len(c.Ports, 1)
			assert.Equal(c.Ports[0].Name, "http")
			assert.Equal(c.Ports[0].ContainerPort, int32(8080))
			assert.Nil(c.VolumeMounts, "Expected app container to have no volume mounts")
			return nil
		})
	// expect a call to create the Service
	cli.EXPECT().
		Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, service *corev1.Service, patch client.Patch, applyOpts ...client.PatchOption) error {
			assert.Equal(serviceAPIVersion, service.APIVersion)
			assert.Equal(serviceKind, service.Kind)
			return nil
		})
	// expect a call to list the deployment
	cli.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *appsv1.DeploymentList, opts ...client.ListOption) error {
			list.Items = []appsv1.Deployment{*getTestDeployment("")}
			return nil
		})
	// expect a call to fetch the deployment
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
			annotateRestartVersion(deployment, "")
			return nil
		})
	// expect a call to update the deployment
	cli.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deploy *appsv1.Deployment, opts ...client.UpdateOption) error {
			assert.Equal(testRestartVersion, deploy.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation])
			return nil
		})
	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoHelidonWorkload, opts ...client.UpdateOption) error {
			assert.Len(workload.Status.Resources, 2)
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, "test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileKubeSystem tests to make sure we do not reconcile
// Any resource that belong to the kube-system namespace
func TestReconcileKubeSystem(t *testing.T) {
	metricsexporter.RequiredInitialization()
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// create a request and reconcile it
	request := newRequest(vzconst.KubeSystem, "unit-test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.Nil(err)
	assert.True(result.IsZero())
}

// TestReconcileFailed tests to make sure the failure metric is being exposed
func TestReconcileFailed(t *testing.T) {
	testAppConfigName := "unit-test-app-config"
	testNamespace := "test-ns"
	metricsexporter.RequiredInitialization()
	assert := asserts.New(t)
	clientBuilder := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	// Create a request and reconcile it
	reconciler := newReconciler(clientBuilder)
	request := newRequest(testNamespace, testAppConfigName)
	reconcileerrorCounterObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.HelidonReconcileError)
	assert.NoError(err)
	// Expect a call to fetch the error
	reconcileFailedCounterBefore := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	reconcileerrorCounterObject.Get().Inc()
	reconciler.Reconcile(context.TODO(), request)
	reconcileFailedCounterAfter := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	assert.Equal(reconcileFailedCounterBefore, reconcileFailedCounterAfter-1)
}
