// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
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
	vzapp.AddToScheme(scheme)
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

// TestHandleOwnerRefs tests the handling of a workload resource with owner references
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has owner references
// THEN the Handle function should succeed and the workload resource not mutated
func TestHandleOwnerRefs(t *testing.T) {
	assert := assert.New(t)
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", nil)
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "foo",
				},
			},
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testDeployment))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(res.Allowed)
	assert.Nil(res.Patches, "expected no changes to workload resource")
}

// TestHandleNoNamespaceLabel tests the handling of a workload resource whose namespace is not labeled with
// 	"verrazzano-managed": "true"
// GIVEN a call to the webhook Handle function
// WHEN the workload resource namespace is not labeled with "verrazzano-managed": "true"
// THEN the Handle function should succeed and the workload resource not mutated
func TestHandleNoNamespaceLabel(t *testing.T) {
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
	assert.True(res.Allowed)
	assert.Nil(res.Patches, "expected no changes to workload resource")
}

// TestHandleNamespaceLabelFalse tests the handling of a workload resource whose namespace is labeled with
// 	"verrazzano-managed": "false"
// GIVEN a call to the webhook Handle function
// WHEN the workload resource namespace is labeled with "verrazzano-managed": "false"
// THEN the Handle function should succeed and the workload resource not mutated
func TestHandleNamespaceLabelFalse(t *testing.T) {
	assert := assert.New(t)
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "false"})
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testDeployment))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(res.Allowed)
	assert.Nil(res.Patches, "expected no changes to workload resource")
}

// TestHandleMetricsNone tests the handling of a workload resource with "app.verrazzano.io/metrics": "none"
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has  "app.verrazzano.io/metrics": "none"
// THEN the Handle function should succeed and the workload resource not mutated
func TestHandleMetricsNone(t *testing.T) {
	assert := assert.New(t)
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Annotations: map[string]string{
				"app.verrazzano.io/metrics": "none",
			},
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testDeployment))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(res.Allowed)
	assert.Nil(res.Patches, "expected no changes to workload resource")
}

// TestHandleInvalidMetricsTemplate tests the handling of a workload resource with references a metrics template
//  that does not exist
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has  "app.verrazzano.io/metrics": "badTemplate"
// THEN the Handle function should generate an error
func TestHandleInvalidMetricsTemplate(t *testing.T) {
	assert := assert.New(t)
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Annotations: map[string]string{
				"app.verrazzano.io/metrics": "badTemplate",
			},
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testDeployment))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.False(res.Allowed)
	assert.Equal("metricstemplates.app.verrazzano.io \"badTemplate\" not found", res.Result.Message)
}

// TestHandleMetricsTemplateWorkloadNamespace tests the handling of a workload resource which references a metrics
//  template found in the namespace of the workload resource
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has a valid metrics template reference
// THEN the Handle function should succeed and the workload resource is patched
func TestHandleMetricsTemplateWorkloadNamespace(t *testing.T) {
	assert := assert.New(t)
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
	v.createConfigMap(t, "test", "testPromConfigMap")
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			UID:       "11",
			Annotations: map[string]string{
				"app.verrazzano.io/metrics": "testTemplateSameNamespace",
			},
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testDeployment))
	testTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "testTemplateSameNamespace",
			UID:       "22",
		},
		Spec: vzapp.MetricsTemplateSpec{
			WorkloadSelector: vzapp.WorkloadSelector{},
			PrometheusConfig: vzapp.PrometheusConfig{
				TargetConfigMap: vzapp.TargetConfigMap{
					Namespace: "test",
					Name:      "testPromConfigMap",
				},
			},
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testTemplate))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(res.Allowed)
	assert.NotNil(res.Patches)
	assert.Len(res.Patches, 3)
	assert.Equal("add", res.Patches[0].Operation)
	assert.Equal("/metadata/annotations/app.verrazzano.io~1metrics-prometheus-configmap-uid", res.Patches[0].Path)
	assert.Equal("33", res.Patches[0].Value)
	assert.Equal("add", res.Patches[1].Operation)
	assert.Equal("/metadata/annotations/app.verrazzano.io~1metrics-template-uid", res.Patches[1].Path)
	assert.Equal("22", res.Patches[1].Value)
	assert.Equal("add", res.Patches[2].Operation)
	assert.Equal("/metadata/annotations/app.verrazzano.io~1metrics-workload-uid", res.Patches[2].Path)
	assert.Equal("11", res.Patches[2].Value)
}

// TestHandleMetricsTemplateSystemNamespace tests the handling of a workload resource which references a metrics
//  template found in the verrazzano-system namespace
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has a valid metrics template reference
// THEN the Handle function should succeed and the workload resource is patched
func TestHandleMetricsTemplateSystemNamespace(t *testing.T) {
	assert := assert.New(t)
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
	v.createNamespace(t, "verrazzano-system", nil)
	v.createConfigMap(t, "test", "testPromConfigMap")
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			UID:       "11",
			Annotations: map[string]string{
				"app.verrazzano.io/metrics": "testTemplateSameNamespace",
			},
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testDeployment))
	testTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "verrazzano-system",
			Name:      "testTemplateSameNamespace",
			UID:       "22",
		},
		Spec: vzapp.MetricsTemplateSpec{
			WorkloadSelector: vzapp.WorkloadSelector{},
			PrometheusConfig: vzapp.PrometheusConfig{
				TargetConfigMap: vzapp.TargetConfigMap{
					Namespace: "test",
					Name:      "testPromConfigMap",
				},
			},
		},
	}
	assert.NoError(v.Client.Create(context.TODO(), &testTemplate))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(res.Allowed)
	assert.NotNil(res.Patches)
	assert.Len(res.Patches, 3)
	assert.Equal("add", res.Patches[0].Operation)
	assert.Equal("/metadata/annotations/app.verrazzano.io~1metrics-prometheus-configmap-uid", res.Patches[0].Path)
	assert.Equal("33", res.Patches[0].Value)
	assert.Equal("add", res.Patches[1].Operation)
	assert.Equal("/metadata/annotations/app.verrazzano.io~1metrics-template-uid", res.Patches[1].Path)
	assert.Equal("22", res.Patches[1].Value)
	assert.Equal("add", res.Patches[2].Operation)
	assert.Equal("/metadata/annotations/app.verrazzano.io~1metrics-workload-uid", res.Patches[2].Path)
	assert.Equal("11", res.Patches[2].Value)
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

func (v *ScrapeGeneratorWebhook) createConfigMap(t *testing.T, namespace string, name string) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			UID:       "33",
		},
	}
	_, err := v.KubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), cm, metav1.CreateOptions{})
	assert.NoError(t, err, "unexpected error creating namespace")
}
