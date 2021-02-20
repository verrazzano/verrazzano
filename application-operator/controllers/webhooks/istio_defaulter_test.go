// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	istiofake "istio.io/client-go/pkg/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// TestHandleBadRequest tests handling an invalid admission.Request
// GIVEN an IstioWebhook and an admission.Request
//  WHEN Handle is called with an invalid admission.Request containing no content
//  THEN Handle should return an error with http.StatusBadRequest
func TestHandleBadRequest(t *testing.T) {
	decoder := decoder()
	defaulter := &IstioWebhook{}
	err := defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
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
	defaulter := &IstioWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
		IstioClient:   istiofake.NewSimpleClientset(),
	}
	// Create a pod with Istio injection disabled
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-disabled",
			Namespace: "default",
			Annotations: map[string]string{
				"sidecar.istio.io/inject": "false",
			},
		},
	}
	pod, err := defaulter.KubeClient.CoreV1().Pods("default").Create(context.TODO(), p, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	decoder := decoder()
	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, v1.StatusReason("No action required, pod labeled with sidecar.istio.io/inject: false"), res.Result.Reason)
}

// TestHandleNoOnwerReference tests handling an admission.Request
// GIVEN a IstioWebhook and an admission.Request
//  WHEN Handle is called with an admission.Request containing a pod resource with no owner references
//  THEN Handle should return an Allowed response with no action required
func TestHandleNoOnwerReference(t *testing.T) {
	defaulter := &IstioWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
		IstioClient:   istiofake.NewSimpleClientset(),
	}
	// Create a simple pod
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-pod",
			Namespace: "default",
		},
	}
	pod, err := defaulter.KubeClient.CoreV1().Pods("default").Create(context.TODO(), p, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	decoder := decoder()
	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, v1.StatusReason("No action required, pod is not a child of an ApplicationConfiguration resource"), res.Result.Reason)
}

// TestHandleNoAppConfigOnwerReference tests handling an admission.Request
// GIVEN a IstioWebhook and an admission.Request
//  WHEN Handle is called with an admission.Request containing a pod resource with no parent appconfig owner references
//  THEN Handle should return an Allowed response with no action required
func TestHandleNoAppConfigOnwerReference(t *testing.T) {
	defaulter := &IstioWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
		IstioClient:   istiofake.NewSimpleClientset(),
	}

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

	u = newUnstructured("v1", "Pod", "test-pod")
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

	decoder := decoder()
	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, v1.StatusReason("No action required, pod is not a child of an ApplicationConfiguration resource"), res.Result.Reason)
}

// TestHandleAppConfigOnwerReference1 tests handling an admission.Request
// GIVEN a IstioWebhook and an admission.Request
//  WHEN Handle is called with an admission.Request containing a pod resource with a parent appconfig owner reference
//    and a default service account referenced by the pod
//  THEN Handle should return an Allowed response with patch values
func TestHandleAppConfigOnwerReference1(t *testing.T) {
	defaulter := &IstioWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
		IstioClient:   istiofake.NewSimpleClientset(),
	}

	// Create an applicationConfiguration resource
	u := newUnstructured("core.oam.dev/v1alpha2", "ApplicationConfiguration", "test-appconfig")
	resource := schema.GroupVersionResource{
		Group:    "core.oam.dev",
		Version:  "v1alpha2",
		Resource: "applicationconfigurations",
	}
	_, err := defaulter.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating application config")

	// Create a pod without specifying a service account
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "test-appconfig",
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
			},
		},
	}
	pod, err := defaulter.KubeClient.CoreV1().Pods("default").Create(context.TODO(), p, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	decoder := decoder()
	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.NotEmpty(t, res.Patches)

	// Get the authorization policy resource we created and do some validations
	authPolicy, err := defaulter.IstioClient.SecurityV1beta1().AuthorizationPolicies("default").Get(context.TODO(), "test-appconfig", metav1.GetOptions{})
	assert.NoError(t, err, "Unexpected error getting authorization policy")
	assert.Equal(t, authPolicy.Name, "test-appconfig")
	assert.Equal(t, authPolicy.Namespace, "default")
	assert.Contains(t, authPolicy.Labels, IstioAppLabel)
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Name, "test-appconfig")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Kind, "ApplicationConfiguration")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].APIVersion, "core.oam.dev/v1alpha2")
	assert.Contains(t, authPolicy.Spec.Selector.MatchLabels, IstioAppLabel)
	assert.Equal(t, len(authPolicy.Spec.GetRules()[0].From[0].Source.Principals), 2)
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account")
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/default/sa/test-appconfig")
}

