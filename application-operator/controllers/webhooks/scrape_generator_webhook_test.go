// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
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
	"k8s.io/apimachinery/pkg/types"
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
		Client:     cli,
		Decoder:    decoder,
		KubeClient: fake.NewSimpleClientset(),
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
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", nil)
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testPod))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Pod", testPod)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed, "Expected validation to succeed.")
}

// TestHandleDeployment tests the handling of a Deployment resource
// GIVEN a call validate Deployment on create or update
// WHEN the Deployment is properly formed
// THEN the validation should succeed
func TestHandleDeployment(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", nil)
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed, "Expected validation to succeed.")
}

// TestHandleReplicaSet tests the handling of a ReplicaSet resource
// GIVEN a call validate ReplicaSet on create or update
// WHEN the ReplicaSet is properly formed
// THEN the validation should succeed
func TestHandleReplicaSet(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", nil)
	testReplicaSet := appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testReplicaSet",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testReplicaSet))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "ReplicaSet", testReplicaSet)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed, "Expected validation to succeed.")
}

// TestHandleStatefulSet tests the handling of a StatefulSet resource
// GIVEN a call validate StatefulSet on create or update
// WHEN the StatefulSet is properly formed
// THEN the validation should succeed
func TestHandleStatefulSet(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", nil)
	testStatefulSet := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testStatefulSet",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testStatefulSet))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "StatefulSet", testStatefulSet)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed, "Expected validation to succeed.")
}

// TestHandleOwnerRefs tests the handling of a workload resource with owner references
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has owner references
// THEN the Handle function should succeed and the metricsBinding is not created
func TestHandleOwnerRefs(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", nil)
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Nil(t, res.Patches, "expected no changes to workload resource")

	// validate that metrics binding was not created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.EqualError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding), "metricsbindings.app.verrazzano.io \"testDeployment-deployment\" not found")
}

// TestHandleNoNamespaceLabel tests the handling of a workload resource whose namespace is not labeled with
// 	"verrazzano-managed": "true"
// GIVEN a call to the webhook Handle function
// WHEN the workload resource namespace is not labeled with "verrazzano-managed": "true"
// THEN the Handle function should succeed and the metricsBinding is not created
func TestHandleNoNamespaceLabel(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", nil)
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Nil(t, res.Patches, "expected no changes to workload resource")

	// validate that metrics binding was not created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.EqualError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding), "metricsbindings.app.verrazzano.io \"testDeployment-deployment\" not found")
}

// TestHandleNamespaceLabelFalse tests the handling of a workload resource whose namespace is labeled with
// 	"verrazzano-managed": "false"
// GIVEN a call to the webhook Handle function
// WHEN the workload resource namespace is labeled with "verrazzano-managed": "false"
// THEN the Handle function should succeed and the metricsBinding is not created
func TestHandleNamespaceLabelFalse(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "false"})
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "test",
		},
	}
	assert.NoError(t, v.Client.Create(context.TODO(), &testDeployment))

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Nil(t, res.Patches, "expected no changes to workload resource")

	// validate that metrics binding was not created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.EqualError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding), "metricsbindings.app.verrazzano.io \"testDeployment-deployment\" not found")
}

// TestHandleMetricsNone tests the handling of a workload resource with "app.verrazzano.io/metrics": "none"
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has  "app.verrazzano.io/metrics": "none"
// THEN the Handle function should succeed and the metricsBinding is not created
func TestHandleMetricsNone(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Nil(t, res.Patches, "expected no changes to workload resource")

	// validate that metrics binding was not created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.EqualError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding), "metricsbindings.app.verrazzano.io \"testDeployment-deployment\" not found")
}

