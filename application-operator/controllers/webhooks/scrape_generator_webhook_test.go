// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// newScrapeGeneratorWebhook creates a new ScrapeGeneratorWebhook
func newScrapeGeneratorWebhook() ScrapeGeneratorWebhook {
	scheme := newScheme()
	scheme.AddKnownTypes(schema.GroupVersion{
		Version: "v1",
	}, &corev1.Pod{}, &appsv1.Deployment{}, &appsv1.ReplicaSet{}, &appsv1.StatefulSet{})
	decoder, _ := admission.NewDecoder(scheme)
	cli := ctrlfake.NewFakeClientWithScheme(scheme)
	v := ScrapeGeneratorWebhook{
		Client:        cli,
		Decoder:       decoder,
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}
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
	v.createNamespace(t, "test", nil)
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testPod))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Pod", testPod)
	res := v.Handle(context.TODO(), req)
	assert.True(res.Allowed, "Expected validation to succeed.")
}

// TestHandleDeployment tests the handling of a Deployment resource
// GIVEN a call validate Deployment on create or update
// WHEN the Deployment is properly formed
// THEN the validation should succeed
func TestHandleDeployment(t *testing.T) {
	assert := assert.New(t)
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", nil)
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testDeployment))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(res.Allowed, "Expected validation to succeed.")
}

// TestHandleReplicaSet tests the handling of a ReplicaSet resource
// GIVEN a call validate ReplicaSet on create or update
// WHEN the ReplicaSet is properly formed
// THEN the validation should succeed
func TestHandleReplicaSet(t *testing.T) {
	assert := assert.New(t)
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", nil)
	testReplicaSet := appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testReplicaSet))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "ReplicaSet", testReplicaSet)
	res := v.Handle(context.TODO(), req)
	assert.True(res.Allowed, "Expected validation to succeed.")
}

// TestHandleStatefulSet tests the handling of a StatefulSet resource
// GIVEN a call validate StatefulSet on create or update
// WHEN the StatefulSet is properly formed
// THEN the validation should succeed
func TestHandleStatefulSet(t *testing.T) {
	assert := assert.New(t)
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", nil)
	testStatefulSet := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testStatefulSet))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "StatefulSet", testStatefulSet)
	res := v.Handle(context.TODO(), req)
	assert.True(res.Allowed, "Expected validation to succeed.")
}

func (v *ScrapeGeneratorWebhook) createNamespace(t *testing.T, name string, labels map[string]string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	_, err := v.KubeClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	assert.NoError(t, err, "unexpected error creating namespace")
}
