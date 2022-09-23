// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"bytes"
	"context"
	"github.com/go-logr/logr"
	"io/ioutil"
	"os"
	"testing"
	"text/template"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	"github.com/golang/mock/gomock"
	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	asserts "github.com/stretchr/testify/assert"
	imagesv1alpha1 "github.com/verrazzano/verrazzano/image-patch-operator/api/images/v1alpha1"
	"github.com/verrazzano/verrazzano/image-patch-operator/mocks"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	k8sapps "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	k8score "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

// TestReconcilerSetupWithManager tests the creation and setup of a new reconciler
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var mgr *mocks.MockManager
	var cli *mocks.MockClient
	var scheme *runtime.Scheme
	var reconciler ImageBuildRequestReconciler
	var err error

	mocker = gomock.NewController(t)
	mgr = mocks.NewMockManager(mocker)
	cli = mocks.NewMockClient(mocker)
	scheme = runtime.NewScheme()
	_ = imagesv1alpha1.AddToScheme(scheme)
	reconciler = ImageBuildRequestReconciler{Client: cli, Scheme: scheme}
	mgr.EXPECT().GetControllerOptions().AnyTimes()
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(logr.Discard())
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}

// TestNewImageBuildRequest tests the Reconcile method for the following:
// GIVEN an ImageBuildRequest is applied
// WHEN the ImageJob is created
// THEN verify that the status of the ImageBuildRequest reflects that the build is in progress
// AND verify that the IBR environmental variables are passed to the ImageJob correctly
func TestNewImageBuildRequest(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	params := map[string]string{
		"BASE_IMAGE":            "ghcr.io/oracle/oraclelinux:8-slim",
		"JDK_INSTALLER":         "jdk-8u281-linux-x64.tar.gz",
		"WEBLOGIC_INSTALLER":    "fmw_12.2.1.4.0_wls.jar",
		"IMAGE_NAME":            "test-build",
		"IMAGE_TAG":             "test-tag",
		"IMAGE_REGISTRY":        "phx.ocir.io",
		"IMAGE_REPOSITORY":      "myrepo/verrazzano",
		"JDK_INSTALLER_VERSION": "8u281",
		"WLS_INSTALLER_VERSION": "12.2.1.4.0",
		"IBR_NAME":              "cluster1",
	}

	// Creating an ImageBuildRequest resource
	assert.NoError(createResourceFromTemplate(cli, "test/templates/imagebuildrequest_instance.yaml", params))

	// Set test values for resource limits and requests
	cpuValueString := "1100m"
	memoryValueString := "1Gi"
	_ = os.Setenv("WIT_POD_RESOURCE_LIMIT_CPU", cpuValueString)
	_ = os.Setenv("WIT_POD_RESOURCE_LIMIT_MEMORY", memoryValueString)
	_ = os.Setenv("WIT_POD_RESOURCE_REQUEST_CPU", cpuValueString)
	_ = os.Setenv("WIT_POD_RESOURCE_REQUEST_MEMORY", memoryValueString)

	request := newRequest("default", "cluster1")
	reconciler := newImageBuildRequestReconciler(cli)
	_, err := reconciler.Reconcile(context.TODO(), request)
	assert.NoError(err)

	// Verify the DryRun flag is set to false by default (this value can be changed in the helm config values.yaml file)
	assert.Equal(false, reconciler.DryRun)

	ibr := &imagesv1alpha1.ImageBuildRequest{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "cluster1"}, ibr)

	// Ensure that IBR resource exists
	assert.NoError(err)

	// The status, condition, and message of the IBR should reflect that the ImageJob is in progress
	assert.Equal(imagesv1alpha1.StateType("Building"), ibr.Status.State)
	assert.Equal(imagesv1alpha1.ConditionType("BuildStarted"), ibr.Status.Conditions[0].Type)
	assert.Equal("ImageBuildRequest build in progress", ibr.Status.Conditions[0].Message)

	jb := &batchv1.Job{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "verrazzano-images-cluster1"}, jb)

	// Ensure that a Kubernetes job is created when an IBR is created
	assert.NoError(err)

	// Verify istio-injection disabled for the job
	assert.Equal("false", jb.Labels["sidecar.istio.io/inject"])

	// Convert test values for resource limits and requests to type Quantity
	cpuValue, _ := resource.ParseQuantity(cpuValueString)
	memoryValue, _ := resource.ParseQuantity(memoryValueString)

	// Verify that the resource limits and requests are the expected test values
	assert.Equal(cpuValue, jb.Spec.Template.Spec.Containers[0].Resources.Limits["cpu"])
	assert.Equal(memoryValue, jb.Spec.Template.Spec.Containers[0].Resources.Limits["memory"])
	assert.Equal(cpuValue, jb.Spec.Template.Spec.Containers[0].Resources.Requests["cpu"])
	assert.Equal(memoryValue, jb.Spec.Template.Spec.Containers[0].Resources.Requests["memory"])

	// Testing that the spec fields of the IBR propagate to the environmental variables of the ImageJob
	assert.Equal("test-build", jb.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Equal("test-tag", jb.Spec.Template.Spec.Containers[0].Env[1].Value)
	assert.Equal("phx.ocir.io", jb.Spec.Template.Spec.Containers[0].Env[2].Value)
	assert.Equal("myrepo/verrazzano", jb.Spec.Template.Spec.Containers[0].Env[3].Value)
	assert.Equal("ghcr.io/oracle/oraclelinux:8-slim", jb.Spec.Template.Spec.Containers[0].Env[4].Value)
	assert.Equal("jdk-8u281-linux-x64.tar.gz", jb.Spec.Template.Spec.Containers[0].Env[5].Value)
	assert.Equal("8u281", jb.Spec.Template.Spec.Containers[0].Env[6].Value)
	assert.Equal("fmw_12.2.1.4.0_wls.jar", jb.Spec.Template.Spec.Containers[0].Env[7].Value)
	assert.Equal("12.2.1.4.0", jb.Spec.Template.Spec.Containers[0].Env[8].Value)
	assert.Equal("false", jb.Spec.Template.Spec.Containers[0].Env[11].Value)
	assert.Equal("true", jb.Spec.Template.Spec.Containers[0].Env[12].Value)

	// Verifying that the PV, PVC, and Volume Mount are present on the created job
	assert.Equal("installers-storage", jb.Spec.Template.Spec.Volumes[1].Name)
	assert.Equal("installers-storage-claim", jb.Spec.Template.Spec.Volumes[1].PersistentVolumeClaim.ClaimName)
	assert.Equal("installers-storage", jb.Spec.Template.Spec.Containers[0].VolumeMounts[1].Name)
	assert.Equal("/installers", jb.Spec.Template.Spec.Containers[0].VolumeMounts[1].MountPath)

}

