// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"testing"

	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var validSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      constants.MCAgentSecret,
		Namespace: constants.VerrazzanoSystemNamespace,
	},
	Data: map[string][]byte{constants.ClusterNameData: []byte("cluster1"), constants.AdminKubeconfigData: []byte("kubeconfig")},
}

// TestProcessAgentThreadNoProjects tests agent thread when no projects exist
// GIVEN a request to process the agent loop
// WHEN the a new VerrazzanoProjects resources exists
// THEN ensure that there are no calls to sync any multi-cluster resources
func TestProcessAgentThreadNoProjects(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)
	adminStatusMock := mocks.NewMockStatusWriter(adminMocker)

	// Managed Cluster - expect call to get the cluster registration secret.
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.ObjectMeta = validSecret.ObjectMeta
			secret.Data = validSecret.Data
			return nil
		})

	// Admin Cluster - expect a get followed by status update on VMC to record last agent connect time
	vmcName := types.NamespacedName{Name: string(validSecret.Data[constants.ClusterNameData]), Namespace: constants.VerrazzanoMultiClusterNamespace}
	expectGetAPIServerURLCalled(mcMock)
	expectGetPrometheusHostCalled(mcMock)
	expectAdminVMCStatusUpdateSuccess(adminMock, vmcName, adminStatusMock, assert)

	// Admin Cluster - expect call to list VerrazzanoProject objects - return an empty list
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...*client.ListOptions) error {
			return nil
		})

	// Managed Cluster - expect call to list VerrazzanoProject objects - return an empty list
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...*client.ListOptions) error {
			return nil
		})

	// Managed Cluster - expect call to list Namespace objects - return an empty list
	mcMock.EXPECT().
		List(gomock.Any(), &corev1.NamespaceList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, list *corev1.NamespaceList, opts ...*client.ListOptions) error {
			return nil
		})

	// Make the request
	s := &Syncer{
		AdminClient:        adminMock,
		LocalClient:        mcMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err := s.ProcessAgentThread()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
	assert.Equal(validSecret.ResourceVersion, s.SecretResourceVersion)
}

// TestProcessAgentThreadSecretDeleted tests agent thread when the registration secret is deleted
// GIVEN a request to process the agent loop
// WHEN the registration secret has been deleted
// THEN ensure that there are no calls to get VerrazzanoProject resources
func TestProcessAgentThreadSecretDeleted(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Managed Cluster - expect call to get the cluster registration secret.
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			return errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, name.Name)
		})

	// Do not expect any further calls because the registration secret no longer exists

	// Make the request
	s := &Syncer{
		AdminClient:        adminMock,
		LocalClient:        mcMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
		AgentSecretFound:   true,
		AgentSecretValid:   true,
	}

	err := s.ProcessAgentThread()

	// Validate the results
	asserts.Equal(t, false, s.AgentSecretFound)
	asserts.Equal(t, false, s.AgentSecretValid)

	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestValidateSecret tests secret validation function
func TestValidateSecret(t *testing.T) {
	assert := asserts.New(t)

	// Valid secret
	err := validateAgentSecret(&validSecret)
	assert.NoError(err)

	// A secret without a cluster name
	invalidSecret := validSecret
	invalidSecret.Data = map[string][]byte{constants.AdminKubeconfigData: []byte("kubeconfig")}
	err = validateAgentSecret(&invalidSecret)
	assert.Error(err)
	assert.Contains(err.Error(), fmt.Sprintf("missing the required field %s", constants.ClusterNameData))

	// A secret without a kubeconfig
	invalidSecret.Data = map[string][]byte{constants.ClusterNameData: []byte("cluster1")}
	err = validateAgentSecret(&invalidSecret)
	assert.Error(err)
	assert.Contains(err.Error(), fmt.Sprintf("missing the required field %s", constants.AdminKubeconfigData))
}

