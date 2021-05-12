// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kerrs "k8s.io/apimachinery/pkg/api/errors"

	kcore "k8s.io/api/core/v1"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/mocks"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	kapps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TestHelidonHandlerApply_ManagedCluster tests the creation of the FLUENTD sidecar in the
// application pod with default settings on a managed cluster
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
	loggingSecretName := clusters.MCRegistrationSecretFullName.Name
	scope := newLoggingScope(namespace, "esHost", loggingSecretName)

	expectationsForWorkload(mockClient, namespace, appContainerName, workloadName)
	expectationsForApplyUseManagedClusterSecret(t, mockClient, namespace)
	expectationsForConfigMap(t, mockClient, namespace, appContainerName, workloadName, true)
	expectDeploymentUpdatedWithFluentd(t, mockClient, appContainerName)

	h := &HelidonHandler{
		Client: mockClient,
		Log:    log.NullLogger{},
	}
	res, err := h.Apply(context.Background(), workload, scope)
	asserts.Nil(t, res)
	asserts.Nil(t, err)
	mocker.Finish()
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
	workload := workloadOf(namespace, workloadName)
	loggingSecretName := "someLoggingSecret"
	scope := newLoggingScope(namespace, "esHost", loggingSecretName)

	expectationsForWorkload(mockClient, namespace, appContainerName, workloadName)
	expectationsForApplyNonManagedCluster(t, mockClient, namespace, loggingSecretName)
	expectationsForConfigMap(t, mockClient, namespace, appContainerName, workloadName, false)
	expectDeploymentUpdatedWithFluentd(t, mockClient, appContainerName)

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
	loggingSecretName := "myLoggingSecret"
	appContainerName := "testUpdate-app-container"
	workload := workloadOf(namespace, workloadName)
	scope := newLoggingScope(namespace, "esHost", loggingSecretName)
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *kapps.Deployment) error {
			appContainer := kcore.Container{Name: appContainerName, Image: "test-app-container-image"}
			fluentdContainer := CreateFluentdContainer(scope, workload.Namespace, workload.Name)
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
			deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, CreateFluentdSecretVolume(scope.SecretName))
			volumes := CreateFluentdHostPathVolumes()
			deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volumes...)
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

// TestHelidoHandlerRemove tests the removal of the FLUENTD sidecar in the application pod
// GIVEN an application workload referred in a loggingScope
// WHEN Remove is called
// THEN ensure that the expected FLUENTD sidecar container is removed
func TestHelidoHandlerRemove(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	namespace := "hello-ns"
	workloadName := "hello-workload"
	loggingSecretName := "myLoggingSecret"
	appContainerName := "testDelete-app-container"
	workload := workloadOf(namespace, workloadName)
	scope := newLoggingScope(namespace, "esHost", loggingSecretName)
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *kapps.Deployment) error {
			appContainer := kcore.Container{Name: appContainerName, Image: "test-app-container-image"}
			fluentdContainer := CreateFluentdContainer(scope, workload.Namespace, workload.Name)
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
			deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, CreateFluentdSecretVolume(scope.SecretName))
			volumes := CreateFluentdHostPathVolumes()
			deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volumes...)
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
			appCons, fluentdFound := searchContainers(dep.Spec.Template.Spec.Containers)
			asserts.Contains(t, appCons, appContainerName)
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