// TestIBRJobSucceeded tests the Reconcile method for the following:
// GIVEN an ImageBuildRequest is applied
// WHEN the ImageJob is completed
// THEN verify that the status of the ImageBuildRequest reflects the build is completed
func TestIBRJobSucceeded(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	params := map[string]string{
		"IBR_NAME": "cluster1",
	}

	// Creating an ImageBuildRequest resource and Kubernetes job in succeeded state
	assert.NoError(createResourceFromTemplate(cli, "test/templates/imagebuildrequest_instance.yaml", params))
	assert.NoError(createResourceFromTemplate(cli, "test/templates/job_succeeded.yaml", params))

	request := newRequest("default", "cluster1")
	reconciler := newImageBuildRequestReconciler(cli)
	_, err := reconciler.Reconcile(context.TODO(), request)
	assert.NoError(err)

	// Testing if ImageBuildRequest status reflects that ImageJob is complete
	ibr := &imagesv1alpha1.ImageBuildRequest{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "cluster1"}, ibr)
	assert.NoError(err)
	assert.Equal(imagesv1alpha1.StateType("Published"), ibr.Status.State)
	assert.Equal(imagesv1alpha1.ConditionType("BuildCompleted"), ibr.Status.Conditions[0].Type)
	assert.Equal("ImageBuildRequest build completed successfully", ibr.Status.Conditions[0].Message)

	// Verify the job exists in a succeeded state
	jb := &batchv1.Job{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "verrazzano-images-cluster1"}, jb)
	assert.NoError(err)
	assert.Equal(int32(1), jb.Status.Succeeded)
}