// TestHandleAppConfigOnwerReference2 tests handling an admission.Request
// GIVEN a IstioWebhook and an admission.Request
//  WHEN Handle is called with an admission.Request containing a pod resource with a parent appconfig owner reference
//    and a non-default service account referenced by the pod
//  THEN Handle should return an Allowed response with patch values
func TestHandleAppConfigOnwerReference2(t *testing.T) {
	defaulter := &IstioWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
		IstioClient:   istiofake.NewSimpleClientset(),
	}

	// Create a non-default service account
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa",
			Namespace: "default"},
	}
	serviceAccount, err := defaulter.KubeClient.CoreV1().ServiceAccounts("default").Create(context.TODO(), sa, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating service account")

	// Create an applicationConfiguration resource
	u := newUnstructured("core.oam.dev/v1alpha2", "ApplicationConfiguration", "test-appconfig")
	resource := schema.GroupVersionResource{
		Group:    "core.oam.dev",
		Version:  "v1alpha2",
		Resource: "applicationconfigurations",
	}
	_, err = defaulter.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating application config")

	// Create a pod referencing the service account we created
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "test-appconfig",
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount.Name,
		},
	}
	pod, err := defaulter.KubeClient.CoreV1().Pods("default").Create(context.TODO(), p, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	decoder := decoder()
	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.NotEmpty(t, res.Patches)

	// Get the authorization policy resource we created and do some validations
	authPolicy, err := defaulter.IstioClient.SecurityV1beta1().AuthorizationPolicies("default").Get(context.TODO(), "test-appconfig", metav1.GetOptions{})
	assert.NoError(t, err, "Unexpected error getting authorization policy")
	assert.Equal(t, authPolicy.Name, "test-appconfig")
	assert.Equal(t, authPolicy.Namespace, "default")
	assert.Contains(t, authPolicy.Labels, IstioAppLabel)
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Name, "test-appconfig")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Kind, "ApplicationConfiguration")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].APIVersion, "core.oam.dev/v1alpha2")
	assert.Contains(t, authPolicy.Spec.Selector.MatchLabels, IstioAppLabel)
	assert.Equal(t, len(authPolicy.Spec.GetRules()[0].From[0].Source.Principals), 2)
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account")
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/default/sa/test-sa")
}

// TestHandleAppConfigOnwerReference3 tests handling an admission.Request
// GIVEN a IstioWebhook and an admission.Request
//  WHEN Handle is called twice with an admission.Request containing a pod resource with a parent appconfig owner reference
//    A different service account is used on each call.
//  THEN Handle should return an Allowed response with patch values
func TestHandleAppConfigOnwerReference3(t *testing.T) {
	defaulter := &IstioWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
		IstioClient:   istiofake.NewSimpleClientset(),
	}

	// Create a non-default service account
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa",
			Namespace: "default"},
	}
	serviceAccount, err := defaulter.KubeClient.CoreV1().ServiceAccounts("default").Create(context.TODO(), sa, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating service account")

	// Create an applicationConfiguration resource
	u := newUnstructured("core.oam.dev/v1alpha2", "ApplicationConfiguration", "test-appconfig")
	resource := schema.GroupVersionResource{
		Group:    "core.oam.dev",
		Version:  "v1alpha2",
		Resource: "applicationconfigurations",
	}
	_, err = defaulter.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating application config")

	// Create a pod referencing the service account we created
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod1",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "test-appconfig",
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount.Name,
		},
	}
	pod, err := defaulter.KubeClient.CoreV1().Pods("default").Create(context.TODO(), p, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	decoder := decoder()
	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.NotEmpty(t, res.Patches)

	// Get the authorization policy resource we created and do some validations
	authPolicy, err := defaulter.IstioClient.SecurityV1beta1().AuthorizationPolicies("default").Get(context.TODO(), "test-appconfig", metav1.GetOptions{})
	assert.NoError(t, err, "Unexpected error getting authorization policy")
	assert.Equal(t, authPolicy.Name, "test-appconfig")
	assert.Equal(t, authPolicy.Namespace, "default")
	assert.Contains(t, authPolicy.Labels, IstioAppLabel)
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Name, "test-appconfig")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Kind, "ApplicationConfiguration")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].APIVersion, "core.oam.dev/v1alpha2")
	assert.Contains(t, authPolicy.Spec.Selector.MatchLabels, IstioAppLabel)
	assert.Equal(t, len(authPolicy.Spec.GetRules()[0].From[0].Source.Principals), 2)
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account")
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/default/sa/test-sa")

	// Create a non-default service account, different than first one we created
	sa = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa2",
			Namespace: "default"},
	}
	serviceAccount, err = defaulter.KubeClient.CoreV1().ServiceAccounts("default").Create(context.TODO(), sa, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating service account")

	// Create a pod referencing the second service account we created
	p = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod2",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "test-appconfig",
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount.Name,
		},
	}
	pod, err = defaulter.KubeClient.CoreV1().Pods("default").Create(context.TODO(), p, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req = admission.Request{}
	req.Namespace = "default"
	marshaledPod, err = json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res = defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.NotEmpty(t, res.Patches)

	// Get the authorization policy resource we created and do some validations
	authPolicy, err = defaulter.IstioClient.SecurityV1beta1().AuthorizationPolicies("default").Get(context.TODO(), "test-appconfig", metav1.GetOptions{})
	assert.NoError(t, err, "Unexpected error getting authorization policy")
	assert.Equal(t, authPolicy.Name, "test-appconfig")
	assert.Equal(t, authPolicy.Namespace, "default")
	assert.Contains(t, authPolicy.Labels, IstioAppLabel)
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Name, "test-appconfig")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Kind, "ApplicationConfiguration")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].APIVersion, "core.oam.dev/v1alpha2")
	assert.Contains(t, authPolicy.Spec.Selector.MatchLabels, IstioAppLabel)
	assert.Equal(t, len(authPolicy.Spec.GetRules()[0].From[0].Source.Principals), 3)
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account")
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/default/sa/test-sa")
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/default/sa/test-sa2")
}