// Test_getEnvValue tests getEnvValue
// GIVEN a request for a specified ENV name
// WHEN the env array contains such an env
// THEN returns the env value, empty string if not found
func Test_getEnvValue(t *testing.T) {
	container := corev1.Container{}
	container.Env = []corev1.EnvVar{}
	asserts.Equal(t, "", getEnvValue(&[]corev1.Container{container}, registrationSecretVersion), "expected cluster name")
	container.Env = []corev1.EnvVar{
		{
			Name:  registrationSecretVersion,
			Value: "version1",
		},
	}
	asserts.Equal(t, "version1", getEnvValue(&[]corev1.Container{container}, registrationSecretVersion), "expected cluster name")
	container.Env = []corev1.EnvVar{
		{
			Name:  "env1",
			Value: "value1",
		},
		{
			Name:  registrationSecretVersion,
			Value: "version1",
		},
	}
	asserts.Equal(t, "version1", getEnvValue(&[]corev1.Container{container}, registrationSecretVersion), "expected cluster name")
}

// Test_getEnvValue tests updateEnvValue
// GIVEN a request for a specified ENV name/value
// WHEN the env array contains such an env
// THEN updates its value, append the env name/value if not found
func Test_updateEnvValue(t *testing.T) {
	var testEnvs []corev1.EnvVar
	newValue := "version2"
	newEnvs := updateEnvValue(testEnvs, registrationSecretVersion, newValue)
	asserts.Equal(t, registrationSecretVersion, newEnvs[0].Name, "expected env")
	asserts.Equal(t, newValue, newEnvs[0].Value, "expected env value")
	testEnvs = []corev1.EnvVar{
		{
			Name:  registrationSecretVersion,
			Value: "version1",
		},
	}
	newValue = "version2"
	newEnvs = updateEnvValue(testEnvs, registrationSecretVersion, newValue)
	asserts.Equal(t, registrationSecretVersion, newEnvs[0].Name, "expected env")
	asserts.Equal(t, newValue, newEnvs[0].Value, "expected env value")
	testEnvs = []corev1.EnvVar{
		{
			Name:  "env1",
			Value: "value1",
		},
		{
			Name:  registrationSecretVersion,
			Value: "version1",
		},
	}
	newEnvs = updateEnvValue(testEnvs, registrationSecretVersion, newValue)
	asserts.Equal(t, registrationSecretVersion, newEnvs[1].Name, "expected env")
	asserts.Equal(t, newValue, newEnvs[1].Value, "expected env value")
}

func getTestDeploymentSpec(secretVersion string) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Env: []corev1.EnvVar{
							{
								Name:  registrationSecretVersion,
								Value: secretVersion,
							},
						},
					},
				},
			},
		},
	}
}

// TestSyncer_updateDeployment tests updating deployment
// GIVEN a request to update the deployment
// WHEN the cluster registration secret has been changed
// THEN ensure that the deployment is updated
func TestSyncer_updateDeployment(t *testing.T) {
	deploymentName := "some-operrator"
	type fields struct {
		oldSecretVersion string
		newSecretVersion string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "new registration",
			fields: fields{
				oldSecretVersion: "",
				newSecretVersion: "version1",
			},
		},
		{
			name: "delete registration",
			fields: fields{
				oldSecretVersion: "version1",
				newSecretVersion: "",
			},
		},
		{
			name: "update registration",
			fields: fields{
				oldSecretVersion: "version1",
				newSecretVersion: "version2",
			},
		},
		{
			name: "no registration",
			fields: fields{
				oldSecretVersion: "",
				newSecretVersion: "",
			},
		},
		{
			name: "same registration",
			fields: fields{
				oldSecretVersion: "version1",
				newSecretVersion: "version1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVersion := tt.fields.oldSecretVersion
			newVersion := tt.fields.newSecretVersion

			// Managed cluster mocks
			mcMocker := gomock.NewController(t)
			mcMock := mocks.NewMockClient(mcMocker)

			// Managed Cluster - expect call to get the cluster registration secret.
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCRegistrationSecret}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
					secret.Name = constants.MCRegistrationSecret
					secret.Namespace = constants.VerrazzanoSystemNamespace
					secret.ResourceVersion = newVersion
					return nil
				})

			// Managed Cluster - expect call to get the verrazzano monitoring operator deployment.
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: deploymentName}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
					deployment.Name = deploymentName
					deployment.Namespace = constants.VerrazzanoSystemNamespace
					deployment.Spec = getTestDeploymentSpec(oldVersion)
					return nil
				})

			// update only when registration is updated
			if oldVersion != newVersion {
				// Managed Cluster - expect another call to get the verrazzano operator deployment prior to updating it
				mcMock.EXPECT().
					Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: deploymentName}, gomock.Not(gomock.Nil())).
					DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
						deployment.Name = deploymentName
						deployment.Namespace = constants.VerrazzanoSystemNamespace
						deployment.Spec = getTestDeploymentSpec(oldVersion)
						return nil
					})

				// Managed Cluster - expect call to update the deployment.
				mcMock.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, deployment *appsv1.Deployment) error {
						asserts.Equal(t, newVersion, getEnvValue(&deployment.Spec.Template.Spec.Containers, registrationSecretVersion), "expected env value for "+registrationSecretVersion)
						return nil
					})
			}

			// Make the request
			s := &Syncer{
				LocalClient: mcMock,
				Log:         ctrl.Log.WithName("test"),
				Context:     context.TODO(),
			}
			s.updateDeployment(deploymentName)

			// Validate the results
			mcMocker.Finish()
		})
	}
}

