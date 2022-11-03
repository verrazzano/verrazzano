// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workloadselector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

// TestMatch tests DoesWorkloadMatch
// GIVEN a namespace label selector, object selector, and specific GVK values
// WHEN DoesWorkloadMatch is called
// THEN a match of true is returned
func TestMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	labels := map[string]string{
		"test-label": "true",
	}

	// Create the namespace
	ws.createNamespace(t, "test-ns", labels)

	// Create a deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", labels)

	// Namespace selector with match labels
	namespaceSelector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	// Object selector with match expressions
	objectSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "test-label",
				Operator: metav1.LabelSelectorOpExists,
			},
		},
	}

	// workload resource matches
	found, err := ws.DoesWorkloadMatch(deploy, namespaceSelector, objectSelector, []string{"apps"}, []string{"v1"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

// TestMatchDefaults tests DoesWorkloadMatch
// GIVEN no namespace label selector, no object selector, and no GVK values
// WHEN DoesWorkloadMatch is called
// THEN a match of true is returned
func TestMatchDefaults(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	labels := map[string]string{
		"test-label": "true",
	}

	// Create the namespace
	ws.createNamespace(t, "test-ns", labels)

	// Create a deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", labels)

	// workload resource matches
	found, err := ws.DoesWorkloadMatch(deploy, nil, nil, nil, nil, nil)
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

// TestNoMatchNamespace tests DoesWorkloadMatch
// GIVEN a namespace label selector, object selector, and specific GVK values
// WHEN DoesWorkloadMatch is called
// THEN a match of false is returned because namespace did not match
func TestNoMatchNamespace(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	labels := map[string]string{
		"test-label": "true",
	}

	// Create the namespace
	ws.createNamespace(t, "test-ns", nil)

	// Create a deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", labels)

	// Namespace selector with match labels
	namespaceSelector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	// Object selector with match expressions
	objectSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "test-label",
				Operator: metav1.LabelSelectorOpExists,
			},
		},
	}

	// workload resource does not match
	found, err := ws.DoesWorkloadMatch(deploy, namespaceSelector, objectSelector, []string{"apps"}, []string{"v1"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

// TestEmptyNamespaceSelectorMatch tests doesNamespaceMatch
// GIVEN an empty namespace label selector
// WHEN doesNamespaceMatch is called
// THEN a match of true is returned
func TestEmptyNamespaceSelectorMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create a couple of namespaces
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// Create a deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Empty namespace selector
	namespaceSelector := &metav1.LabelSelector{}

	// Namespace match found
	found, err := ws.doesNamespaceMatch(deploy, namespaceSelector)
	assert.NoError(t, err, "unexpected error getting namespaces")
	assert.True(t, found, "expected to find match")
}

// TestNilNamespaceSelectorMatch tests doesNamespaceMatch
// GIVEN a nil namespace label selector
// WHEN doesNamespaceMatch is called
// THEN a match of true is returned
func TestNilNamespaceSelectorMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create a couple of namespaces
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// Create a deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Namespace match found - nil namespace selector specified
	found, err := ws.doesNamespaceMatch(deploy, nil)
	assert.NoError(t, err, "unexpected error getting namespaces")
	assert.True(t, found, "expected to find match")
}

// TestMatchLabelsNamespaceSelectorMatch tests doesNamespaceMatch
// GIVEN a namespace label selector using a MatchLabel
// WHEN doesNamespaceMatch is called
// THEN a match of true is returned
func TestMatchLabelsNamespaceSelectorMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	namespaceLabels := map[string]string{
		"test-label": "true",
	}

	// Create a couple of namespaces
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", namespaceLabels)

	// Create a deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Namespace selector with match labels
	namespaceSelector := &metav1.LabelSelector{
		MatchLabels: namespaceLabels,
	}

	// Namespace match found
	found, err := ws.doesNamespaceMatch(deploy, namespaceSelector)
	assert.NoError(t, err, "unexpected error getting namespaces")
	assert.True(t, found, "expected to find match")
}

