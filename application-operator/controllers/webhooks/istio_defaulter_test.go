// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	istiofake "istio.io/client-go/pkg/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"
)

var defaulter = &IstioWebhook{
	DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
	KubeClient:    fake.NewSimpleClientset(),
	IstioClient:   istiofake.NewSimpleClientset(),
}

// TestHandleBadRequest tests handling an invalid admission.Request
// GIVEN an IstioWebhook and an admission.Request
//  WHEN Handle is called with an invalid admission.Request containing no content
//  THEN Handle should return an error with http.StatusBadRequest
func TestHandleBadRequest(t *testing.T) {
	decoder := decoder()
	defaulter := &IstioWebhook{}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, int32(http.StatusBadRequest), res.Result.Code)
}

// TestHandleIstioDisabled tests handling an admission.Request
// GIVEN a IstioWebhook and an admission.Request
//  WHEN Handle is called with an admission.Request containing a pod resource with Istio disabled
//  THEN Handle should return an Allowed response with no action required
func TestHandleIstioDisabled(t *testing.T) {
	decoder := decoder()
	defaulter := &IstioWebhook{}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: podReadYaml2Json(t, "istio-disabled.yaml")}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, v1.StatusReason("No action required, pod labeled with sidecar.istio.io/inject: false"), res.Result.Reason)
}

// TestHandleNoOnwerReference tests handling an admission.Request
// GIVEN a IstioWebhook and an admission.Request
//  WHEN Handle is called with an admission.Request containing a pod resource with no owner references
//  THEN Handle should return an Allowed response with no action required
func TestHandleNoOnwerReference(t *testing.T) {
	decoder := decoder()
	defaulter := &IstioWebhook{}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: podReadYaml2Json(t, "simple-pod.yaml")}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, v1.StatusReason("No action required, pod is not a child of an ApplicationConfiguration resource"), res.Result.Reason)
}

// TestHandleNoAppConfigOnwerReference tests handling an admission.Request
// GIVEN a IstioWebhook and an admission.Request
//  WHEN Handle is called with an admission.Request containing a pod resource with no parent appconfig owner references
//  THEN Handle should return an Allowed response with no action required
func TestHandleNoAppConfigOnwerReference(t *testing.T) {
	decoder := decoder()
	u := newUnstructured("apps/v1", "Deployment", "test-deployment")
	resource := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	_, err := defaulter.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating deployment")

	u = newUnstructured("apps/v1", "ReplicaSet", "test-replicaSet")
	ownerReferences := []metav1.OwnerReference{
		{
			Name:       "test-deployment",
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
	}
	u.SetOwnerReferences(ownerReferences)
	resource = schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "replicasets",
	}
	_, err = defaulter.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating replica set")

	u = newUnstructured("v1", "Pod", "test-pod1")
	ownerReferences = []metav1.OwnerReference{
		{
			Name:       "test-replicaSet",
			Kind:       "ReplicaSet",
			APIVersion: "apps/v1",
		},
	}
	u.SetOwnerReferences(ownerReferences)
	resource = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}
	pod, err := defaulter.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, v1.StatusReason("No action required, pod is not a child of an ApplicationConfiguration resource"), res.Result.Reason)
}

// TestHandleAppConfigOnwerReference tests handling an admission.Request
// GIVEN a IstioWebhook and an admission.Request
//  WHEN Handle is called with an admission.Request containing a pod resource with a parent appconfig owner references
//  THEN Handle should return an Allowed response with patch values
func TestHandleAppConfigOnwerReference(t *testing.T) {
	decoder := decoder()
	u := newUnstructured("core.oam.dev/v1alpha2", "ApplicationConfiguration", "test-appconfig")
	resource := schema.GroupVersionResource{
		Group:    "core.oam.dev",
		Version:  "v1alpha2",
		Resource: "applicationconfigurations",
	}
	_, err := defaulter.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating application config")

	u = newUnstructured("v1", "Pod", "test-pod2")
	ownerReferences := []metav1.OwnerReference{
		{
			Name:       "test-appconfig",
			Kind:       "ApplicationConfiguration",
			APIVersion: "core.oam.dev/v1alpha2",
		},
	}
	u.SetOwnerReferences(ownerReferences)
	resource = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}
	pod, err := defaulter.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.NotEmpty(t, res.Patches)
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

func newUnstructured(apiVersion string, kind string, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": "default",
				"name":      name,
			},
		},
	}
}
