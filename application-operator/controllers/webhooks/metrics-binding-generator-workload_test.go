// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// newGeneratorWorkloadWebhook creates a new GeneratorWorkloadWebhook
func newGeneratorWorkloadWebhook() GeneratorWorkloadWebhook {
	scheme := newScheme()
	scheme.AddKnownTypes(schema.GroupVersion{
		Version: "v1",
	}, &corev1.Pod{}, &appsv1.Deployment{}, &appsv1.ReplicaSet{}, &appsv1.StatefulSet{}, &corev1.Namespace{})
	vzapp.AddToScheme(scheme)
	decoder, _ := admission.NewDecoder(scheme)
	cli := ctrlfake.NewFakeClientWithScheme(scheme)
	v := GeneratorWorkloadWebhook{
		Client:     cli,
		Decoder:    decoder,
		KubeClient: fake.NewSimpleClientset(),
	}
	return v
}

// newGeneratorWorkloadRequest creates a new admissionRequest with the provided operation and object.
func newGeneratorWorkloadRequest(op admissionv1beta1.Operation, kind string, obj interface{}) admission.Request {
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
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", nil, nil)
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testPod))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Pod", testPod)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed, "Expected validation to succeed.")
}

// TestHandleDeployment tests the handling of a Deployment resource
// GIVEN a call validate Deployment on create or update
// WHEN the Deployment is properly formed
// THEN the validation should succeed
func TestHandleDeployment(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", nil, nil)
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed, "Expected validation to succeed.")
}

// TestHandleReplicaSet tests the handling of a ReplicaSet resource
// GIVEN a call validate ReplicaSet on create or update
// WHEN the ReplicaSet is properly formed
// THEN the validation should succeed
func TestHandleReplicaSet(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", nil, nil)
	testReplicaSet := appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testReplicaSet",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testReplicaSet))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "ReplicaSet", testReplicaSet)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed, "Expected validation to succeed.")
}

// TestHandleStatefulSet tests the handling of a StatefulSet resource
// GIVEN a call validate StatefulSet on create or update
// WHEN the StatefulSet is properly formed
// THEN the validation should succeed
func TestHandleStatefulSet(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", nil, nil)
	testStatefulSet := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testStatefulSet",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testStatefulSet))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "StatefulSet", testStatefulSet)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed, "Expected validation to succeed.")
}

// TestHandleOwnerRefs tests the handling of a workload resource with owner references
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has owner references
// THEN the Handle function should succeed and the metricsBinding is not created
func TestHandleOwnerRefs(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", nil, nil)
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "foo",
				},
			},
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Nil(t, res.Patches, "expected no changes to workload resource")

	// validate that metrics binding was not created as expected
	v.validateNoMetricsBinding(t)
}

// TestHandleMetricsNone tests the handling of a workload resource with "app.verrazzano.io/metrics": "none"
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has  "app.verrazzano.io/metrics": "none"
// THEN the Handle function should succeed and the metricsBinding is not created
func TestHandleMetricsNone(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, nil)
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
			Annotations: map[string]string{
				"app.verrazzano.io/metrics": "none",
			},
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Nil(t, res.Patches, "expected no changes to workload resource")

	// validate that metrics binding was not created as expected
	v.validateNoMetricsBinding(t)
}

// TestHandleMetricsNoneNamespace tests the handling of a namespace with "app.verrazzano.io/metrics": "none"
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has  "app.verrazzano.io/metrics": "none"
// THEN the Handle function should succeed and the metricsBinding is not created
func TestHandleMetricsNoneNamespace(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, map[string]string{"app.verrazzano.io/metrics": "none"})
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Nil(t, res.Patches, "expected no changes to workload resource")

	// validate that metrics binding was not created as expected
	v.validateNoMetricsBinding(t)
}

// TestHandleInvalidMetricsTemplate tests the handling of a workload resource with references a metrics template
//  that does not exist
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has  "app.verrazzano.io/metrics": "badTemplate"
// THEN the Handle function should generate an error
func TestHandleInvalidMetricsTemplate(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, nil)
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
			Annotations: map[string]string{
				"app.verrazzano.io/metrics": "badTemplate",
			},
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, "metricstemplates.app.verrazzano.io \"badTemplate\" not found", res.Result.Message)
}