// TestMatchLabelsNamespaceSelectorNoMatch tests doesNamespaceMatch
// GIVEN a namespace label selector using a MatchLabel
// WHEN doesNamespaceMatch is called
// THEN a match of false is returned
func TestMatchLabelsNamespaceSelectorNoMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	namespaceLabels := map[string]string{
		"test-label": "true",
	}

	// Create a couple of namespaces
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// Create a deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Namespace selector with match labels
	namespaceSelector := &metav1.LabelSelector{
		MatchLabels: namespaceLabels,
	}

	// Namespace match not found
	found, err := ws.doesNamespaceMatch(deploy, namespaceSelector)
	assert.NoError(t, err, "unexpected error getting namespaces")
	assert.False(t, found, "expected to not find match")
}

// TestMatchExpressionsNamespaceSelector tests doesNamespaceMatch
// GIVEN a namespace label selector using a MatchExpression
// WHEN doesNamespaceMatch is called
// THEN a match of true is returned
func TestMatchExpressionsNamespaceSelector(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	namespaceLabels := map[string]string{
		"test-label": "true",
	}

	// Create a couple of namespaces
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", namespaceLabels)

	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Namespace selector with match expressions
	namespaceSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "test-label",
				Operator: metav1.LabelSelectorOpExists,
			},
		},
	}

	// Namespace selector with match labels
	found, err := ws.doesNamespaceMatch(deploy, namespaceSelector)
	assert.NoError(t, err, "unexpected error getting namespaces")
	assert.True(t, found, "expected to find match")
}

// TestMatchExpressionsNamespaceSelectorNoMatch tests doesNamespaceMatch
// GIVEN a namespace label selector using a MatchExpression
// WHEN doesNamespaceMatch is called
// THEN a match of false is returned
func TestMatchExpressionsNamespaceSelectorNoMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create a couple of namespaces
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Namespace selector with match expressions
	namespaceSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "test-label",
				Operator: metav1.LabelSelectorOpExists,
			},
		},
	}

	// Namespace match not found
	found, err := ws.doesNamespaceMatch(deploy, namespaceSelector)
	assert.NoError(t, err, "unexpected error getting namespaces")
	assert.False(t, found, "expected to not find match")
}