// TestHelidoHandlerApply tests the creation of the FLUENTD sidecar failed with missing deployment
// GIVEN an application workload referred in a loggingScope
// WHEN Apply is called
// THEN ensure that Apply call returns a error
func TestHelidoHandlerApplyNoDeployment(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	namespace := "hello-ns"
	workloadName := "hello-workload"
	loggingSecretName := "myLoggingSecret"
	workload := workloadOf(namespace, workloadName)
	scope := newLoggingScope(namespace, "esHost", loggingSecretName)
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

// TestHelidoHandlerRemoveNoDeployment tests removal of the FLUENTD sidecar failed with missing deployment
// GIVEN an application workload referred in a loggingScope
// WHEN Remove is called
// THEN ensure that Remove call returns a error
func TestHelidoHandlerRemoveNoDeployment(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	namespace := "hello-ns"
	workloadName := "hello-workload"
	loggingSecretName := "myLoggingSecret"
	workload := workloadOf(namespace, workloadName)
	scope := newLoggingScope(namespace, "esHost", loggingSecretName)
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
func newLoggingScope(namespace, esHost, esSecret string) *LoggingScope {
	scope := LoggingScope{}
	scope.ElasticSearchURL = "http://esHost:9200"
	scope.SecretName = esSecret
	scope.SecretNamespace = namespace
	scope.FluentdImage = "fluentd/image/location"
	return &scope
}

func vmiSecret(sec *kcore.Secret) *kcore.Secret {
	sec.Name = "verrazzano"
	sec.Namespace = "verrazzano-system"
	sec.Data = map[string][]byte{
		constants.ElasticsearchUsernameData: []byte("verrazzano"),
		constants.ElasticsearchPasswordData: []byte(genPassword(10)),
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
			appCons, fluentdFound := searchContainers(dep.Spec.Template.Spec.Containers)
			asserts.Contains(t, appCons, appContainerName)
			asserts.True(t, fluentdFound)
			return nil
		})
}

// expectationsForApplyUseManagedClusterSecret - adds expectations when the loggingSecretName is the same
// as the managed cluster's ES secret name
func expectationsForApplyUseManagedClusterSecret(t *testing.T, mockClient *mocks.MockClient, namespace string) {
	managedClusterVmiSecretKey := clusters.MCRegistrationSecretFullName
	loggingSecretName := managedClusterVmiSecretKey.Name

	// GET supplied ES secret which we return as not found
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: loggingSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *kcore.Secret) error {
			return kerrs.NewNotFound(schema.ParseGroupResource("v1.Secret"), loggingSecretName)
		})

	// Check if managed cluster secret VMI secret exists - return a valid secret
	// (GET is called once to check if we should use managed cluster, and once to actually
	// perform the copy over to app NS)
	mockClient.EXPECT().
		Get(gomock.Any(), managedClusterVmiSecretKey, gomock.Not(gomock.Nil())).
		Times(2).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, sec *kcore.Secret) error {
			sec.Name = managedClusterVmiSecretKey.Name
			sec.Namespace = managedClusterVmiSecretKey.Namespace
			sec.Data = map[string][]byte{
				"es-url":    []byte("test-es-url"),
				"username":  []byte("verrazzano"),
				"password":  []byte(genPassword(10)),
				"ca-bundle": []byte("test-ca-bundle"),
			}
			return nil
		})

	managedClusterSecretNameInAppNS := types.NamespacedName{Namespace: namespace, Name: managedClusterVmiSecretKey.Name}

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
func expectationsForApplyNonManagedCluster(t *testing.T, mockClient *mocks.MockClient, namespace string, loggingSecretName string) {
	// GET supplied ES secret which we return as not found
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: loggingSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *kcore.Secret) error {
			return kerrs.NewNotFound(schema.ParseGroupResource("v1.Secret"), loggingSecretName)
		})

	// Check that empty secret is created in app NS, with no data contents
	mockClient.EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&kcore.Secret{}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *kcore.Secret, opt *client.CreateOptions) error {
			asserts.Equal(t, loggingSecretName, sec.Name)
			asserts.Equal(t, namespace, sec.Namespace)
			asserts.Nil(t, sec.Data)
			return nil
		})

	// No further action is expected on a non-managed cluster
}

// expectationsForWorkload - adds expectations to get the workload
func expectationsForWorkload(mockClient *mocks.MockClient, namespace string, appContainerName string, workloadName string) {
	// GET workload
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *kapps.Deployment) error {
			appContainer := kcore.Container{Name: appContainerName, Image: "test-app-container-image"}
			deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers, appContainer)
			return nil
		})
}

// expectationsForConfigMap - adds expectations to get the workload and the fluentd config map and create it if not found.
func expectationsForConfigMap(t *testing.T, mockClient *mocks.MockClient, namespace string, appContainerName string, workloadName string, caFile bool) {
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
			asserts.Equal(t, caFile, strings.Contains(conf.Data["fluentd.conf"], CAFileConfig), "expect fluentd.conf to have ca_file")
			return nil
		})
}

func Test_getFluentdConfigurationForHelidon(t *testing.T) {
	tests := []struct {
		name             string
		requiresCABundle bool
		containsCAFile   bool
	}{
		{
			name:             "without ca-bundle",
			requiresCABundle: false,
			containsCAFile:   false,
		},
		{
			name:             "with ca-bundle",
			requiresCABundle: true,
			containsCAFile:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workloadName := "testworkload"
			appContainersNames := []string{"testc1", "testc2"}
			conf, _ := getFluentdConfigurationForHelidon(workloadName, appContainersNames, tt.requiresCABundle)
			got := strings.Contains(conf, CAFileConfig)
			if got != tt.containsCAFile {
				t.Errorf("getFluentdConfigurationForHelidon() containsCAFile = %v, want %v", got, tt.containsCAFile)
			}
		})
	}
}
