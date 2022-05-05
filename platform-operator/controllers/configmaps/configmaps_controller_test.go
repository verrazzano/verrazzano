package configmaps

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

var defaultNamespace = "default"

func TestVerrazzanoConfigMapsReconciler_Reconcile(t *testing.T) {

}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return scheme
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name}}
}

func newConfigMapReconciler(c client.Client) VerrazzanoConfigMapsReconciler {
	scheme := newScheme()
	reconciler := VerrazzanoConfigMapsReconciler{
		Client: c,
		Scheme: scheme,
	}
	return reconciler
}
