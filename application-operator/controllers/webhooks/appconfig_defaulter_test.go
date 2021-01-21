// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"net/http"
	"path/filepath"
	"testing"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	"k8s.io/apimachinery/pkg/runtime"
)

func decoder() *admission.Decoder {
	scheme := runtime.NewScheme()
	core.AddToScheme(scheme)
	decoder, err := admission.NewDecoder(scheme)
	if err != nil {
		log.Error(err, "error new decoder")
	}
	return decoder
}

// TestAppConfigDefaulterHandleError tests handling an invalid appconfig admission.Request
// GIVEN a AppConfigDefaulter and an appconfig admission.Request
//  WHEN Handle is called with an invalid admission.Request containing no content
//  THEN Handle should return an error with http.StatusBadRequest
func TestAppConfigDefaulterHandleError(t *testing.T) {
	decoder := decoder()
	defaulter := &AppConfigWebhook{}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, int32(http.StatusBadRequest), res.Result.Code)
}

// TestAppConfigDefaulterHandle tests handling an appconfig admission.Request
// GIVEN a AppConfigDefaulter and an appconfig admission.Request
//  WHEN Handle is called with an admission.Request containing appconfig
//  THEN Handle should return a patch response
func TestAppConfigDefaulterHandle(t *testing.T) {
	decoder := decoder()
	defaulter := &AppConfigWebhook{}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, "hello-conf.yaml")}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.NotEqual(t, 0, len(res.Patches))
}

// TestAppConfigDefaulterHandle tests handling an appconfig Delete admission.Request
// GIVEN a AppConfigDefaulter and an appconfig Delete admission.Request
//  WHEN Handle is called with an admission.Request containing appconfig
//  THEN Handle should return an Allowed response with no patch
func TestAppConfigDefaulterHandleDelete(t *testing.T) {
	decoder := decoder()
	defaulter := &AppConfigWebhook{}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{
		AdmissionRequest: admissionv1beta1.AdmissionRequest{Operation: v1beta1.Delete},
	}
	req.OldObject = runtime.RawExtension{Raw: readYaml2Json(t, "hello-conf.yaml")}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, 0, len(res.Patches))
}

// TestAppConfigDefaulterHandleMarshalError tests handling an appconfig with json marshal error
// GIVEN a AppConfigDefaulter with mock appconfigMarshalFunc
//  WHEN Handle is called with an admission.Request containing appconfig
//  THEN Handle should return error with http.StatusInternalServerError
func TestAppConfigDefaulterHandleMarshalError(t *testing.T) {
	decoder := decoder()
	defaulter := &AppConfigWebhook{}
	defaulter.InjectDecoder(decoder)
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

func (*mockErrorDefaulter) Default(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error {
	return fmt.Errorf("mockErrorDefaulter error")
}

func (*mockErrorDefaulter) Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error {
	return nil
}

// TestAppConfigDefaulterHandleDefaultError tests handling a defaulter error
// GIVEN a AppConfigDefaulter with mock defaulter
//  WHEN Handle is called with an admission.Request containing appconfig
//  THEN Handle should return error with http.StatusInternalServerError
func TestAppConfigDefaulterHandleDefaultError(t *testing.T) {
	decoder := decoder()
	defaulter := &AppConfigWebhook{Defaulters: []AppConfigDefaulter{&mockErrorDefaulter{}}}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, "hello-conf.yaml")}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, int32(http.StatusInternalServerError), res.Result.Code)
}

func readYaml2Json(t *testing.T, path string) []byte {
	filename, _ := filepath.Abs(fmt.Sprintf("testdata/%s", path))
	yamlBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Error reading %v: %v", path, err)
	}
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		log.Error(err, "Error json marshal")
	}
	return jsonBytes
}
