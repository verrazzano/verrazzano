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
	"github.com/verrazzano/verrazzano/application-operator/mocks"
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
			Name: "hello-helidon-deployment-new",
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
	// expect a call to fetch the VerrazzanoHelidonWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-helidon-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoHelidonWorkload) error {
			workload.Spec.DeploymentTemplate = *deploymentTemplate
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoHelidonWorkload"
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

	params := map[string]string{
		"##APPCONF_NAME##":          "test-appconf",
		"##APPCONF_NAMESPACE##":     testNamespace,
		"##COMPONENT_NAME##":        "test-component",
		"##SCOPE_NAME##":            "test-scope",
		"##SCOPE_NAMESPACE##":       testNamespace,
		"##INJEST_HOST##":           "test-injest-host",
		"##INJEST_PORT##":           "9200",
		"##INJEST_SECRET_NAME##":    "test-secret-name",
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
	// expect a call to fetch the fluentd config and return a not found error
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "fluentd-config-helidon-test-deployment"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.GroupResource{Group: "", Resource: "configmap"}, "fluentd-config-helidon-test-deployment")).
		Times(1)
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
	// expect a call to fetch the Elasticsearch endpoint secret and return a not found error.
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-secret-name"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.GroupResource{Group: "", Resource: "secret"}, "test-secret-name")).
		Times(1)
	// expect a call to fetch the Elasticsearch endpoint secret and return a not found error.
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "verrazzano-system", Name: "verrazzano"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *v1.Secret) error {
			obj.Data = map[string][]byte{"test-data-key": []byte("test-data-value")}
			return nil
		}).
		Times(1)
	// expect a call to create a Secret
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, obj *v1.Secret, opts ...client.CreateOption) error {
			assert.Equal(testNamespace, obj.Namespace)
			assert.Equal("test-secret-name", obj.Name)
			assert.Len(obj.Data, 1)
			assert.Equal(obj.Data["test-data-key"], []byte("test-data-value"))
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
	return Reconciler{
		Client: c,
		Log:    ctrl.Log.WithName("test"),
		Scheme: newScheme(),
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