// TestIBRJobFailed tests the Reconcile method for the following:
// GIVEN an ImageBuildRequest is applied
// WHEN the ImageJob fails
// THEN verify that the status of the ImageBuildRequest reflects the build failed
func TestIBRJobFailed(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	params := map[string]string{
		"IBR_NAME": "cluster1",
	}

	// Creating an ImageBuildRequest resource and Kubernetes job in failed state
	assert.NoError(createResourceFromTemplate(cli, "test/templates/imagebuildrequest_instance.yaml", params))
	assert.NoError(createResourceFromTemplate(cli, "test/templates/job_failed.yaml", params))

	request := newRequest("default", "cluster1")
	reconciler := newImageBuildRequestReconciler(cli)
	_, err := reconciler.Reconcile(context.TODO(), request)
	assert.NoError(err)

	// Testing if ImageBuildRequest status reflects that ImageJob failed
	ibr := &imagesv1alpha1.ImageBuildRequest{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "cluster1"}, ibr)
	assert.NoError(err)
	assert.Equal(imagesv1alpha1.StateType("Failed"), ibr.Status.State)
	assert.Equal(imagesv1alpha1.ConditionType("BuildFailed"), ibr.Status.Conditions[0].Type)
	assert.Equal("ImageBuildRequest build failed to complete", ibr.Status.Conditions[0].Message)

	// Verify the job exists in a failed state
	jb := &batchv1.Job{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "verrazzano-images-cluster1"}, jb)
	assert.NoError(err)
	assert.Equal(int32(1), jb.Status.Failed)
}

// TestIBRDryRun tests the Reconcile method for the following:
// GIVEN an ImageBuildRequest is applied
// WHEN DryRun is set to true
// THEN verify that the status of the ImageBuildRequest reflects that a DryRun is in progress
func TestIBRDryRun(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	params := map[string]string{
		"IBR_NAME": "cluster1",
	}

	// Creating an ImageBuildRequest resource
	assert.NoError(createResourceFromTemplate(cli, "test/templates/imagebuildrequest_instance.yaml", params))

	request := newRequest("default", "cluster1")
	reconciler := newImageBuildRequestReconciler(cli)

	// Running the image job as a DryRun
	reconciler.DryRun = true
	_, err := reconciler.Reconcile(context.TODO(), request)
	assert.NoError(err)

	// Testing if ImageBuildRequest status reflects that ImageJob is in progress of a DryRun
	ibr := &imagesv1alpha1.ImageBuildRequest{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "cluster1"}, ibr)
	assert.NoError(err)
	assert.Equal(imagesv1alpha1.StateType("DryRunActive"), ibr.Status.State)
	assert.Equal(imagesv1alpha1.ConditionType("DryRunStarted"), ibr.Status.Conditions[0].Type)
	assert.Equal("ImageBuildRequest DryRun in progress", ibr.Status.Conditions[0].Message)

	// Verifying a job gets created when DryRun is active
	jb := &batchv1.Job{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "verrazzano-images-cluster1"}, jb)
	assert.NoError(err)
}

// TestIBRDryRunJobSucceeded tests the Reconcile method for the following:
// GIVEN an ImageBuildRequest is applied
// WHEN the ImageJob completes a DryRun successfully
// THEN verify that the status of the ImageBuildRequest reflects the DryRun is completed
func TestIBRDryRunJobSucceeded(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	params := map[string]string{
		"IBR_NAME": "cluster1",
	}

	// Creating an ImageBuildRequest resource and Kubernetes job in succeeded state
	assert.NoError(createResourceFromTemplate(cli, "test/templates/imagebuildrequest_instance.yaml", params))
	assert.NoError(createResourceFromTemplate(cli, "test/templates/job_succeeded.yaml", params))

	request := newRequest("default", "cluster1")
	reconciler := newImageBuildRequestReconciler(cli)

	// Running the image job as a DryRun
	reconciler.DryRun = true
	_, err := reconciler.Reconcile(context.TODO(), request)
	assert.NoError(err)

	// Testing if ImageBuildRequest status reflects that ImageJob DryRun is complete
	ibr := &imagesv1alpha1.ImageBuildRequest{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "cluster1"}, ibr)
	assert.NoError(err)
	assert.Equal(imagesv1alpha1.StateType("DryRunPrinted"), ibr.Status.State)
	assert.Equal(imagesv1alpha1.ConditionType("DryRunCompleted"), ibr.Status.Conditions[0].Type)
	assert.Equal("ImageBuildRequest DryRun completed successfully", ibr.Status.Conditions[0].Message)

	// Verify the job exists in a succeeded state
	jb := &batchv1.Job{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "verrazzano-images-cluster1"}, jb)
	assert.NoError(err)
	assert.Equal(int32(1), jb.Status.Succeeded)
}

