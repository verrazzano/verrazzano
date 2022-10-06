// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package containerizedworkload

import (
	"context"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"os"
	"strings"
	"testing"

	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	testRestartVersion = "new-restart"
	testNamespace      = "test-namespace"

	appConfigName = "test-appconf"
	componentName = "test-component"
)

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = oamcore.AddToScheme(scheme)
	return scheme
}

// newReconciler creates a new reconciler for testing
func newReconciler(c client.Client) Reconciler {
	return Reconciler{
		Client: c,
		Log:    zap.S().With("test"),
		Scheme: newScheme(),
	}
}

// newRequest creates a new reconciler request for testing
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// TestReconcileRestart tests reconciling a ContainerizedWorkload when the restart-version specified in the annotations.
// This should result in restart-version written to the Deployment.
// GIVEN a ContainerizedWorkload resource
// WHEN the controller Reconcile function is called and the restart-version is specified
// THEN the restart-version written
func TestReconcileRestart(t *testing.T) {
	assert := asserts.New(t)
	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}
	annotations := map[string]string{vzconst.RestartVersionAnnotation: testRestartVersion}

	// expect a call to fetch the ContainerizedWorkload
	params := map[string]string{
		"##OAM_APP_NAME##":         "test-oam-app-name",
		"##OAM_COMP_NAME##":        "test-oam-comp-name",
		"##TRAIT_NAME##":           "test-trait-name",
		"##TRAIT_NAMESPACE##":      "test-namespace",
		"##WORKLOAD_APIVER##":      "core.oam.dev/v1alpha2",
		"##WORKLOAD_KIND##":        "ContainerizedWorkload",
		"##WORKLOAD_NAME##":        "test-workload-name",
		"##PROMETHEUS_NAME##":      "vmi-system-prometheus-0",
		"##PROMETHEUS_NAMESPACE##": "verrazzano-system",
		"##DEPLOYMENT_NAMESPACE##": "test-namespace",
		"##DEPLOYMENT_NAME##":      "test-workload-name",
	}
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: "test-verrazzano-containerized-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *oamv1.ContainerizedWorkload) error {
			assert.NoError(updateObjectFromYAMLTemplate(workload, "test/templates/containerized_workload_deployment.yaml", params))
			workload.ObjectMeta.Labels = labels
			workload.ObjectMeta.Annotations = annotations
			return nil
		}).Times(1)
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

	// create a request and reconcile it
	request := newRequest(testNamespace, "test-verrazzano-containerized-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileKubeSystem tests to make sure we do not reconcile
// Any resource that belong to the kube-system namespace
func TestReconcileKubeSystem(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	// create a request and reconcile it
	request := newRequest(vzconst.KubeSystem, "test-verrazzano-containerized-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.True(result.IsZero())
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

// readTemplate reads a string template from a file and replaces values in the template from param maps
// template - The filename of a template
// params - a vararg of param maps
func readTemplate(template string, params ...map[string]string) (string, error) {
	bytes, err := os.ReadFile("../../" + template)
	if err != nil {
		bytes, err = os.ReadFile("../" + template)
		if err != nil {
			bytes, err = os.ReadFile(template)
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

func getTestDeployment(restartVersion string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{}
	annotateRestartVersion(deployment, restartVersion)
	return deployment
}

func annotateRestartVersion(deployment *appsv1.Deployment, restartVersion string) {
	deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	deployment.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = restartVersion
}
