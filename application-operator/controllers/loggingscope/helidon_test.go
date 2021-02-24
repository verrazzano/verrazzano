// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"math/rand"
	"strings"
	"testing"
	"time"

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
// WHEN Apply is called
// THEN ensure that the expected FLUENTD sidecar container is created
func TestHelidoHandlerApply(t *testing.T) {
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