// Test_discardStatusMessages tests the discardStatusMessages function
func Test_discardStatusMessages(t *testing.T) {
	statusUpdateChan := make(chan clusters.StatusUpdateMessage, 12)
	for i := 0; i < 10; i++ {
		statusUpdateChan <- clusters.StatusUpdateMessage{}
	}
	discardStatusMessages(statusUpdateChan)

	asserts.Equal(t, 0, len(statusUpdateChan))
}

func expectAdminVMCStatusUpdateFailure(adminMock *mocks.MockClient, vmcName types.NamespacedName, adminStatusMock *mocks.MockStatusWriter, assert *asserts.Assertions) {
	adminMock.EXPECT().
		Get(gomock.Any(), vmcName, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "clusters.verrazzano.io", Resource: "VerrazzanoManagedCluster"}, vmcName.Name))
}

func expectAdminVMCStatusUpdateSuccess(adminMock *mocks.MockClient, vmcName types.NamespacedName, adminStatusMock *mocks.MockStatusWriter, assert *asserts.Assertions) {
	adminMock.EXPECT().
		Get(gomock.Any(), vmcName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, vmc *platformopclusters.VerrazzanoManagedCluster) error {
			vmc.Name = vmcName.Name
			vmc.Namespace = vmcName.Namespace
			return nil
		})
	adminMock.EXPECT().Status().Return(adminStatusMock)
	adminStatusMock.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&platformopclusters.VerrazzanoManagedCluster{})).
		DoAndReturn(func(ctx context.Context, vmc *platformopclusters.VerrazzanoManagedCluster) error {
			assert.Equal(vmcName.Namespace, vmc.Namespace)
			assert.Equal(vmcName.Name, vmc.Name)
			assert.NotNil(vmc.Status)
			assert.NotNil(vmc.Status.LastAgentConnectTime)
			assert.NotNil(vmc.Status.APIUrl)
			assert.NotNil(vmc.Status.PrometheusHost)
			return nil
		})
}

// TestSyncer_updateVMCStatus tests updateVMCStatus method
// GIVEN updateVMCStatus is called
// WHEN the status update of VMC on admin cluster succeeds
// THEN updateVMCStatus returns nil error
// GIVEN updateVMCStatus is called
// WHEN the status update of VMC on admin cluster fails
// THEN updateVMCStatus returns a non-nil error
func TestSyncer_updateVMCStatus(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)
	adminStatusMock := mocks.NewMockStatusWriter(adminMocker)
	localClientMock := mocks.NewMockClient(adminMocker)

	s := &Syncer{
		AdminClient:        adminMock,
		Log:                log,
		ManagedClusterName: "my-test-cluster",
		LocalClient:        localClientMock,
	}
	vmcName := types.NamespacedName{Name: s.ManagedClusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}

	expectGetAPIServerURLCalled(localClientMock)
	expectGetPrometheusHostCalled(localClientMock)
	// Mock the success of status updates and assert that updateVMCStatus returns nil error
	expectAdminVMCStatusUpdateSuccess(adminMock, vmcName, adminStatusMock, assert)
	assert.Nil(s.updateVMCStatus())

	// Mock the failure of status updates and assert that updateVMCStatus returns non-nil error
	expectAdminVMCStatusUpdateFailure(adminMock, vmcName, adminStatusMock, assert)
	assert.NotNil(s.updateVMCStatus())

	adminMocker.Finish()
}

