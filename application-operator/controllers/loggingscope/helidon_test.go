// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kerrs "k8s.io/apimachinery/pkg/api/errors"

	kcore "k8s.io/api/core/v1"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/mocks"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	kapps "k8s.io/api/apps/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TestHelidoHandlerApply tests the creation of the FLUENTD sidecard in the application pod
// GIVEN an application workload referred in a loggingScope
// WHEN Apply is called on a managed cluster with the default VMI secret of the managed cluster
// THEN ensure that the expected FLUENTD sidecar container is created and managed cluster VMI secret
// copied to app NS
func TestHelidonHandlerApply_ManagedCluster(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	namespace := "hello-ns"
	workloadName := "hello-workload"
	appContainerName := "testApply-app-container"
	workload := workloadOf(namespace, workloadName)
	managedClusterVmiSecretKey := clusters.GetManagedClusterElasticsearchSecretKey()
	esSecretName := managedClusterVmiSecretKey.Name
	scope := newLoggingScope(namespace, "esHost", esSecretName)

	commonExpectationsForApply(mockClient, namespace, appContainerName, workloadName, esSecretName)
	expectationsForApplyUseManagedClusterSecret(t, mockClient, namespace)
	expectDeploymentUpdatedWithFluentd(t, mockClient, appContainerName)

	h := &HelidonHandler{
		Client: mockClient,
		Log:    log.NullLogger{},
	}
	res, err := h.Apply(context.Background(), workload, scope)
	asserts.Nil(t, res)
	asserts.Nil(t, err)
}

// TestHelidonHandlerApply_NonManagedCluster tests the creation of the FLUENTD sidecar in the
// application pod on a non-managed cluster
// GIVEN an application workload referred in a loggingScope
// WHEN Apply is called on a non-managed cluster
// THEN ensure that the expected FLUENTD sidecar container is created with the secret supplied in loggingscope
// and only an empty secret is created in app NS for mounting on fluentd container
func TestHelidonHandlerApply_NonManagedCluster(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	namespace := "hello-ns"
	workloadName := "hello-workload"
	appContainerName := "testApply-app-container"
	esSecretName := "myEsSecret"
	workload := workloadOf(namespace, workloadName)
	scope := newLoggingScope(namespace, "esHost", esSecretName)

	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *kapps.Deployment) error {
			appContainer := kcore.Container{Name: appContainerName, Image: "test-app-container-image"}
			deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers, appContainer)
			return nil
		})
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: fluentdConfigMapName(workloadName)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *kcore.ConfigMap) error {
			return kerrs.NewNotFound(schema.ParseGroupResource("v1.ConfigMap"), fluentdConfigMapName(workloadName))
		})
	mockClient.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, conf *kcore.ConfigMap, opt *client.CreateOptions) error {
			return nil
		})
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: esSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *kcore.Secret) error {
			return kerrs.NewNotFound(schema.ParseGroupResource("v1.ConfigMap"), fluentdConfigMapName(workloadName))
		})
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "verrazzano-system", Name: "verrazzano"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, sec *kcore.Secret) error {
			vmiSecret(sec)
			return nil
		})
	mockClient.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *kcore.Secret, opt *client.CreateOptions) error {
			return nil
		})
	mockClient.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, dep *kapps.Deployment) error {
			appCon, fluentdFound := searchContainers(dep.Spec.Template.Spec.Containers)
			asserts.Equal(t, appContainerName, appCon)
			asserts.True(t, fluentdFound)
			return nil
		})

	h := &HelidonHandler{
		Client: mockClient,
		Log:    log.NullLogger{},
	}
	res, err := h.Apply(context.Background(), workload, scope)
	asserts.Nil(t, res)
	asserts.Nil(t, err)
	mocker.Finish()
}

