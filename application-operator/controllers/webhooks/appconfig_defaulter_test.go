// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"go.uber.org/zap"
	istiofake "istio.io/client-go/pkg/clientset/versioned/fake"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"
)

func decoder() *admission.Decoder {
	scheme := runtime.NewScheme()
	_ = core.AddToScheme(scheme)
	decoder, err := admission.NewDecoder(scheme)
	if err != nil {
		zap.S().Errorf("Failed creating new decoder: %v", err)
	}
	return decoder
}

// TestAppConfigDefaulterHandleError tests handling an invalid appconfig admission.Request
// GIVEN a AppConfigDefaulter and an appconfig admission.Request
//
//	WHEN Handle is called with an invalid admission.Request containing no content
//	THEN Handle should return an error with http.StatusBadRequest
func TestAppConfigDefaulterHandleError(t *testing.T) {

	decoder := decoder()
	defaulter := &AppConfigWebhook{}
	_ = defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, int32(http.StatusBadRequest), res.Result.Code)
}

// TestAppConfigDefaulterHandle tests handling an appconfig admission.Request
// GIVEN a AppConfigDefaulter and an appconfig admission.Request
//
//	WHEN Handle is called with an admission.Request containing appconfig
//	THEN Handle should return a patch response
func TestAppConfigDefaulterHandle(t *testing.T) {

	decoder := decoder()
	defaulter := &AppConfigWebhook{}
	_ = defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, "hello-conf.yaml")}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.NotEqual(t, 0, len(res.Patches))
}

// TestAppConfigWebhookHandleDelete tests handling an appconfig Delete admission.Request
// GIVEN a AppConfigWebhook and an appconfig Delete admission.Request
//
//	WHEN Handle is called with an admission.Request containing appconfig
//	THEN Handle should return an Allowed response with no patch
func TestAppConfigWebhookHandleDelete(t *testing.T) {

	testAppConfigWebhookHandleDelete(t, true, true, false)
}

// TestAppConfigWebhookHandleDeleteDryRun tests handling a dry run appconfig Delete admission.Request
// GIVEN a AppConfigWebhook and an appconfig Delete admission.Request
//
//	WHEN Handle is called with an admission.Request containing appconfig and set for a dry run
//	THEN Handle should return an Allowed response with no patch
func TestAppConfigWebhookHandleDeleteDryRun(t *testing.T) {

	testAppConfigWebhookHandleDelete(t, true, true, true)
}

// TestAppConfigWebhookHandleDeleteCertNotFound tests handling an appconfig Delete admission.Request where the app config cert is not found
// GIVEN a AppConfigWebhook and an appconfig Delete admission.Request
//
//	WHEN Handle is called with an admission.Request containing appconfig and the cert is not found
//	THEN Handle should return an Allowed response with no patch
func TestAppConfigWebhookHandleDeleteCertNotFound(t *testing.T) {

	testAppConfigWebhookHandleDelete(t, false, true, false)
}

// TestAppConfigWebhookHandleDeleteSecretNotFound tests handling an appconfig Delete admission.Request where the app config secret is not found
// GIVEN a AppConfigWebhook and an appconfig Delete admission.Request
//
//	WHEN Handle is called with an admission.Request containing appconfig and the secret is not found
//	THEN Handle should return an Allowed response with no patch
func TestAppConfigWebhookHandleDeleteSecretNotFound(t *testing.T) {

	testAppConfigWebhookHandleDelete(t, true, false, false)
}

// TestAppConfigDefaulterHandleMarshalError tests handling an appconfig with json marshal error
// GIVEN a AppConfigDefaulter with mock appconfigMarshalFunc
//
//	WHEN Handle is called with an admission.Request containing appconfig
//	THEN Handle should return error with http.StatusInternalServerError
func TestAppConfigDefaulterHandleMarshalError(t *testing.T) {

	decoder := decoder()
	defaulter := &AppConfigWebhook{}
	_ = defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, "hello-conf.yaml")}
	appconfigMarshalFunc = func(v interface{}) ([]byte, error) {
		return nil, fmt.Errorf("json marshal error")
	}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, int32(http.StatusInternalServerError), res.Result.Code)
}

type mockErrorDefaulter struct {
}

func (*mockErrorDefaulter) Default(appConfig *oamv1.ApplicationConfiguration, dryRun bool, log *zap.SugaredLogger) error {
	return fmt.Errorf("mockErrorDefaulter error")
}

func (*mockErrorDefaulter) Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool, log *zap.SugaredLogger) error {
	return nil
}

// TestAppConfigDefaulterHandleDefaultError tests handling a defaulter error
// GIVEN a AppConfigDefaulter with mock defaulter
//
//	WHEN Handle is called with an admission.Request containing appconfig
//	THEN Handle should return error with http.StatusInternalServerError
func TestAppConfigDefaulterHandleDefaultError(t *testing.T) {

	decoder := decoder()
	defaulter := &AppConfigWebhook{Defaulters: []AppConfigDefaulter{&mockErrorDefaulter{}}}
	_ = defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, "hello-conf.yaml")}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, int32(http.StatusInternalServerError), res.Result.Code)
}

// TestHandleFailed tests to make sure the failure metric is being exposed
func TestHandleFailed(t *testing.T) {

	assert := assert.New(t)
	// Create a request and decode(Handle) it
	decoder := decoder()
	defaulter := &AppConfigWebhook{}
	_ = defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	defaulter.Handle(context.TODO(), req)
	reconcileerrorCounterObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.AppconfigHandleError)
	assert.NoError(err)
	// Expect a call to fetch the error
	reconcileFailedCounterBefore := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	reconcileerrorCounterObject.Get().Inc()
	reconcileFailedCounterAfter := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	assert.Equal(reconcileFailedCounterBefore, reconcileFailedCounterAfter-1)
}

func testAppConfigWebhookHandleDelete(t *testing.T, certFound, secretFound, dryRun bool) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	if !dryRun {
		// list projects
		mockClient.EXPECT().
			List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any())

	}
	decoder := decoder()

	webhook := &AppConfigWebhook{
		Client:      mockClient,
		KubeClient:  fake.NewSimpleClientset(),
		IstioClient: istiofake.NewSimpleClientset(),
	}
	_ = webhook.InjectDecoder(decoder)
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{Operation: admissionv1.Delete},
	}
	req.OldObject = runtime.RawExtension{Raw: readYaml2Json(t, "hello-conf.yaml")}
	if dryRun {
		dryRun := true
		req.DryRun = &dryRun
	}
	res := webhook.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, 0, len(res.Patches))
}

func readYaml2Json(t *testing.T, path string) []byte {
	filename, _ := filepath.Abs(fmt.Sprintf("testdata/%s", path))
	yamlBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Error reading %v: %v", path, err)
	}
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		zap.S().Errorf("Failed json marshal: %v", err)
	}
	return jsonBytes
}
