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

func TestEmptyNamespaceSelectorMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create a couple of namespaces
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// Empty namespace selector
	namespaceSelector := &metav1.LabelSelector{}

	// Namespace match found
	namespaces, err := ws.getMatchingNamespaces(namespaceSelector)
	assert.NoError(t, err, "unexpected error getting namespaces")
	assert.Len(t, namespaces.Items, 2)
	assert.Equal(t, "default", namespaces.Items[0].Name)
	assert.Equal(t, "test-ns", namespaces.Items[1].Name)
}

func TestNilNamespaceSelectorMatch(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create a couple of namespaces
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)

	// Namespace match found - nil namespace selector specified
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespaces")
	assert.Len(t, namespaces.Items, 2)
	assert.Equal(t, "default", namespaces.Items[0].Name)
	assert.Equal(t, "test-ns", namespaces.Items[1].Name)
}

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

	// Namespace selector with match labels
	namespaceSelector := &metav1.LabelSelector{
		MatchLabels: namespaceLabels,
	}

	// Namespace match found
	namespaces, err := ws.getMatchingNamespaces(namespaceSelector)
	assert.NoError(t, err, "unexpected error getting namespaces")
	assert.Len(t, namespaces.Items, 1)
	assert.Equal(t, "test-ns", namespaces.Items[0].Name)
}

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

	// Namespace selector with match labels
	namespaceSelector := &metav1.LabelSelector{
		MatchLabels: namespaceLabels,
	}

	// Namespace match not found
	namespaces, err := ws.getMatchingNamespaces(namespaceSelector)
	assert.NoError(t, err, "unexpected error getting namespaces")
	assert.Len(t, namespaces.Items, 0)
}

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

	// Namespace selector with match labels
	namespaceSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "test-label",
				Operator: metav1.LabelSelectorOpExists,
			},
		},
	}

	// Namespace selector with match labels
	namespaces, err := ws.getMatchingNamespaces(namespaceSelector)
	assert.NoError(t, err, "unexpected error getting namespaces")
	assert.Len(t, namespaces.Items, 1)
	assert.Equal(t, "test-ns", namespaces.Items[0].Name)
}

func TestMatchExactGVK(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource matches
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, []string{"apps"}, []string{"v1"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

func TestMatchWildcardVersion(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource matches
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, []string{"apps"}, []string{"*"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

func TestMatchWildcardGroup(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource matches
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, []string{"*"}, []string{"v1"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

func TestMatchWildcardKind(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource matches
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, []string{"apps"}, []string{"v1"}, []string{"*"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

func TestNoMatchExactGVK(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// One namespace, not including the namespace of the deployment
	ws.createNamespace(t, "default", nil)
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource does not match
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, []string{"apps"}, []string{"v1"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

func TestNoMatchWildcardVersion(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource does not match
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, []string{"foo"}, []string{"*"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

func TestNoMatchWildcardGroup(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource does not match
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, []string{"*"}, []string{"foo"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

func TestNoMatchWildcardKind(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource does not match
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, []string{"apps"}, []string{"foo"}, []string{"*"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

func TestMatchLabelsObjectSelectorMatch(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	labels := map[string]string{
		"test-label": "true",
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", labels)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// Object selector with match labels
	objectSelector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	// workload resource match
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, nil, nil, nil)
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

func TestMatchExpressionsObjectSelectorMatch(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	labels := map[string]string{
		"test-label": "true",
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", labels)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

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
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, []string{"*"}, []string{"*"}, []string{"*"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

func TestMatchLabelsObjectSelectorNoMatch(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	labels := map[string]string{
		"test-label": "true",
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy", nil)

	// Two namespace
	ws.createNamespace(t, "default", nil)
	ws.createNamespace(t, "test-ns", nil)
	namespaces, err := ws.getMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// Object selector with match labels
	objectSelector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	// workload resource match
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, nil, nil, nil)
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
	uout, err := w.DynamicClient.Resource(resource).Namespace(namespace).Create(context.TODO(), u, metav1.CreateOptions{})
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