// TestHandleMetricsTemplateWorkloadNamespace tests the handling of a workload resource which references a metrics
//  template found in the namespace of the workload resource
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has a valid metrics template reference
// THEN the Handle function should succeed and the metricsBinding is created
func TestHandleMetricsTemplateWorkloadNamespace(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, nil)
	v.createConfigMap(t, "test", "testPromConfigMap")
	testDeployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
			UID:       "11",
			Annotations: map[string]string{
				"app.verrazzano.io/metrics": "testTemplateWorkloadNamespace",
			},
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))
	testTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "testTemplateWorkloadNamespace",
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
	assert.NoError(t, v.Client.Create(context.TODO(), &testTemplate))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Len(t, res.Patches, 1)
	assert.Equal(t, "add", res.Patches[0].Operation)
	assert.Equal(t, "/metadata/labels", res.Patches[0].Path)
	assert.Contains(t, res.Patches[0].Value, constants.MetricsWorkloadLabel)

	// validate that metrics binding was created as expected
	v.validateMetricsBinding(t, "test", "testTemplateWorkloadNamespace")
}

// TestHandleMetricsTemplateSystemNamespace tests the handling of a workload resource which references a metrics
//  template found in the verrazzano-system namespace
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has a valid metrics template reference
// THEN the Handle function should succeed and the metricsBinding is created
func TestHandleMetricsTemplateSystemNamespace(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, nil)
	v.createNamespace(t, "verrazzano-system", nil, nil)
	v.createConfigMap(t, "test", "testPromConfigMap")
	testDeployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
			UID:       "11",
			Annotations: map[string]string{
				"app.verrazzano.io/metrics": "testTemplateSameNamespace",
			},
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))
	testTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "verrazzano-system",
			Name:      "testTemplateSameNamespace",
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
	assert.NoError(t, v.Client.Create(context.TODO(), &testTemplate))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Len(t, res.Patches, 1)
	assert.Equal(t, "add", res.Patches[0].Operation)
	assert.Equal(t, "/metadata/labels", res.Patches[0].Path)
	assert.Contains(t, res.Patches[0].Value, constants.MetricsWorkloadLabel)

	// validate that metrics binding was created as expected
	v.validateMetricsBinding(t, "verrazzano-system", "testTemplateSameNamespace")
}

// TestHandleMetricsTemplateConfigMapNotFound tests the handling of a workload resource which references a metrics
//  template found with Prometheus config map that does not exist
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has an invalid Prometheus config map reference
// THEN the Handle function should fail and return an error
func TestHandleMetricsTemplateConfigMapNotFound(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, nil)
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			UID:       "11",
			Annotations: map[string]string{
				"app.verrazzano.io/metrics": "testTemplateWorkloadNamespace",
			},
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))
	testTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "testTemplateWorkloadNamespace",
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
	assert.NoError(t, v.Client.Create(context.TODO(), &testTemplate))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, "configmaps \"testPromConfigMap\" not found", res.Result.Message)
}

// TestHandleMatchWorkloadNamespace tests the handling of a workload resource with no metrics template specified
//  but matches a template found in the workload resources namespace
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has no metrics template reference
// THEN the Handle function should succeed and the metricsBinding is created
func TestHandleMatchWorkloadNamespace(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, nil)
	v.createNamespace(t, "verrazzano-system", nil, nil)
	v.createConfigMap(t, "test", "testPromConfigMap")
	testDeployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
			UID:       "11",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))
	testTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "testTemplateWorkloadNamespace",
		},
		Spec: vzapp.MetricsTemplateSpec{
			WorkloadSelector: vzapp.WorkloadSelector{
				APIGroups: []string{
					"apps",
				},
				APIVersions: []string{
					"v1",
				},
				Resources: []string{
					"deployment",
				},
			},
			PrometheusConfig: vzapp.PrometheusConfig{
				TargetConfigMap: vzapp.TargetConfigMap{
					Namespace: "test",
					Name:      "testPromConfigMap",
				},
			},
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testTemplate))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Len(t, res.Patches, 1)
	assert.Equal(t, "add", res.Patches[0].Operation)
	assert.Equal(t, "/metadata/labels", res.Patches[0].Path)
	assert.Contains(t, res.Patches[0].Value, constants.MetricsWorkloadLabel)

	// validate that metrics binding was created as expected
	v.validateMetricsBinding(t, "test", "testTemplateWorkloadNamespace")
}

