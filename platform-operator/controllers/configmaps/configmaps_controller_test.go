package configmaps

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestConfigMapReconciler tests Reconciler method of Configmap controller.
func TestConfigMapReconciler(t *testing.T) {
	asserts := assert.New(t)
	cli := fake.NewClientBuilder().WithObjects(&testVZ, &testConfigMap).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request := newRequest(testNS, testCMName)
	reconciler := newConfigMapReconciler(cli)
	res, err := reconciler.Reconcile(context.TODO(), request)

	asserts.NoError(err)
	asserts.Equal(false, res.Requeue)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
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
	vzLog := vzlog.DefaultLogger()
	scheme := newScheme()
	reconciler := VerrazzanoConfigMapsReconciler{
		Client: c,
		Scheme: scheme,
		log:    vzLog,
	}
	return reconciler
}
