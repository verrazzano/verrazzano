package controllers

import (
	"context"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

const namespace = "unit-mcsecret-namespace"
const crName = "unit-mcsecret"

// TestReconcilerSetupWithManager test the creation of the MultiClusterSecretReconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var mgr *mocks.MockManager
	var cli *mocks.MockClient
	var scheme *runtime.Scheme
	var reconciler MultiClusterSecretReconciler
	var err error

	mocker = gomock.NewController(t)
	mgr = mocks.NewMockManager(mocker)
	cli = mocks.NewMockClient(mocker)
	scheme = runtime.NewScheme()
	clustersv1alpha1.AddToScheme(scheme)
	reconciler = MultiClusterSecretReconciler{Client: cli, Scheme: scheme}
	mgr.EXPECT().GetConfig().Return(&rest.Config{})
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(log.NullLogger{})
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}

// TestReconcileCreateSecret tests the basic happy path of reconciling a MultiClusterSecret. We
// expect to write out a K8S secret
// GIVEN a MultiClusterSecret resource is created
// WHEN the controller Reconcile function is called
// THEN expect a Secret to be created
func TestReconcileCreateSecret(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	secretData := map[string][]byte{"username": []byte("aaaaa")}

	mcSecretSample := getSampleMCSecret(namespace, crName, secretData)

	// expect a call to fetch the MultiClusterSecret (and another to fetch and update corev1.Secret)
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcSecret *clustersv1alpha1.MultiClusterSecret) error {
				mcSecret.ObjectMeta = mcSecretSample.ObjectMeta
				mcSecret.TypeMeta = mcSecretSample.TypeMeta
				mcSecret.Spec = mcSecretSample.Spec
			return nil
		})

	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Secret"}, crName))

	// expect a call to create the K8S secret
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, s *v1.Secret, opts ...client.CreateOption) error {
			assert.Equal(v1.SecretTypeOpaque, s.Type)
			assert.Equal(namespace, s.ObjectMeta.Namespace)
			assert.Equal(crName, s.ObjectMeta.Name)
			assert.Equal(secretData, s.Data)
			assert.Equal(1, len(s.OwnerReferences))
			assert.Equal("MultiClusterSecret", s.OwnerReferences[0].Kind)
			assert.Equal(clustersv1alpha1.GroupVersion.String(), s.OwnerReferences[0].APIVersion)
			assert.Equal(crName, s.OwnerReferences[0].Name)
			return nil
		})

	// expect a call to update the status of the multicluster secret
	cli.EXPECT().Status().Return(mockStatusWriter)

	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&mcSecretSample)).
		Return(nil)

	// create a request and reconcile it
	request := newRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileUpdateSecret tests the path of reconciling a MultiClusterSecret when the underlying
// secret already exists i.e. update
// expect to update a K8S secret
// GIVEN a MultiClusterSecret resource is created
// WHEN the controller Reconcile function is called
// THEN expect a Secret to be updated
func TestReconcileUpdateSecret(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	newSecretData := map[string][]byte{"username": []byte("aaaaa")}
	existingSecretData := map[string][]byte{"username": []byte("existing")}

	mcSecretSample := getSampleMCSecret(namespace, crName, newSecretData)

	// expect a call to fetch the MultiClusterSecret (and another to fetch and update corev1.Secret)
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcSecret *clustersv1alpha1.MultiClusterSecret) error {
			mcSecret.ObjectMeta = mcSecretSample.ObjectMeta
			mcSecret.TypeMeta = mcSecretSample.TypeMeta
			mcSecret.Spec = mcSecretSample.Spec
			return nil
		})

	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Data = existingSecretData
			secret.ObjectMeta = mcSecretSample.ObjectMeta
			return nil
		})

	// expect a call to update the K8S secret with the new secret data
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, s *v1.Secret, opts ...client.CreateOption) error {
			assert.Equal(v1.SecretTypeOpaque, s.Type)
			assert.Equal(namespace, s.ObjectMeta.Namespace)
			assert.Equal(crName, s.ObjectMeta.Name)
			assert.Equal(newSecretData, s.Data)
			return nil
		})

	// expect a call to update the status of the multicluster secret
	cli.EXPECT().Status().Return(mockStatusWriter)

	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&mcSecretSample)).
		Return(nil)

	// create a request and reconcile it
	request := newRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

func getSampleMCSecret(ns string, name string, secretData map[string][]byte) clustersv1alpha1.MultiClusterSecret {
	var mcSecret clustersv1alpha1.MultiClusterSecret
	mcSecret.Spec.Template = clustersv1alpha1.SecretTemplate{Type: v1.SecretTypeOpaque, Data: secretData}
	mcSecret.ObjectMeta.Namespace = namespace
	mcSecret.ObjectMeta.Name = crName
	mcSecret.APIVersion = clustersv1alpha1.GroupVersion.String()
	mcSecret.Kind = "MultiClusterSecret"
	mcSecret.Spec.Placement.Clusters = []clustersv1alpha1.Cluster{clustersv1alpha1.Cluster{Name: "myCluster"}}
	return mcSecret
}

// newReconciler creates a new reconciler for testing
// c - The K8s client to inject into the reconciler
func newReconciler(c client.Client) MultiClusterSecretReconciler {
	return MultiClusterSecretReconciler{
		Client: c,
		Log:    ctrl.Log.WithName("test"),
		Scheme: newScheme(),
	}
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clustersv1alpha1.AddToScheme(scheme)
	return scheme
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}
}