// TestHandleInvalidMetricsTemplate tests the handling of a workload resource with references a metrics template
//  that does not exist
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has  "app.verrazzano.io/metrics": "badTemplate"
// THEN the Handle function should generate an error
func TestHandleInvalidMetricsTemplate(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
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
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Empty(t, res.Patches)

	// validate that metrics binding was created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.NoError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding))
	assert.Len(t, metricsBinding.OwnerReferences, 1)
	assert.Equal(t, "apps/v1", metricsBinding.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Deployment", metricsBinding.OwnerReferences[0].Kind)
	assert.Equal(t, "testDeployment", metricsBinding.OwnerReferences[0].Name)
	assert.Equal(t, "11", string(metricsBinding.OwnerReferences[0].UID))
	assert.True(t, *metricsBinding.OwnerReferences[0].BlockOwnerDeletion)
	assert.True(t, *metricsBinding.OwnerReferences[0].Controller)
	assert.Equal(t, "test", metricsBinding.Spec.MetricsTemplate.Namespace)
	assert.Equal(t, "testTemplateWorkloadNamespace", metricsBinding.Spec.MetricsTemplate.Name)
}

// TestHandleMetricsTemplateSystemNamespace tests the handling of a workload resource which references a metrics
//  template found in the verrazzano-system namespace
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has a valid metrics template reference
// THEN the Handle function should succeed and the metricsBinding is created
func TestHandleMetricsTemplateSystemNamespace(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
	v.createNamespace(t, "verrazzano-system", nil)
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Empty(t, res.Patches)

	// validate that metrics binding was created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.NoError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding))
	assert.Len(t, metricsBinding.OwnerReferences, 1)
	assert.Equal(t, "apps/v1", metricsBinding.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Deployment", metricsBinding.OwnerReferences[0].Kind)
	assert.Equal(t, "testDeployment", metricsBinding.OwnerReferences[0].Name)
	assert.Equal(t, "11", string(metricsBinding.OwnerReferences[0].UID))
	assert.True(t, *metricsBinding.OwnerReferences[0].BlockOwnerDeletion)
	assert.True(t, *metricsBinding.OwnerReferences[0].Controller)
	assert.Equal(t, "verrazzano-system", metricsBinding.Spec.MetricsTemplate.Namespace)
	assert.Equal(t, "testTemplateSameNamespace", metricsBinding.Spec.MetricsTemplate.Name)
}

// TestHandleMetricsTemplateConfigMapNotFound tests the handling of a workload resource which references a metrics
//  template found with Prometheus config map that does not exist
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has an invalid Prometheus config map reference
// THEN the Handle function should fail and return an error
func TestHandleMetricsTemplateConfigMapNotFound(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
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
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
	v.createNamespace(t, "verrazzano-system", nil)
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)

	// validate that metrics binding was created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.NoError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding))
	assert.Len(t, metricsBinding.OwnerReferences, 1)
	assert.Equal(t, "apps/v1", metricsBinding.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Deployment", metricsBinding.OwnerReferences[0].Kind)
	assert.Equal(t, "testDeployment", metricsBinding.OwnerReferences[0].Name)
	assert.Equal(t, "11", string(metricsBinding.OwnerReferences[0].UID))
	assert.True(t, *metricsBinding.OwnerReferences[0].BlockOwnerDeletion)
	assert.True(t, *metricsBinding.OwnerReferences[0].Controller)
	assert.Equal(t, "test", metricsBinding.Spec.MetricsTemplate.Namespace)
	assert.Equal(t, "testTemplateWorkloadNamespace", metricsBinding.Spec.MetricsTemplate.Name)
}

// TestHandleMatchSystemNamespace tests the handling of a workload resource with no metrics template specified
//  but matches a template found in the verrazzano-system namespace
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has no metrics template reference
// THEN the Handle function should succeed and the metricsBinding is created
func TestHandleMatchSystemNamespace(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
	v.createNamespace(t, "verrazzano-system", nil)
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)

	// validate that metrics binding was created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.NoError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding))
	assert.Len(t, metricsBinding.OwnerReferences, 1)
	assert.Equal(t, "apps/v1", metricsBinding.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Deployment", metricsBinding.OwnerReferences[0].Kind)
	assert.Equal(t, "testDeployment", metricsBinding.OwnerReferences[0].Name)
	assert.Equal(t, "11", string(metricsBinding.OwnerReferences[0].UID))
	assert.True(t, *metricsBinding.OwnerReferences[0].BlockOwnerDeletion)
	assert.True(t, *metricsBinding.OwnerReferences[0].Controller)
	assert.Equal(t, "verrazzano-system", metricsBinding.Spec.MetricsTemplate.Namespace)
	assert.Equal(t, "testTemplateSystemNamespace", metricsBinding.Spec.MetricsTemplate.Name)
}

// TestHandleMatchNotFound tests the handling of a workload resource with no metrics template specified
//  and a matching template not found
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has no metrics template reference
// THEN the Handle function should succeed and no metricsBinding is created
func TestHandleMatchNotFound(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
	v.createNamespace(t, "verrazzano-system", nil)
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Empty(t, res.Patches)

	// validate that metrics binding was not created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.EqualError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding), "metricsbindings.app.verrazzano.io \"testDeployment-deployment\" not found")
}

// TestHandleWorkloadUIDNotFound tests the handling of a workload resource with no UID
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has no UID defined
// THEN the Handle function should succeed and no metrics binding is created
func TestHandleWorkloadUIDNotFound(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
	v.createNamespace(t, "verrazzano-system", nil)
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)

	// validate that metrics binding was not created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.EqualError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding), "metricsbindings.app.verrazzano.io \"testDeployment-deployment\" not found")
}

// TestHandleMatchTemplateNoWorkloadSelector tests the handling of a workload resource with no metrics template specified
//  and a metrics template that doesn't have a workload selector specified
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has no metrics template reference
// THEN the Handle function should succeed and no metricsBinding is created
func TestHandleMatchTemplateNoWorkloadSelector(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
	v.createNamespace(t, "verrazzano-system", nil)
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Empty(t, res.Patches)

	// validate that metrics binding was not created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.EqualError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding), "metricsbindings.app.verrazzano.io \"testDeployment-deployment\" not found")
}

// TestHandleNoConfigMap tests the handling of a workload resource that doesn't have a Prometheus target
//  config map specified in the metrics template
// GIVEN a call to the webhook Handle function
// WHEN the workload resource has a metrics template reference
// THEN the Handle function should succeed and no metricsBinding is created
func TestHandleNoConfigMap(t *testing.T) {
	v := newScrapeGeneratorWebhook()

	// Test data
	v.createNamespace(t, "test", map[string]string{"verrazzano-managed": "true"})
	v.createNamespace(t, "verrazzano-system", nil)
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

	req := newScrapeGeneratorRequest(admissionv1beta1.Create, "Deployment", testDeployment)
	res := v.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Empty(t, res.Patches)

	// validate that metrics binding was not created as expected
	namespacedName := types.NamespacedName{Namespace: "test", Name: "testDeployment-deployment"}
	metricsBinding := &vzapp.MetricsBinding{}
	assert.EqualError(t, v.Client.Get(context.TODO(), namespacedName, metricsBinding), "metricsbindings.app.verrazzano.io \"testDeployment-deployment\" not found")
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
		},
	}
	_, err := v.KubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), cm, metav1.CreateOptions{})
	assert.NoError(t, err, "unexpected error creating namespace")
}