func expectGetAPIServerURLCalled(mock *mocks.MockClient) {
	// Expect a call to get the console ingress and return the ingress.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *extv1beta1.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			ingress.Spec.TLS = []extv1beta1.IngressTLS{{
				Hosts: []string{"console"},
			}}
			return nil
		})
}

func expectGetPrometheusHostCalled(mock *mocks.MockClient) {
	// Expect a call to get the prometheus ingress and return the host.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzPrometheusIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *extv1beta1.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			ingress.Spec.TLS = []extv1beta1.IngressTLS{{
				Hosts: []string{"prometheus"},
			}}
			return nil
		})
}

// TestSyncer_configureLogging tests configuring logging by updating Fluentd daemonset
// GIVEN a request to configure the logging
// WHEN the cluster name in registration secret has been changed or the elasticsearch secret has been updated
// THEN ensure that Fluentd daemonset is updated
func TestSyncer_configureLogging(t *testing.T) {
	type fields struct {
		oldSecretVersion string
		newSecretVersion string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "new registration",
			fields: fields{
				oldSecretVersion: "",
				newSecretVersion: "version1",
			},
		},
		{
			name: "delete registration",
			fields: fields{
				oldSecretVersion: "version1",
				newSecretVersion: "",
			},
		},
		{
			name: "update registration",
			fields: fields{
				oldSecretVersion: "version1",
				newSecretVersion: "version2",
			},
		},
		{
			name: "no registration",
			fields: fields{
				oldSecretVersion: "",
				newSecretVersion: "",
			},
		},
		{
			name: "same registration",
			fields: fields{
				oldSecretVersion: "version1",
				newSecretVersion: "version1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVersion := tt.fields.oldSecretVersion
			newVersion := tt.fields.newSecretVersion

			// Managed cluster mocks
			mcMocker := gomock.NewController(t)
			mcMock := mocks.NewMockClient(mcMocker)

			// Managed Cluster - expect call to get the cluster registration secret.
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCRegistrationSecret}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
					secret.Name = constants.MCRegistrationSecret
					secret.Namespace = constants.VerrazzanoSystemNamespace
					secret.ResourceVersion = newVersion
					return nil
				})

			// Managed Cluster - expect call to get the verrazzano operator deployment.
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: "fluentd"}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, ds *appsv1.DaemonSet) error {
					ds.Name = "fluentd"
					ds.Namespace = constants.VerrazzanoSystemNamespace
					ds.Spec = getTestDaemonSetSpec(oldVersion)
					return nil
				})
			// update only when registration is updated
			if oldVersion != newVersion {
				mcMock.EXPECT().
					Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: "fluentd"}, gomock.Not(gomock.Nil())).
					DoAndReturn(func(ctx context.Context, name types.NamespacedName, ds *appsv1.DaemonSet) error {
						ds.Name = "fluentd"
						ds.Namespace = constants.VerrazzanoSystemNamespace
						ds.Spec = getTestDaemonSetSpec(oldVersion)
						return nil
					})
				mcMock.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, ds *appsv1.DaemonSet) error {
						asserts.Equal(t, newVersion, getEnvValue(&ds.Spec.Template.Spec.Containers, registrationSecretVersion), "expected env value for "+registrationSecretVersion)
						return nil
					})
			}

			// Make the request
			s := &Syncer{
				LocalClient: mcMock,
				Log:         ctrl.Log.WithName("test"),
				Context:     context.TODO(),
			}
			s.configureLogging()

			// Validate the results
			mcMocker.Finish()
		})
	}
}
func getTestDaemonSetSpec(secretVersion string) appsv1.DaemonSetSpec {
	return appsv1.DaemonSetSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "fluentd",
						Env: []corev1.EnvVar{
							{
								Name:  registrationSecretVersion,
								Value: secretVersion,
							},
						},
					},
				},
			},
		},
	}
}