// TestIBRJobFailed tests the Reconcile method for the following:
// GIVEN an ImageBuildRequest is applied
// WHEN the ImageJob DryRun fails
// THEN verify that the status of the ImageBuildRequest reflects the DryRun failed
func TestIBRDryRunJobFailed(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	params := map[string]string{
		"IBR_NAME": "cluster1",
	}

	// Creating an ImageBuildRequest resource and Kubernetes job in failed state
	assert.NoError(createResourceFromTemplate(cli, "test/templates/imagebuildrequest_instance.yaml", params))
	assert.NoError(createResourceFromTemplate(cli, "test/templates/job_failed.yaml", params))

	request := newRequest("default", "cluster1")
	reconciler := newImageBuildRequestReconciler(cli)

	// Running the image job as a DryRun
	reconciler.DryRun = true
	_, err := reconciler.Reconcile(context.TODO(), request)
	assert.NoError(err)

	// Testing if ImageBuildRequest status reflects that ImageJob DryRun failed
	ibr := &imagesv1alpha1.ImageBuildRequest{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "cluster1"}, ibr)
	assert.NoError(err)
	assert.Equal(imagesv1alpha1.StateType("DryRunFailure"), ibr.Status.State)
	assert.Equal(imagesv1alpha1.ConditionType("DryRunFailed"), ibr.Status.Conditions[0].Type)
	assert.Equal("ImageBuildRequest DryRun failed to complete", ibr.Status.Conditions[0].Message)

	// Verify the job exists in a failed state
	jb := &batchv1.Job{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "verrazzano-images-cluster1"}, jb)
	assert.NoError(err)
	assert.Equal(int32(1), jb.Status.Failed)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = core.AddToScheme(scheme)
	_ = k8sapps.AddToScheme(scheme)
	_ = imagesv1alpha1.AddToScheme(scheme)
	_ = k8score.AddToScheme(scheme)
	_ = certapiv1.AddToScheme(scheme)
	_ = k8net.AddToScheme(scheme)
	_ = istioclient.AddToScheme(scheme)
	_ = batchv1.AddToScheme(scheme)
	return scheme
}

// createResourceFromTemplate builds a resource by merging the data with the template file and then
// creates the resource using the client.
func createResourceFromTemplate(cli client.Client, template string, data interface{}) error {
	uns := unstructured.Unstructured{}
	if err := updateUnstructuredFromYAMLTemplate(&uns, template, data); err != nil {
		return err
	}
	if err := cli.Create(context.Background(), &uns); err != nil {
		return err
	}
	return nil
}

// updateUnstructuredFromYAMLTemplate updates an unstructured from a populated YAML template file.
// uns - The unstructured to update
// template - The template file
// params - The param maps to merge into the template
func updateUnstructuredFromYAMLTemplate(uns *unstructured.Unstructured, template string, data interface{}) error {
	str, err := executeTemplate(template, data)
	if err != nil {
		return err
	}
	ybytes, err := yaml.YAMLToJSON([]byte(str))
	if err != nil {
		return err
	}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(ybytes, nil, uns)
	if err != nil {
		return err
	}
	return nil
}

// executeTemplate reads a template from a file and replaces values in the template from param maps
// template - The filename of a template
// params - a vararg of param maps
func executeTemplate(templateFile string, data interface{}) (string, error) {
	file := "../../" + templateFile
	if _, err := os.Stat(file); err != nil {
		file = "../" + templateFile
		if _, err := os.Stat(file); err != nil {
			file = templateFile
			if _, err := os.Stat(file); err != nil {
				return "", err
			}
		}
	}
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	t, err := template.New(templateFile).Parse(string(b))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = t.ExecuteTemplate(&buf, templateFile, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
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

// newImageBuildRequestReconciler creates a new reconciler for testing
// c - The Kerberos client to inject into the reconciler
func newImageBuildRequestReconciler(c client.Client) ImageBuildRequestReconciler {
	scheme := newScheme()
	//controller := image
	reconciler := ImageBuildRequestReconciler{
		Client: c,
		Scheme: scheme}
	return reconciler
}