// TestHelidoHandlerApplyErrorWaitingForDeploymentUpdate tests Apply waiting for Deployment update
// GIVEN an application workload referred in a loggingScope
// WHEN updating the appconfig
// THEN ensure that the Apply call must error if the Deployment is not updated
func TestHelidoHandlerApplyRequeueForDeploymentUpdate(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	namespace := "hello-ns"
	workloadName := "hello-workload"
	esSecretName := "myEsSecret"
	appContainerName := "testUpdate-app-container"
	workload := workloadOf(namespace, workloadName)
	scope := newLoggingScope(namespace, "esHost", esSecretName)
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *kapps.Deployment) error {
			appContainer := kcore.Container{Name: appContainerName, Image: "test-app-container-image"}
			fluentdContainer := CreateFluentdContainer(workload.Namespace, workload.Name, "appContainer", scope.Spec.FluentdImage, scope.Spec.SecretName, scope.Spec.ElasticSearchURL)
			deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers, appContainer, fluentdContainer)
			vol := kcore.Volume{
				Name: "app-volume",
				VolumeSource: kcore.VolumeSource{
					HostPath: &kcore.HostPathVolumeSource{
						Path: "/var/log",
					},
				},
			}
			deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, vol, CreateFluentdConfigMapVolume(workload.Name))
			deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, CreateFluentdSecretVolume(scope.Spec.SecretName))
			volumes := CreateFluentdHostPathVolumes()
			for _, volume := range volumes {
				deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volume)
			}
			return nil
		})

	h := &HelidonHandler{
		Client: mockClient,
		Log:    log.NullLogger{},
	}
	res, err := h.Apply(context.Background(), workload, scope)
	asserts.NotNil(t, res)
	asserts.True(t, res.Requeue)
	asserts.Nil(t, err)
	mocker.Finish()
}

// TestHelidoHandlerRemove tests the removal of the FLUENTD sidecard in the application pod
// GIVEN an application workload referred in a loggingScope
// WHEN Remove is called
// THEN ensure that the expected FLUENTD sidecar container is removed
func TestHelidoHandlerRemove(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	namespace := "hello-ns"
	workloadName := "hello-workload"
	esSecretName := "myEsSecret"
	appContainerName := "testDelete-app-container"
	workload := workloadOf(namespace, workloadName)
	scope := newLoggingScope(namespace, "esHost", esSecretName)
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *kapps.Deployment) error {
			appContainer := kcore.Container{Name: appContainerName, Image: "test-app-container-image"}
			fluentdContainer := CreateFluentdContainer(workload.Namespace, workload.Name, "appContainer", scope.Spec.FluentdImage, scope.Spec.SecretName, scope.Spec.ElasticSearchURL)
			deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers, appContainer, fluentdContainer)
			vol := kcore.Volume{
				Name: "app-volume",
				VolumeSource: kcore.VolumeSource{
					HostPath: &kcore.HostPathVolumeSource{
						Path: "/var/log",
					},
				},
			}
			deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, vol, CreateFluentdConfigMapVolume(workload.Name))
			deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, CreateFluentdSecretVolume(scope.Spec.SecretName))
			volumes := CreateFluentdHostPathVolumes()
			for _, volume := range volumes {
				deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volume)
			}
			return nil
		})

	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: fluentdConfigMapName(workloadName)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *kcore.ConfigMap) error {
			return nil
		})
	mockClient.EXPECT().
		Delete(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, conf *kcore.ConfigMap) error {
			return nil
		})
	mockClient.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, dep *kapps.Deployment) error {
			appCon, fluentdFound := searchContainers(dep.Spec.Template.Spec.Containers)
			asserts.Equal(t, appContainerName, appCon)
			asserts.False(t, fluentdFound)
			asserts.Equal(t, 1, len(dep.Spec.Template.Spec.Volumes))
			return nil
		})

	h := &HelidonHandler{
		Client: mockClient,
		Log:    log.NullLogger{},
	}
	removed, err := h.Remove(context.Background(), workload, scope)
	asserts.True(t, removed)
	asserts.Nil(t, err)
	mocker.Finish()
}

func workloadOf(namespace, workloadName string) vzapi.QualifiedResourceRelation {
	return vzapi.QualifiedResourceRelation{
		APIVersion: "core.oam.dev/v1alpha2",
		Name:       workloadName,
		Namespace:  namespace,
		Kind:       "ContainerizedWorkload",
	}
}

