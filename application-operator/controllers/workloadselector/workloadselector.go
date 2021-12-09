package workloadselector

import (
	"context"
	"strings"

	"github.com/gertd/go-pluralize"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// WorkloadSelector type for accessing functions
type WorkloadSelector struct {
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
}

// GetMatchingNamespaces returns a list of namespaces matching the specified namespace label selector
func (w *WorkloadSelector) GetMatchingNamespaces(namespaceSelector *metav1.LabelSelector) (*corev1.NamespaceList, error) {
	labelMap, err := metav1.LabelSelectorAsMap(namespaceSelector)
	if err != nil {
		return nil, err
	}
	options := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}

	return w.KubeClient.CoreV1().Namespaces().List(context.TODO(), options)
}

func (w *WorkloadSelector) doesWorkloadMatch(workload *unstructured.Unstructured, namespaces *corev1.NamespaceList, objectSelector *metav1.LabelSelector, groups []string, versions []string, kinds []string) (bool, error) {
	// If no namespace list is passed then nothing to check
	if len(namespaces.Items) == 0 {
		return false, nil
	}

	// Get the group and version of the workload resource
	gv, err := schema.ParseGroupVersion(workload.GetAPIVersion())
	if err != nil {
		return false, nil
	}

	// Check that the workload resource GVK matches expected GVKs
	if !checkMatch(gv.Version, versions) || !checkMatch(gv.Group, groups) || !checkMatch(workload.GetKind(), kinds) {
		return false, nil
	}

	for _, namespace := range namespaces.Items {
		resource := schema.GroupVersionResource{
			Group:    gv.Group,
			Version:  gv.Version,
			Resource: pluralize.NewClient().Plural(strings.ToLower(workload.GetKind())),
		}
		labelMap, err := metav1.LabelSelectorAsMap(objectSelector)
		if err != nil {
			return false, err
		}
		options := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(labelMap).String(),
		}
		objects, err := w.DynamicClient.Resource(resource).Namespace(namespace.Name).List(context.TODO(), options)
		if err != nil {
			return false, err
		}
		for _, object := range objects.Items {
			if object.GetName() == workload.GetName() {
				return true, nil
			}
		}
	}

	return false, nil
}

func checkMatch(match string, matches []string) bool {
	if len(matches) == 0 {
		return true
	}
	for _, value := range matches {
		if value == "*" {
			return true
		}
		if value == strings.ToLower(match) {
			return true
		}
	}

	return false
}