// TestHandleMatchSystemNamespace tests the handling of a workload resource with no metrics template specified
//  but matches a template found in the verrazzano-system namespace
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has no metrics template reference
// THEN the Handle function should succeed and the metricsBinding is created
func TestHandleMatchSystemNamespace(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, nil)
	v.createNamespace(t, "verrazzano-system", nil, nil)
	v.createConfigMap(t, "test", "testPromConfigMap")
	testDeployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
			UID:       "11",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))
	testTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "verrazzano-system",
			Name:      "testTemplateSystemNamespace",
		},
		Spec: vzapp.MetricsTemplateSpec{
			WorkloadSelector: vzapp.WorkloadSelector{
				APIGroups: []string{
					"apps",
				},
				APIVersions: []string{
					"v1",
				},
				Resources: []string{
					"deployment",
				},
			},
			PrometheusConfig: vzapp.PrometheusConfig{
				TargetConfigMap: vzapp.TargetConfigMap{
					Namespace: "test",
					Name:      "testPromConfigMap",
				},
			},
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testTemplate))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Len(t, res.Patches, 1)
	assert.Equal(t, "add", res.Patches[0].Operation)
	assert.Equal(t, "/metadata/labels", res.Patches[0].Path)
	assert.Contains(t, res.Patches[0].Value, constants.MetricsWorkloadLabel)

	// validate that metrics binding was created as expected
	v.validateMetricsBinding(t, "verrazzano-system", "testTemplateSystemNamespace")
}

// TestHandleMatchNotFound tests the handling of a workload resource with no metrics template specified
//  and a matching template not found
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has no metrics template reference
// THEN the Handle function should succeed and no metricsBinding is created
func TestHandleMatchNotFound(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, nil)
	v.createNamespace(t, "verrazzano-system", nil, nil)
	v.createConfigMap(t, "test", "testPromConfigMap")
	testDeployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
			UID:       "11",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))
	testTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "verrazzano-system",
			Name:      "testTemplateSystemNamespace",
		},
		Spec: vzapp.MetricsTemplateSpec{
			WorkloadSelector: vzapp.WorkloadSelector{
				APIGroups: []string{
					"apps",
				},
				APIVersions: []string{
					"*",
				},
				Resources: []string{
					"foo",
				},
			},
			PrometheusConfig: vzapp.PrometheusConfig{
				TargetConfigMap: vzapp.TargetConfigMap{
					Namespace: "test",
					Name:      "testPromConfigMap",
				},
			},
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testTemplate))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Empty(t, res.Patches)

	// validate that metrics binding was not created as expected
	v.validateNoMetricsBinding(t)
}

// TestHandleMatchTemplateNoWorkloadSelector tests the handling of a workload resource with no metrics template specified
//  and a metrics template that doesn't have a workload selector specified
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has no metrics template reference
// THEN the Handle function should succeed and no metricsBinding is created
func TestHandleMatchTemplateNoWorkloadSelector(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, nil)
	v.createNamespace(t, "verrazzano-system", nil, nil)
	v.createConfigMap(t, "test", "testPromConfigMap")
	testDeployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
			UID:       "11",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))
	testTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "verrazzano-system",
			Name:      "testTemplateSystemNamespace",
		},
		Spec: vzapp.MetricsTemplateSpec{
			PrometheusConfig: vzapp.PrometheusConfig{
				TargetConfigMap: vzapp.TargetConfigMap{
					Namespace: "test",
					Name:      "testPromConfigMap",
				},
			},
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testTemplate))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Empty(t, res.Patches)

	// validate that metrics binding was not created as expected
	v.validateNoMetricsBinding(t)
}