// TestHelidoHandlerApply tests the creation of the FLUENTD sidecard failed with missing deployment
// GIVEN an application workload referred in a loggingScope
// WHEN Apply is called
// THEN ensure that Apply call returns a error
func TestHelidoHandlerApplyNoDeployment(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	namespace := "hello-ns"
	workloadName := "hello-workload"
	esSecretName := "myEsSecret"
	workload := workloadOf(namespace, workloadName)
	scope := newLoggingScope(namespace, "esHost", esSecretName)
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *kapps.Deployment) error {
			return kerrs.NewNotFound(schema.ParseGroupResource("v1.Deployment"), workloadName)
		})
	h := &HelidonHandler{
		Client: mockClient,
		Log:    log.NullLogger{},
	}
	res, err := h.Apply(context.Background(), workload, scope)
	asserts.Nil(t, res)
	asserts.NotNil(t, err)
	asserts.True(t, kerrs.IsNotFound(err))
	mocker.Finish()
}

// TestHelidoHandlerRemoveNoDeployment tests removal of the FLUENTD sidecard failed with missing deployment
// GIVEN an application workload referred in a loggingScope
// WHEN Remove is called
// THEN ensure that Remove call returns a error
func TestHelidoHandlerRemoveNoDeployment(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	namespace := "hello-ns"
	workloadName := "hello-workload"
	esSecretName := "myEsSecret"
	workload := workloadOf(namespace, workloadName)
	scope := newLoggingScope(namespace, "esHost", esSecretName)
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *kapps.Deployment) error {
			return kerrs.NewNotFound(schema.ParseGroupResource("v1.Deployment"), workloadName)
		})
	h := &HelidonHandler{
		Client: mockClient,
		Log:    log.NullLogger{},
	}
	removed, err := h.Remove(context.Background(), workload, scope)
	asserts.NotNil(t, err)
	asserts.True(t, removed)
	asserts.True(t, kerrs.IsNotFound(err))
	mocker.Finish()
}

// newLoggingScope creates a test logging scope
func newLoggingScope(namespace, esHost, esSecret string) *vzapi.LoggingScope {
	scope := vzapi.LoggingScope{}
	scope.TypeMeta = kmeta.TypeMeta{APIVersion: vzapi.GroupVersion.Identifier(), Kind: vzapi.LoggingScopeKind}
	scope.ObjectMeta = kmeta.ObjectMeta{Namespace: namespace, Name: "testScopeName"}
	scope.Spec.ElasticSearchURL = "http://esHost:9200"
	scope.Spec.SecretName = esSecret
	scope.Spec.FluentdImage = "fluentd/image/location"
	return &scope
}

func vmiSecret(sec *kcore.Secret) *kcore.Secret {
	sec.Name = "verrazzano"
	sec.Namespace = "verrazzano-system"
	sec.Data = map[string][]byte{
		secretUserKey:     []byte("verrazzano"),
		secretPasswordKey: []byte(genPassword(10)),
	}
	return sec
}

var passwordChars = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func genPassword(passSize int) string {
	rand.Seed(time.Now().UnixNano())
	var b strings.Builder
	for i := 0; i < passSize; i++ {
		b.WriteRune(passwordChars[rand.Intn(len(passwordChars))])
	}
	return b.String()
}

// expectDeploymentUpdatedWithFluentd - expect that the deployment is updated with a fluentd
// sidecar container in addition to the app container
func expectDeploymentUpdatedWithFluentd(t *testing.T, mockClient *mocks.MockClient, appContainerName string) {
	// UPDATE deployment to add fluentd container
	mockClient.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, dep *kapps.Deployment) error {
			appCon, fluentdFound := searchContainers(dep.Spec.Template.Spec.Containers)
			asserts.Equal(t, appContainerName, appCon)
			asserts.True(t, fluentdFound)
			return nil
		})
}