// TestMatchExactGVK tests doesObjectMatch
// GIVEN specific GVK values and no object label selector
// WHEN doesObjectMatch is called
// THEN a match of true is returned
func TestMatchExactGVK(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource matches
	found, err := ws.doesObjectMatch(deploy, objectSelector, []string{"apps"}, []string{"v1"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

// TestMatchWildcardVersion tests doesObjectMatch
// GIVEN GVK values with a wildcard version and no object label selector
// WHEN doesObjectMatch is called
// THEN a match of true is returned
func TestMatchWildcardVersion(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource matches
	found, err := ws.doesObjectMatch(deploy, objectSelector, []string{"apps"}, []string{"*"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

// TestMatchWildcardGroup tests doesObjectMatch
// GIVEN GVK values with a wildcard group and no object label selector
// WHEN doesObjectMatch is called
// THEN a match of true is returned
func TestMatchWildcardGroup(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource matches
	found, err := ws.doesObjectMatch(deploy, objectSelector, []string{"*"}, []string{"v1"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

// TestMatchWildcardKind tests doesObjectMatch
// GIVEN GVK values with a wildcard Kind and no object label selector
// WHEN doesObjectMatch is called
// THEN a match of true is returned
func TestMatchWildcardKind(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource matches
	found, err := ws.doesObjectMatch(deploy, objectSelector, []string{"apps"}, []string{"v1"}, []string{"*"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

// TestNoMatchExactGVK tests doesObjectMatch
// GIVEN specific GVK values and no object label selector
// WHEN doesObjectMatch is called
// THEN a match of false is returned
func TestNoMatchExactGVK(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// One namespace, not including the namespace of the deployment
	ws.createNamespace(t, "default", nil)

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource does not match
	found, err := ws.doesObjectMatch(deploy, objectSelector, []string{"apps"}, []string{"v2"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

// TestNoMatchWildcardVersion tests doesObjectMatch
// GIVEN GVK values with wildcard version and no object label selector
// WHEN doesObjectMatch is called
// THEN a match of false is returned
func TestNoMatchWildcardVersion(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource does not match
	found, err := ws.doesObjectMatch(deploy, objectSelector, []string{"foo"}, []string{"*"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

// TestNoMatchWildcardGroup tests doesObjectMatch
// GIVEN GVK values with wildcard group version and no object label selector
// WHEN doesObjectMatch is called
// THEN a match of false is returned
func TestNoMatchWildcardGroup(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource does not match
	found, err := ws.doesObjectMatch(deploy, objectSelector, []string{"*"}, []string{"foo"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

// TestNoMatchWildcardKind tests doesObjectMatch
// GIVEN GVK values with wildcard kind version and no object label selector
// WHEN doesObjectMatch is called
// THEN a match of false is returned
func TestNoMatchWildcardKind(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource does not match
	found, err := ws.doesObjectMatch(deploy, objectSelector, []string{"apps"}, []string{"foo"}, []string{"*"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

// TestMatchLabelsObjectSelectorMatch tests doesObjectMatch
// GIVEN an object label selector using a MatchLabel
// WHEN doesObjectMatch is called
// THEN a match of true is returned
func TestMatchLabelsObjectSelectorMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	labels := map[string]string{
		"test-label": "true",
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", labels)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// Object selector with match labels
	objectSelector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	// workload resource match
	found, err := ws.doesObjectMatch(deploy, objectSelector, nil, nil, nil)
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

// TestMatchExpressionsObjectSelectorMatch tests doesObjectMatch
// GIVEN an object label selector using a MatchExpression
// WHEN doesObjectMatch is called
// THEN a match of true is returned
func TestMatchExpressionsObjectSelectorMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	labels := map[string]string{
		"test-label": "true",
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", labels)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// Object selector with match expressions
	objectSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "test-label",
				Operator: metav1.LabelSelectorOpExists,
			},
		},
	}

	// workload resource match
	found, err := ws.doesObjectMatch(deploy, objectSelector, []string{"*"}, []string{"*"}, []string{"*"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

// TestMatchLabelsObjectSelectorNoMatch tests doesObjectMatch
// GIVEN an object label selector using a MatchLabel
// WHEN doesObjectMatch is called
// THEN a match of false is returned
func TestMatchLabelsObjectSelectorNoMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	labels := map[string]string{
		"test-label": "true",
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// Object selector with match labels
	objectSelector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	// workload resource match
	found, err := ws.doesObjectMatch(deploy, objectSelector, nil, nil, nil)
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

// TestMatchExpressionsObjectSelectorNoMatch tests doesObjectMatch
// GIVEN an object label selector using a MatchExpression
// WHEN doesObjectMatch is called
// THEN a match of false is returned
func TestMatchExpressionsObjectSelectorNoMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// Object selector with match expressions
	objectSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "test-label",
				Operator: metav1.LabelSelectorOpExists,
			},
		},
	}

	// workload resource did not match
	found, err := ws.doesObjectMatch(deploy, objectSelector, []string{"*"}, []string{"*"}, []string{"*"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

func (w *WorkloadSelector) createNamespace(t *testing.T, name string, labels map[string]string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	_, err := w.KubeClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	assert.NoError(t, err, "unexpected error creating namespace")
}

func (w *WorkloadSelector) createDeployment(t *testing.T, namespace string, name string, labels map[string]string) *unstructured.Unstructured {
	u := newUnstructured("apps/v1", "Deployment", namespace, name)
	u.SetLabels(labels)
	resource := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	uout, err := dynamicClient.Resource(resource).Namespace(namespace).Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "unexpected error creating deployment")

	return uout
}

func newUnstructured(apiVersion string, kind string, namespace string, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
		},
	}
}