// TestHandleNoConfigMap tests the handling of a workload resource that doesn't have a Prometheus target
//  config map specified in the metrics template
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has a metrics template reference
// THEN the Handle function should succeed and no metricsBinding is created
func TestHandleNoConfigMap(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, nil)
	v.createNamespace(t, "verrazzano-system", nil, nil)
	testDeployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
			Annotations: map[string]string{
				"app.verrazzano.io/metrics": "testTemplateWorkloadNamespace",
			},
			UID: "11",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))
	testTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "testTemplateWorkloadNamespace",
		},
		Spec: vzapp.MetricsTemplateSpec{},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testTemplate))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Empty(t, res.Patches)

	// validate that metrics binding was not created as expected
	v.validateNoMetricsBinding(t)
}

// TestHandleNamespaceAnnotation tests the handling of a namespace that specifies a template
// GIVEN a call to the webhook Handle function
// WHEN the namespace has a metrics template reference
// THEN the Handle function should succeed and a metricsBinding is created
func TestHandleNamespaceAnnotation(t *testing.T) {
	v := newGeneratorWorkloadWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"}, map[string]string{"app.verrazzano.io/metrics": "testTemplateDifferentNamespace"})
	v.createNamespace(t, constants.VerrazzanoSystemNamespace, nil, nil)
	v.createConfigMap(t, "test", "testPromConfigMap")
	testDeployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))
	testTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      "testTemplateDifferentNamespace",
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
	extraTemplate := vzapp.MetricsTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      "testWrongTemplate",
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
	assert.NoError(t, v.Client.Create(context.TODO(), &testTemplate))
	assert.NoError(t, v.Client.Create(context.TODO(), &extraTemplate))

	req := newGeneratorWorkloadRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Len(t, res.Patches, 1)
	assert.Equal(t, "add", res.Patches[0].Operation)
	assert.Equal(t, "/metadata/labels", res.Patches[0].Path)
	assert.Contains(t, res.Patches[0].Value, constants.MetricsWorkloadLabel)

	// validate that metrics binding was created as expected
	v.validateMetricsBinding(t, "verrazzano-system", "testTemplateDifferentNamespace")
}

func (v *GeneratorWorkloadWebhook) createNamespace(t *testing.T, name string, labels map[string]string, annotations map[string]string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
	}
	err := v.Client.Create(context.TODO(), ns)
	assert.NoError(t, err, "unexpected error creating namespace")
	_, err = v.KubeClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	assert.NoError(t, err, "unexpected error creating namespace")
}

func (v *GeneratorWorkloadWebhook) createConfigMap(t *testing.T, namespace string, name string) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	_, err := v.KubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), cm, metav1.CreateOptions{})
	assert.NoError(t, err, "unexpected error creating namespace")
}

func (v *GeneratorWorkloadWebhook) validateNoMetricsBinding(t *testing.T) {
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.EqualError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding), "metricsbindings.app.verrazzano.io \"testDeployment-deployment\" not found")
}

func (v *GeneratorWorkloadWebhook) validateMetricsBinding(t *testing.T, templateNamespace string, templateName string) {
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-apps-v1-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.NoError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding))
	assert.Equal(t, "apps/v1", metricsBinding.Spec.Workload.TypeMeta.APIVersion)
	assert.Equal(t, "Deployment", metricsBinding.Spec.Workload.TypeMeta.Kind)
	assert.Equal(t, "testDeployment", metricsBinding.Spec.Workload.Name)
	assert.Equal(t, templateNamespace, metricsBinding.Spec.MetricsTemplate.Namespace)
	assert.Equal(t, templateName, metricsBinding.Spec.MetricsTemplate.Name)
	assert.Equal(t, "test", metricsBinding.Spec.PrometheusConfigMap.Namespace)
	assert.Equal(t, "testPromConfigMap", metricsBinding.Spec.PrometheusConfigMap.Name)
}
