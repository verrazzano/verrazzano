// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/verrazzano/verrazzano/application-operator/constants"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/stretchr/testify/assert"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// newScrapeGeneratorPodWebhook creates a new ScrapeGeneratorPodWebhook
func newScrapeGeneratorPodWebhook() ScrapeGeneratorPodWebhook {
	scheme := newScheme()
	scheme.AddKnownTypes(schema.GroupVersion{
		Version: "v1",
	}, &corev1.Namespace{}, &corev1.Pod{}, &appsv1.Deployment{}, &appsv1.ReplicaSet{}, &appsv1.StatefulSet{})
	vzapp.AddToScheme(scheme)
	decoder, _ := admission.NewDecoder(scheme)
	cli := ctrlfake.NewFakeClientWithScheme(scheme)
	v := ScrapeGeneratorPodWebhook{
		Client:        cli,
		DynamicClient: fake.NewSimpleDynamicClient(runtime.NewScheme()),
	}
	v.InjectDecoder(decoder)
	return v
}

// TestNoOwnerReferences tests the handling of a Pod resource
// GIVEN a call to the webhook Handle function
// WHEN the pod resource has no owner references
// THEN the Handle function should succeed and the pod is not mutated
func TestNoOwnerReferences(t *testing.T) {
	a := newScrapeGeneratorPodWebhook()

	// Test data
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}
	assert.NoError(t, a.Client.Create(context.TODO(), &pod))

	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := a.Handle(context.TODO(), req)

	assert.True(t, res.Allowed)
	assert.Len(t, res.Patches, 0)
}

// TestOwnerReferenceNoLabel tests the handling of a Pod resource
// GIVEN a call to the webhook Handle function
// WHEN the pod resource has one owner reference and that owner reference
//   is missing the app.verrazzano.io/metrics-binding label
// THEN the Handle function should succeed and the pod is not mutated
func TestOwnerReferenceNoLabel(t *testing.T) {
	a := newScrapeGeneratorPodWebhook()

	// Create a replica set with no owner reference
	u := newUnstructured("apps/v1", "ReplicaSet", "test-replicaSet")
	resource := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "replicasets",
	}
	_, err := a.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating replica set")

	// Create the pod with an owner reference
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "test-replicaSet",
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
			},
		},
	}
	assert.NoError(t, a.Client.Create(context.TODO(), &pod))

	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := a.Handle(context.TODO(), req)

	assert.True(t, res.Allowed)
	assert.Empty(t, res.Patches)
}

// TestMultipleOwnerReferenceNoLabel tests the handling of a Pod resource
// GIVEN a call to the webhook Handle function
// WHEN the pod resource has nested owner reference and all owner references
//   are missing the app.verrazzano.io/metrics-binding label
// THEN the Handle function should succeed and the pod is not mutated
func TestMultipleOwnerReferenceNoLabel(t *testing.T) {
	a := newScrapeGeneratorPodWebhook()

	// Create a deployment with no owner reference
	u := newUnstructured("apps/v1", "Deployment", "test-deployment")
	resource := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	_, err := a.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating deployment")

	// Create a replica set with an owner reference
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
	_, err = a.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating replica set")

	// Create the pod with an owner reference
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "test-replicaSet",
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
			},
		},
	}
	assert.NoError(t, a.Client.Create(context.TODO(), &pod))

	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := a.Handle(context.TODO(), req)

	assert.True(t, res.Allowed)
	assert.Empty(t, res.Patches)
}

// TestOwnerReferenceLabel tests the handling of a Pod resource
// GIVEN a call to the webhook Handle function
// WHEN the pod resource has one owner reference and that owner reference
//   contains the app.verrazzano.io/metrics-binding label
// THEN the Handle function should succeed and the pod is mutated
func TestOwnerReferenceLabel(t *testing.T) {
	a := newScrapeGeneratorPodWebhook()

	// Create a replica set with no owner reference
	u := newUnstructured("apps/v1", "ReplicaSet", "test-replicaSet")
	resource := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "replicasets",
	}
	u.SetLabels(map[string]string{constants.MetricsBindingLabel: "testValue"})
	_, err := a.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating replica set")

	// Create the pod with an owner reference
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "test-replicaSet",
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
			},
		},
	}
	assert.NoError(t, a.Client.Create(context.TODO(), &pod))

	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := a.Handle(context.TODO(), req)

	assert.True(t, res.Allowed)
	assert.Len(t, res.Patches, 1)
	assert.Equal(t, "add", res.Patches[0].Operation)
	assert.Equal(t, "/metadata/labels", res.Patches[0].Path)
	assert.Contains(t, res.Patches[0].Value, constants.MetricsBindingLabel)
}

// TestMultipleOwnerReferenceLabel tests the handling of a Pod resource
// GIVEN a call to the webhook Handle function
// WHEN the pod resource has nested owner references and the 2nd owner reference
//   contains the app.verrazzano.io/metrics-binding label
// THEN the Handle function should succeed and the pod is mutated
func TestMultipleOwnerReferenceLabel(t *testing.T) {
	a := newScrapeGeneratorPodWebhook()

	// Create a deployment with no owner reference
	u := newUnstructured("apps/v1", "Deployment", "test-deployment")
	resource := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	u.SetLabels(map[string]string{constants.MetricsBindingLabel: "testValue"})
	_, err := a.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating deployment")

	// Create a replica set with an owner reference
	u = newUnstructured("apps/v1", "ReplicaSet", "test-replicaSet")
	resource = schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "replicasets",
	}
	ownerReferences := []metav1.OwnerReference{
		{
			Name:       "test-deployment",
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
	}
	u.SetOwnerReferences(ownerReferences)
	_, err = a.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating replica set")

	// Create the pod with an owner reference
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "test-replicaSet",
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
			},
		},
	}
	assert.NoError(t, a.Client.Create(context.TODO(), &pod))

	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := a.Handle(context.TODO(), req)

	assert.True(t, res.Allowed)
	assert.Len(t, res.Patches, 1)
	assert.Equal(t, "add", res.Patches[0].Operation)
	assert.Equal(t, "/metadata/labels", res.Patches[0].Path)
	assert.Contains(t, res.Patches[0].Value, constants.MetricsBindingLabel)
}
