// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"
)

// TestIstioDefaulterHandleError tests handling an invalid admission.Request
// GIVEN an IstioWebhook and an admission.Request
//  WHEN Handle is called with an invalid admission.Request containing no content
//  THEN Handle should return an error with http.StatusBadRequest
func TestIstioDefaulterHandleError(t *testing.T) {
	decoder := decoder()
	defaulter := &IstioWebhook{}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, int32(http.StatusBadRequest), res.Result.Code)
}

// TestIstioDefaulterHandleNoAction tests handling an admission.Request
// GIVEN a IstioWebhook and an admission.Request
//  WHEN Handle is called with an admission.Request containing a pod resource
//  THEN Handle should return an Allowed response with no action required
func TestIstioDefaulterHandleNoAction(t *testing.T) {
	decoder := decoder()
	defaulter := &IstioWebhook{}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: podReadYaml2Json(t, "simple-pod.yaml")}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, v1.StatusReason("No action required, pod was not created from an ApplicationConfiguration resource"), res.Result.Reason)
}

func podReadYaml2Json(t *testing.T, path string) []byte {
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
