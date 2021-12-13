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

func TestEmptyNamespaceSelector(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create a couple of namespaces
	ws.createNamespace(t, "default")
	ws.createNamespace(t, "test-ns")

	// Empty Namespace Selector
	namespaceSelector := &metav1.LabelSelector{}
	namespaces, err := ws.GetMatchingNamespaces(namespaceSelector)
	assert.NoError(t, err, "unexpected error listing namespaces")
	assert.Len(t, namespaces.Items, 2, "namespace list length incorrect")
}

func TestNilNamespaceSelector(t *testing.T) {
	ws := &WorkloadSelector{
		KubeClient: fake.NewSimpleClientset(),
	}

	// Create a couple of namespaces
	ws.createNamespace(t, "default")
	ws.createNamespace(t, "test-ns")

	// nil Namespace Selector
	namespaces, err := ws.GetMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error listing namespaces")
	assert.Len(t, namespaces.Items, 2, "namespace list length incorrect")
}

func TestMatchLabelsNamespaceSelector(t *testing.T) {
}

func TestMatchExpressionsNamespaceSelector(t *testing.T) {
}

func TestNamespaceMatchExactGVK(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy")

	// TODO: create namespace list instead of call createNamespace and getMatchingNamespaces
	// Two namespace
	ws.createNamespace(t, "default")
	ws.createNamespace(t, "test-ns")
	namespaces, err := ws.GetMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource matches
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, []string{"apps"}, []string{"v1"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.True(t, found, "expected to find match")
}

func TestNamespaceNoMatchExactGVK(t *testing.T) {
	ws := &WorkloadSelector{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}

	// Create deployment
	deploy := ws.createDeployment(t, "test-ns", "test-deploy")

	// TODO: create namespace list instead of call createNamespace and getMatchingNamespaces
	// One namespace
	ws.createNamespace(t, "default")
	namespaces, err := ws.GetMatchingNamespaces(nil)
	assert.NoError(t, err, "unexpected error getting namespace list")

	// No object selector
	objectSelector := &metav1.LabelSelector{}

	// workload resource does not match
	found, err := ws.doesWorkloadMatch(deploy, namespaces, objectSelector, []string{"apps"}, []string{"v1"}, []string{"deployment"})
	assert.NoError(t, err, "unexpected error matching resource")
	assert.False(t, found, "expected to not find match")
}

func (w *WorkloadSelector) createNamespace(t *testing.T, name string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := w.KubeClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	assert.NoError(t, err, "unexpected error creating namespace")
}

func (w *WorkloadSelector) createDeployment(t *testing.T, namespace string, name string) *unstructured.Unstructured {
	u := newUnstructured("apps/v1", "Deployment", namespace, name)
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
