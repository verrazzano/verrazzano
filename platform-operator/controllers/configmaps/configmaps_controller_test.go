package configmaps

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

var defaultNamespace = "default"

func TestConfigMapsReconcile(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	expectListVerrazzanoExists(mock, testVZ)
	expectGetConfigMapExists(mock, &testConfigMap, testNS, testCMName)
	request := newRequest(testNS, testCMName)
	reconciler := newConfigMapReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)
	asserts.Nil(err)
	asserts.Empty(result)
}

func expectListVerrazzanoExists(mock *mocks.MockClient, verrazzanoToUse vzapi.Verrazzano) {
	mock.EXPECT().
		List(gomock.Any(), &vzapi.VerrazzanoList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, list *vzapi.VerrazzanoList, opts ...client.ListOption) error {
			list.Items = append(list.Items, verrazzanoToUse)
			return nil
		}).AnyTimes()
}

func expectGetConfigMapExists(mock *mocks.MockClient, cmToUse *corev1.ConfigMap, namespace string, name string) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, cm *corev1.ConfigMap) error {
			cm = cmToUse
			return nil
		}).AnyTimes()
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