// Test_updateEnv tests updateEnv
// GIVEN a request for a specified ENV array
// WHEN the env array contains such an env
// THEN updates its value, append the env name/value if not found
func Test_updateEnv(t *testing.T) {
	oldEnvs := []corev1.EnvVar{
		{
			Name:  "CLUSTER_NAME",
			Value: defaultClusterName,
		},
		{
			Name:  "ELASTICSEARCH_URL",
			Value: defaultElasticURL,
		},
		{
			Name:  "ELASTICSEARCH_USER",
			Value: "",
		},
		{
			Name:  "ELASTICSEARCH_PASSWORD",
			Value: "",
		},
		{
			Name:  "FLUENTD_CONF",
			Value: "fluentd.conf",
		},
	}
	newEnvs := updateEnv(constants.MCRegistrationSecret, "v2", oldEnvs)
	asserts.NotNil(t, findEnv("FLUENTD_CONF", &newEnvs))
	asserts.Equal(t, constants.MCRegistrationSecret, findEnv("CLUSTER_NAME", &newEnvs).ValueFrom.SecretKeyRef.Name)
	asserts.Equal(t, constants.MCRegistrationSecret, findEnv("ELASTICSEARCH_URL", &newEnvs).ValueFrom.SecretKeyRef.Name)
	asserts.Equal(t, constants.MCRegistrationSecret, findEnv("ELASTICSEARCH_USER", &newEnvs).ValueFrom.SecretKeyRef.Name)
	asserts.Equal(t, constants.MCRegistrationSecret, findEnv("ELASTICSEARCH_PASSWORD", &newEnvs).ValueFrom.SecretKeyRef.Name)
	//un-registration of setting secretVersion back to ""
	newEnvs = updateEnv(constants.MCRegistrationSecret, "", newEnvs)
	asserts.NotNil(t, findEnv("FLUENTD_CONF", &newEnvs))
	asserts.Equal(t, defaultClusterName, findEnv("CLUSTER_NAME", &newEnvs).Value)
	asserts.Equal(t, defaultElasticURL, findEnv("ELASTICSEARCH_URL", &newEnvs).Value)
	asserts.Equal(t, defaultSecretName, findEnv("ELASTICSEARCH_USER", &newEnvs).ValueFrom.SecretKeyRef.Name)
	asserts.Equal(t, defaultSecretName, findEnv("ELASTICSEARCH_PASSWORD", &newEnvs).ValueFrom.SecretKeyRef.Name)

}

// Test_updateVolumes tests updateVolumes
// GIVEN a request for a specified volume array
// WHEN the volume array contains such an volume
// THEN updates its value, append the volume name/value if not found
func Test_updateVolumes(t *testing.T) {
	oldVols := []corev1.Volume{
		{
			Name: "varlog",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/log",
				},
			},
		},
		{
			Name: "secret-volume",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "foo",
				},
			},
		},
		{
			Name: "my-config",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/config",
				},
			},
		},
	}
	newVols := updateVolumes(constants.MCRegistrationSecret, "v2", oldVols)
	asserts.NotNil(t, findVol("my-config", &newVols))
	asserts.NotNil(t, findVol("varlog", &newVols))
	asserts.Equal(t, constants.MCRegistrationSecret, findVol("secret-volume", &newVols).VolumeSource.Secret.SecretName)
	//un-registration of setting secretVersion back to ""
	newVols = updateVolumes(constants.MCRegistrationSecret, "", newVols)
	asserts.Equal(t, defaultSecretName, findVol("secret-volume", &newVols).VolumeSource.Secret.SecretName)
	asserts.NotNil(t, findVol("my-config", &newVols))
	asserts.NotNil(t, findVol("varlog", &newVols))
}

func findEnv(name string, envs *[]corev1.EnvVar) *corev1.EnvVar {
	for _, env := range *envs {
		if env.Name == name {
			return &env
		}
	}
	return nil
}

func findVol(name string, vols *[]corev1.Volume) *corev1.Volume {
	for _, vol := range *vols {
		if vol.Name == name {
			return &vol
		}
	}
	return nil
}