// expectationsForApplyUseManagedClusterSecret - adds expectations when the esSecretName is the same
// as the managed cluster's ES secret name
func expectationsForApplyUseManagedClusterSecret(t *testing.T, mockClient *mocks.MockClient, namespace string) {
	managedClusterVmiSecretKey := clusters.GetManagedClusterElasticsearchSecretKey()
	esSecretName := managedClusterVmiSecretKey.Name

	// GET supplied ES secret which we return as not found
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: esSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *kcore.Secret) error {
			return kerrs.NewNotFound(schema.ParseGroupResource("v1.Secret"), esSecretName)
		})

	// Check if managed cluster secret VMI secret exists - return a valid secret
	mockClient.EXPECT().
		Get(gomock.Any(), managedClusterVmiSecretKey, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, sec *kcore.Secret) error {
			sec.Name = managedClusterVmiSecretKey.Name
			sec.Namespace = managedClusterVmiSecretKey.Namespace
			sec.Data = map[string][]byte{
				"username": []byte("verrazzano"),
				"password": []byte(genPassword(10)),
				//todo ca-bundle
			}
			return nil
		})

	managedClusterSecretNameInAppNS := types.NamespacedName{Namespace: namespace, Name: managedClusterVmiSecretKey.Name}

	// Check if managed cluster secret VMI secret is already in app namespace - return not found
	mockClient.EXPECT().
		Get(gomock.Any(), managedClusterSecretNameInAppNS, gomock.Not(gomock.Nil())).
		Return(kerrs.NewNotFound(schema.ParseGroupResource("v1.Secret"), managedClusterSecretNameInAppNS.String()))

	// simulate managed cluster ES secret existing (GET is called once to check if we should use
	// managed cluster, and once to actually perform the copy over to app NS)
	expectedData := map[string][]byte{"username": []byte("someuser")}
	mockClient.EXPECT().
		Get(gomock.Any(), managedClusterVmiSecretKey, gomock.Not(gomock.Nil())).
		Times(2).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *kcore.Secret) error {
			secret.Name = managedClusterVmiSecretKey.Name
			secret.Namespace = managedClusterVmiSecretKey.Namespace
			secret.Data = expectedData
			return nil
		})

	// CREATE managed cluster VMI secret in app namespace
	mockClient.EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&kcore.Secret{}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *kcore.Secret, opt *client.CreateOptions) error {
			asserts.Equal(t, managedClusterSecretNameInAppNS.Name, sec.Name)
			asserts.Equal(t, managedClusterSecretNameInAppNS.Namespace, sec.Namespace)
			return nil
		})
}

// expectationsForApplyNonManagedCluster - adds expectations for the case where this is NOT a managed cluster
// i.e. the managed cluster registration secret does not exist. In this case, we don't expect any
// secrets to be copied to app NS. We do expect an empty secret with no data to be created in
// the app NS for volume mounting on fluentd
func expectationsForApplyNonManagedCluster(t *testing.T, mockClient *mocks.MockClient, namespace string, esSecretName string) {
	managedClusterVmiSecretKey := clusters.GetManagedClusterElasticsearchSecretKey()

	// GET supplied ES secret which we return as not found
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: esSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *kcore.Secret) error {
			return kerrs.NewNotFound(schema.ParseGroupResource("v1.Secret"), esSecretName)
		})

	// Check if managed cluster secret VMI secret exists - return not-found since this is NOT a managed cluster
	mockClient.EXPECT().
		Get(gomock.Any(), managedClusterVmiSecretKey, gomock.Not(gomock.Nil())).
		Return(kerrs.NewNotFound(schema.ParseGroupResource("v1.Secret"), managedClusterVmiSecretKey.Name))

	// Check that empty secret is created in app NS, with no data contents
	mockClient.EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&kcore.Secret{}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *kcore.Secret, opt *client.CreateOptions) error {
			asserts.Equal(t, esSecretName, sec.Name)
			asserts.Equal(t, namespace, sec.Namespace)
			asserts.Nil(t, sec.Data)
			return nil
		})

	// No further action is expected on a non-managed cluster
}

// commonExpectationsForApply - adds expectations common to all vanilla apply use cases - we expect
// apply to get the workload and the fluentd config map and create it if not found.
func commonExpectationsForApply(mockClient *mocks.MockClient, namespace string, appContainerName string, workloadName string, esSecretName string) {
	// GET workload
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *kapps.Deployment) error {
			appContainer := kcore.Container{Name: appContainerName, Image: "test-app-container-image"}
			deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers, appContainer)
			return nil
		})

	// GET fluentd config map which we return as not found
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: fluentdConfigMapName(workloadName)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *kcore.ConfigMap) error {
			return kerrs.NewNotFound(schema.ParseGroupResource("v1.ConfigMap"), fluentdConfigMapName(workloadName))
		})

	// CREATE fluentd config map since it was not found
	mockClient.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, conf *kcore.ConfigMap, opt *client.CreateOptions) error {
			return nil
		})
}
