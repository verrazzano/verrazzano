// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discofake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/restmapper"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// newLabelerPodWebhook creates a new LabelerPodWebhook
func newLabelerPodWebhook() LabelerPodWebhook {
	scheme := newScheme()
	scheme.AddKnownTypes(schema.GroupVersion{
		Version: "v1",
	}, &corev1.Namespace{}, &corev1.Pod{}, &appsv1.Deployment{}, &appsv1.ReplicaSet{}, &appsv1.StatefulSet{})
	vzapp.AddToScheme(scheme)
	decoder, _ := admission.NewDecoder(scheme)
	cli := ctrlfake.NewFakeClientWithScheme(scheme)
	clientset := k8sfake.NewSimpleClientset()
	discoveryClient := clientset.Discovery()
	fakeDiscovery := clientset.Discovery().(*discofake.FakeDiscovery)
	fakeDiscovery.Fake.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{
					Name:         "replicasets",
					SingularName: "replicaset",
					Kind:         "ReplicaSet",
					Namespaced:   true,
				},
				{
					Name:         "deployments",
					SingularName: "deployment",
					Kind:         "Deployment",
					Namespaced:   true,
				},
			},
		},
	}
	gr, _ := restmapper.GetAPIGroupResources(discoveryClient)
	v := LabelerPodWebhook{
		Client:        cli,
		DynamicClient: fake.NewSimpleDynamicClient(runtime.NewScheme()),
		RestMapper:    restmapper.NewDiscoveryRESTMapper(gr),
	}
	v.InjectDecoder(decoder)
	return v
}

// TestNoOwnerReferences tests the handling of a Pod resource
// GIVEN a call to the webhook Handle function
// WHEN the pod resource has no owner references
// THEN the Handle function should succeed and the pod is mutated
func TestNoOwnerReferences(t *testing.T) {
	a := newLabelerPodWebhook()

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
	assert.Len(t, res.Patches, 1)
	assert.Equal(t, "add", res.Patches[0].Operation)
	assert.Equal(t, "/metadata/labels", res.Patches[0].Path)
	assert.Contains(t, res.Patches[0].Value, constants.MetricsWorkloadLabel)
}

// TestOwnerReference tests the handling of a Pod resource
// GIVEN a call to the webhook Handle function
// WHEN the pod resource has one owner reference
// THEN the Handle function should succeed and the pod is mutated
func TestOwnerReference(t *testing.T) {
	a := newLabelerPodWebhook()

	// Create a replica set with no owner reference
	u := newUnstructured("apps/v1", "ReplicaSet", "test-replicaSet")
	resource := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "replicasets",
	}
	u.SetLabels(map[string]string{constants.MetricsWorkloadLabel: "testValue"})
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
	assert.Contains(t, res.Patches[0].Value, constants.MetricsWorkloadLabel)
}

// TestMultipleOwnerReference tests the handling of a Pod resource
// GIVEN a call to the webhook Handle function
// WHEN the pod resource has nested owner references and the 2nd owner reference
//   is the workload resource
// THEN the Handle function should succeed and the pod is mutated
func TestMultipleOwnerReference(t *testing.T) {
	a := newLabelerPodWebhook()

	// Create a deployment with no owner reference
	u := newUnstructured("apps/v1", "Deployment", "test-deployment")
	resource := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	u.SetLabels(map[string]string{constants.MetricsWorkloadLabel: "testValue"})
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
	assert.Contains(t, res.Patches[0].Value, constants.MetricsWorkloadLabel)
}

// TestMultipleOwnerReferenceAndWorkloadResources tests the handling of a Pod resource
// GIVEN a call to the webhook Handle function
// WHEN the pod resource has nested owner references and two owner references are found to be
//   a workload resource
// THEN the Handle function should fail and return an error
func TestMultipleOwnerReferenceAndWorkloadResources(t *testing.T) {
	a := newLabelerPodWebhook()

	// Create a deployment with no owner reference
	u := newUnstructured("apps/v1", "Deployment", "test-deployment")
	resource := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	u.SetLabels(map[string]string{constants.MetricsWorkloadLabel: "testValue"})
	_, err := a.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating deployment")

	// Create another deployment with no owner reference
	u = newUnstructured("apps/v1", "Deployment", "test-deployment2")
	resource = schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	u.SetLabels(map[string]string{constants.MetricsWorkloadLabel: "testValue"})
	_, err = a.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating deployment")

	// Create a replica set with two owner references
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
		{
			Name:       "test-deployment2",
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

	assert.False(t, res.Allowed)
	assert.Equal(t, "multiple workload resources found for test, Verrazzano metrics cannot be enabled", res.Result.Message)
}