// TestHandleAppConfigOnwerReference4 tests handling an admission.Request
// GIVEN a IstioWebhook and an admission.Request
//  WHEN Handle is called twice with an admission.Request containing a pod resource with a parent appconfig owner reference
//    The same service account is used on each call.
//  THEN Handle should return an Allowed response with patch values
func TestHandleAppConfigOnwerReference4(t *testing.T) {
	defaulter := &IstioWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
		IstioClient:   istiofake.NewSimpleClientset(),
	}

	// Create a non-default service account
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa",
			Namespace: "default"},
	}
	serviceAccount, err := defaulter.KubeClient.CoreV1().ServiceAccounts("default").Create(context.TODO(), sa, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating service account")

	// Create an applicationConfiguration resource
	u := newUnstructured("core.oam.dev/v1alpha2", "ApplicationConfiguration", "test-appconfig")
	resource := schema.GroupVersionResource{
		Group:    "core.oam.dev",
		Version:  "v1alpha2",
		Resource: "applicationconfigurations",
	}
	_, err = defaulter.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating application config")

	// Create a pod referencing the service account we created
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod1",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "test-appconfig",
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount.Name,
		},
	}
	pod, err := defaulter.KubeClient.CoreV1().Pods("default").Create(context.TODO(), p, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	decoder := decoder()
	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	req.Namespace = "default"
	marshaledPod, err := json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.NotEmpty(t, res.Patches)

	// Get the authorization policy resource we created and do some validations
	authPolicy, err := defaulter.IstioClient.SecurityV1beta1().AuthorizationPolicies("default").Get(context.TODO(), "test-appconfig", metav1.GetOptions{})
	assert.NoError(t, err, "Unexpected error getting authorization policy")
	assert.Equal(t, authPolicy.Name, "test-appconfig")
	assert.Equal(t, authPolicy.Namespace, "default")
	assert.Contains(t, authPolicy.Labels, IstioAppLabel)
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Name, "test-appconfig")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Kind, "ApplicationConfiguration")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].APIVersion, "core.oam.dev/v1alpha2")
	assert.Contains(t, authPolicy.Spec.Selector.MatchLabels, IstioAppLabel)
	assert.Equal(t, len(authPolicy.Spec.GetRules()[0].From[0].Source.Principals), 2)
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account")
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/default/sa/test-sa")

	// Create a pod referencing the second service account we created
	p = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod2",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "test-appconfig",
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount.Name,
		},
	}
	pod, err = defaulter.KubeClient.CoreV1().Pods("default").Create(context.TODO(), p, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req = admission.Request{}
	req.Namespace = "default"
	marshaledPod, err = json.Marshal(pod)
	assert.NoError(t, err, "Unexpected error marshaling pod")
	req.Object = runtime.RawExtension{Raw: marshaledPod}
	res = defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.NotEmpty(t, res.Patches)

	// Get the authorization policy resource we created and do some validations.
	authPolicy, err = defaulter.IstioClient.SecurityV1beta1().AuthorizationPolicies("default").Get(context.TODO(), "test-appconfig", metav1.GetOptions{})
	assert.NoError(t, err, "Unexpected error getting authorization policy")
	assert.Equal(t, authPolicy.Name, "test-appconfig")
	assert.Equal(t, authPolicy.Namespace, "default")
	assert.Contains(t, authPolicy.Labels, IstioAppLabel)
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Name, "test-appconfig")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].Kind, "ApplicationConfiguration")
	assert.Equal(t, authPolicy.GetOwnerReferences()[0].APIVersion, "core.oam.dev/v1alpha2")
	assert.Contains(t, authPolicy.Spec.Selector.MatchLabels, IstioAppLabel)
	assert.Equal(t, len(authPolicy.Spec.GetRules()[0].From[0].Source.Principals), 2)
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account")
	assert.Contains(t, authPolicy.Spec.GetRules()[0].From[0].Source.Principals, "cluster.local/ns/default/sa/test-sa")
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
