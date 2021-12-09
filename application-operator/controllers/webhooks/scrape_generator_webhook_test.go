// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// newScrapeGeneratorWebhook creates a new ScrapeGeneratorWebhook
func newScrapeGeneratorWebhook() ScrapeGeneratorWebhook {
	scheme := newScheme()
	scheme.AddKnownTypes(schema.GroupVersion{
		Version: "v1",
	}, &corev1.Pod{})
	decoder, _ := admission.NewDecoder(scheme)
	cli := fake.NewFakeClientWithScheme(scheme)
	v := ScrapeGeneratorWebhook{Client: cli, Decoder: decoder}
	return v
}

// newScrapeGeneratorRequest creates a new admissionRequest with the provided operation and object.
func newScrapeGeneratorRequest(op admissionv1beta1.Operation, kind string, obj interface{}) admission.Request {
	raw := runtime.RawExtension{}
	bytes, _ := json.Marshal(obj)
	raw.Raw = bytes
	req := admission.Request{
		AdmissionRequest: admissionv1beta1.AdmissionRequest{
			Kind: metav1.GroupVersionKind{
				Kind: kind,
			},
			Operation: op, Object: raw}}
	return req
}

// TestHandlePod tests the handling of a Pod resource
// GIVEN a call validate Pod on create or update
// WHEN the Pod is properly formed
// THEN the validation should succeed
func TestHandlePod(t *testing.T) {
	assert := assert.New(t)
	v := newScrapeGeneratorWebhook()

	// Test data
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	// Test data
	assert.NoError(v.Client.Create(context.TODO(), &testPod))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Pod", testPod)
	res := v.Handle(context.TODO(), req)
	assert.True(res.Allowed, "Expected validation to succeed.")
	assert.Equal(res.AdmissionResponse.Result.Reason, metav1.StatusReason(StatusReasonSuccess))
}
