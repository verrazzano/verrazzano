// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"
	"text/template"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	"github.com/golang/mock/gomock"
	certapiv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	asserts "github.com/stretchr/testify/assert"
	imagesv1alpha1 "github.com/verrazzano/verrazzano/image-patch-operator/api/images/v1alpha1"
	"github.com/verrazzano/verrazzano/image-patch-operator/mocks"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	k8sapps "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	k8score "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	imagesv1alpha1.AddToScheme(scheme)
	reconciler = ImageBuildRequestReconciler{Client: cli, Scheme: scheme}
	mgr.EXPECT().GetConfig().Return(&rest.Config{})
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(log.NullLogger{})
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
	cli := fake.NewFakeClientWithScheme(newScheme())
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

	request := newRequest("default", "cluster1")
	reconciler := newImageBuildRequestReconciler(cli)
	_, err := reconciler.Reconcile(request)
	assert.NoError(err)

	ibr := &imagesv1alpha1.ImageBuildRequest{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "cluster1"}, ibr)

	// Ensure that IBR resource exists
	assert.NoError(err)

	// The status, condition, and message of the IBR should reflect that the ImageJob is in progress
	assert.Equal(ibr.Status.State, imagesv1alpha1.StateType("Building"))
	assert.Equal(ibr.Status.Conditions[0].Type, imagesv1alpha1.ConditionType("BuildStarted"))
	assert.Equal(ibr.Status.Conditions[0].Message, "ImageBuildRequest build in progress")

	jb := &batchv1.Job{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "verrazzano-images-cluster1"}, jb)

	// Ensure that a Kubernetes job is created when an IBR is created
	assert.NoError(err)

	// Testing that the spec fields of the IBR propagate to the environmental variables of the ImageJob
	assert.Equal(jb.Spec.Template.Spec.Containers[0].Env[0].Value, "test-build")
	assert.Equal(jb.Spec.Template.Spec.Containers[0].Env[1].Value, "test-tag")
	assert.Equal(jb.Spec.Template.Spec.Containers[0].Env[2].Value, "phx.ocir.io")
	assert.Equal(jb.Spec.Template.Spec.Containers[0].Env[3].Value, "myrepo/verrazzano")
	assert.Equal(jb.Spec.Template.Spec.Containers[0].Env[4].Value, "ghcr.io/oracle/oraclelinux:8-slim")
	assert.Equal(jb.Spec.Template.Spec.Containers[0].Env[5].Value, "jdk-8u281-linux-x64.tar.gz")
	assert.Equal(jb.Spec.Template.Spec.Containers[0].Env[6].Value, "8u281")
	assert.Equal(jb.Spec.Template.Spec.Containers[0].Env[7].Value, "fmw_12.2.1.4.0_wls.jar")
	assert.Equal(jb.Spec.Template.Spec.Containers[0].Env[8].Value, "12.2.1.4.0")

	// Verifying that the PV, PVC, and Volume Mount are present on the created job
	assert.Equal(jb.Spec.Template.Spec.Volumes[1].Name, "installers-storage")
	assert.Equal(jb.Spec.Template.Spec.Volumes[1].PersistentVolumeClaim.ClaimName, "installers-storage-claim")
	assert.Equal(jb.Spec.Template.Spec.Containers[0].VolumeMounts[1].Name, "installers-storage")
	assert.Equal(jb.Spec.Template.Spec.Containers[0].VolumeMounts[1].MountPath, "/installers")

}

// TestIBRJobSucceeded tests the Reconcile method for the following:
// GIVEN an ImageBuildRequest is applied
// WHEN the ImageJob is completed
// THEN verify that the status of the ImageBuildRequest reflects the build is completed
func TestIBRJobSucceeded(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewFakeClientWithScheme(newScheme())
	params := map[string]string{
		"IBR_NAME": "cluster1",
	}

	// Creating an ImageBuildRequest resource and Kubernetes job in completed state
	assert.NoError(createResourceFromTemplate(cli, "test/templates/imagebuildrequest_instance.yaml", params))
	assert.NoError(createResourceFromTemplate(cli, "test/templates/job_succeeded.yaml", params))

	request := newRequest("default", "cluster1")
	reconciler := newImageBuildRequestReconciler(cli)
	_, err := reconciler.Reconcile(request)
	assert.NoError(err)

	// Testing if ImageBuildRequest status reflects that ImageJob is complete
	ibr := &imagesv1alpha1.ImageBuildRequest{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "cluster1"}, ibr)
	assert.NoError(err)
	assert.Equal(ibr.Status.State, imagesv1alpha1.StateType("Published"))
	assert.Equal(ibr.Status.Conditions[0].Type, imagesv1alpha1.ConditionType("BuildCompleted"))
	assert.Equal(ibr.Status.Conditions[0].Message, "ImageBuildRequest build completed successfully")

	jb := &batchv1.Job{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "verrazzano-images-cluster1"}, jb)
	assert.NoError(err)
	assert.Equal(jb.Status.Succeeded, int32(1))
}

// TestIBRJobFailed tests the Reconcile method for the following:
// GIVEN an ImageBuildRequest is applied
// WHEN the ImageJob fails
// THEN verify that the status of the ImageBuildRequest reflects the build failed
func TestIBRJobFailed(t *testing.T) {
	assert := asserts.New(t)
	cli := fake.NewFakeClientWithScheme(newScheme())
	params := map[string]string{
		"IBR_NAME": "cluster1",
	}

	// Creating an ImageBuildRequest resource and Kubernetes job in failed state
	assert.NoError(createResourceFromTemplate(cli, "test/templates/imagebuildrequest_instance.yaml", params))
	assert.NoError(createResourceFromTemplate(cli, "test/templates/job_failed.yaml", params))

	request := newRequest("default", "cluster1")
	reconciler := newImageBuildRequestReconciler(cli)
	_, err := reconciler.Reconcile(request)
	assert.NoError(err)

	// Testing if ImageBuildRequest status reflects that ImageJob failed
	ibr := &imagesv1alpha1.ImageBuildRequest{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "cluster1"}, ibr)
	assert.NoError(err)
	assert.Equal(ibr.Status.State, imagesv1alpha1.StateType("Failed"))
	assert.Equal(ibr.Status.Conditions[0].Type, imagesv1alpha1.ConditionType("BuildFailed"))
	assert.Equal(ibr.Status.Conditions[0].Message, "ImageBuildRequest build failed to complete")

	jb := &batchv1.Job{}
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "verrazzano-images-cluster1"}, jb)
	assert.NoError(err)
	assert.Equal(jb.Status.Succeeded, int32(0))
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	//_ = clientgoscheme.AddToScheme(scheme)
	core.AddToScheme(scheme)
	k8sapps.AddToScheme(scheme)
	imagesv1alpha1.AddToScheme(scheme)
	k8score.AddToScheme(scheme)
	certapiv1alpha2.AddToScheme(scheme)
	k8net.AddToScheme(scheme)
	istioclient.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)
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
