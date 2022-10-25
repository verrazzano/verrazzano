// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var validSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      constants.MCAgentSecret,
		Namespace: constants.VerrazzanoSystemNamespace,
	},
	Data: map[string][]byte{constants.ClusterNameData: []byte("cluster1"), mcconstants.KubeconfigKey: []byte("kubeconfig")},
}

// TestProcessAgentThreadNoProjects tests agent thread when no projects exist
// GIVEN a request to process the agent loop
// WHEN the a new VerrazzanoProjects resources exists
// THEN ensure that there are no calls to sync any multi-cluster resources
func TestProcessAgentThreadNoProjects(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

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

	// Managed Cluster - expect call to get the tls-ca-additional secret. Return not found since it
	// is ok for it to be not present.
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: vzconstants.RancherSystemNamespace, Name: vzconstants.AdditionalTLS}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			return errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, name.Name)
		})

	// Admin Cluster - expect a get followed by status update on VMC to record last agent connect time
	vmcName := types.NamespacedName{Name: string(validSecret.Data[constants.ClusterNameData]), Namespace: constants.VerrazzanoMultiClusterNamespace}
	expectGetAPIServerURLCalled(mcMock)
	expectGetPrometheusHostCalled(mcMock)
	expectAdminVMCStatusUpdateSuccess(adminMock, vmcName, adminStatusMock, assert)

	// Admin Cluster - expect call to list VerrazzanoProject objects - return an empty list
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...client.ListOption) error {
			return nil
		})

	expectServiceAndPodMonitorsList(mcMock, assert)

	// Managed Cluster - expect call to list VerrazzanoProject objects - return an empty list
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...client.ListOption) error {
			return nil
		})

	// Managed Cluster - expect call to list Namespace objects - return an empty list
	mcMock.EXPECT().
		List(gomock.Any(), &corev1.NamespaceList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *corev1.NamespaceList, opts ...client.ListOption) error {
			return nil
		})

	expectCASyncSuccess(mcMock, adminMock, assert, "cluster1")

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

func expectServiceAndPodMonitorsList(mock *mocks.MockClient, assert *asserts.Assertions) {
	// Managed Cluster, expect call to list service monitors, return an empty list
	mock.EXPECT().
		List(gomock.Any(), &v1.ServiceMonitorList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.ServiceMonitorList, opts ...client.ListOption) error {
			return nil
		})
	// Managed Cluster, expect call to list pod monitors, return an empty list
	mock.EXPECT().
		List(gomock.Any(), &v1.PodMonitorList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PodMonitorList, opts ...client.ListOption) error {
			return nil
		})

}

// TestProcessAgentThreadSecretDeleted tests agent thread when the registration secret is deleted
// GIVEN a request to process the agent loop
// WHEN the registration secret has been deleted
// THEN ensure that there are no calls to get VerrazzanoProject resources
func TestProcessAgentThreadSecretDeleted(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

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
	invalidSecret.Data = map[string][]byte{mcconstants.KubeconfigKey: []byte("kubeconfig")}
	err = validateAgentSecret(&invalidSecret)
	assert.Error(err)
	assert.Contains(err.Error(), fmt.Sprintf("missing the required field %s", constants.ClusterNameData))

	// A secret without a kubeconfig
	invalidSecret.Data = map[string][]byte{constants.ClusterNameData: []byte("cluster1")}
	err = validateAgentSecret(&invalidSecret)
	assert.Error(err)
	assert.Contains(err.Error(), fmt.Sprintf("missing the required field %s", mcconstants.KubeconfigKey))
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

			// Managed Cluster - expect call to get the Verrazzano monitoring operator deployment.
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
				// Managed Cluster - expect another call to get the Verrazzano operator deployment prior to updating it
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
					Update(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, deployment *appsv1.Deployment, opts ...client.UpdateOption) error {
						asserts.Equal(t, newVersion, getEnvValue(&deployment.Spec.Template.Spec.Containers, registrationSecretVersion), "expected env value for "+registrationSecretVersion)
						return nil
					})
			}

			// Make the request
			s := &Syncer{
				LocalClient: mcMock,
				Log:         zap.S().With("test"),
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
		Update(gomock.Any(), gomock.AssignableToTypeOf(&platformopclusters.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *platformopclusters.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			assert.Equal(vmcName.Namespace, vmc.Namespace)
			assert.Equal(vmcName.Name, vmc.Name)
			assert.NotNil(vmc.Status)
			assert.NotNil(vmc.Status.LastAgentConnectTime)
			assert.NotNil(vmc.Status.APIUrl)
			assert.NotNil(vmc.Status.PrometheusHost)
			return nil
		})
}

func expectCASyncSuccess(localMock, adminMock *mocks.MockClient, assert *asserts.Assertions, testClusterName string) {
	localRegistrationSecret := types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCRegistrationSecret}
	adminCASecret := types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: constants.VerrazzanoLocalCABundleSecret}
	adminRegSecret := types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: getRegistrationSecretName(testClusterName)}
	localIngressTLSSecret := types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VerrazzanoIngressTLSSecret}
	adminMock.EXPECT().
		Get(gomock.Any(), adminCASecret, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Name = adminCASecret.Name
			secret.Namespace = adminCASecret.Namespace
			return nil
		})
	adminMock.EXPECT().
		Get(gomock.Any(), adminRegSecret, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Name = adminRegSecret.Name
			secret.Namespace = adminRegSecret.Namespace
			return nil
		})
	localMock.EXPECT().
		Get(gomock.Any(), localRegistrationSecret, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Name = localRegistrationSecret.Name
			secret.Namespace = localRegistrationSecret.Namespace
			return nil
		})
	localMock.EXPECT().
		Get(gomock.Any(), localIngressTLSSecret, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Name = localIngressTLSSecret.Name
			secret.Namespace = localIngressTLSSecret.Namespace
			return nil
		})

	vmcName := types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: testClusterName}
	clusterCASecret := "clusterCASecret"
	adminMock.EXPECT().
		Get(gomock.Any(), vmcName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, vmc *platformopclusters.VerrazzanoManagedCluster) error {
			vmc.Name = vmcName.Name
			vmc.Namespace = vmcName.Namespace
			vmc.Spec.CASecret = clusterCASecret
			return nil
		})
	adminClusterCASecret := types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: clusterCASecret}
	adminMock.EXPECT().
		Get(gomock.Any(), adminClusterCASecret, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Name = adminClusterCASecret.Name
			secret.Namespace = adminClusterCASecret.Namespace
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
	log := zap.S().With("test")

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
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *networkingv1.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			ingress.Spec.Rules = []networkingv1.IngressRule{{
				Host: "console",
			}}
			return nil
		})
}

func expectGetPrometheusHostCalled(mock *mocks.MockClient) {
	// Expect a call to get the prometheus ingress and return the host.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzPrometheusIngress}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *networkingv1.Ingress) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			ingress.Spec.Rules = []networkingv1.IngressRule{{
				Host: "prometheus",
			}}
			return nil
		})
}

// TestSyncer_configureLogging tests configuring logging by updating Fluentd daemonset
// GIVEN a request to configure the logging
// WHEN the registration secret data doesn't match the daemonset,
// THEN ensure that Fluentd daemonset is updated
func TestSyncer_configureLogging(t *testing.T) {
	type fields struct {
		externalEs     bool
		secretExists   bool
		dsClusterName  string
		dsEsURL        string
		dsSecretName   string
		forceDSRestart bool
		expectUpdateDS bool
	}
	const externalEsURL = "externalEsURL"
	const externalEsSecret = "externalEsSecret"
	const regSecretEsURL = "secretEsURL"
	const regSecretClusterName = "secretClusterName"
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "new registration",
			fields: fields{
				secretExists:   true,
				dsClusterName:  "",
				dsEsURL:        "",
				dsSecretName:   "",
				forceDSRestart: false,
				expectUpdateDS: true,
			},
		},
		{
			name: "delete registration",
			fields: fields{
				secretExists:   false,
				dsClusterName:  regSecretClusterName,
				dsEsURL:        regSecretEsURL,
				dsSecretName:   constants.MCRegistrationSecret,
				forceDSRestart: false,
				expectUpdateDS: true,
			},
		},
		{
			name: "update registration daemonset cluster name changed",
			fields: fields{
				secretExists:   true,
				dsClusterName:  "differentClusterName",
				dsEsURL:        regSecretEsURL,
				dsSecretName:   constants.MCRegistrationSecret,
				forceDSRestart: false,
				expectUpdateDS: true,
			},
		},
		{
			name: "update registration daemonset ES URL changed",
			fields: fields{
				secretExists:   true,
				dsClusterName:  regSecretClusterName,
				dsEsURL:        "differentEsURL",
				dsSecretName:   constants.MCRegistrationSecret,
				forceDSRestart: false,
				expectUpdateDS: true,
			},
		},
		{
			name: "update registration daemonset secret name changed",
			fields: fields{
				secretExists:   true,
				dsClusterName:  regSecretClusterName,
				dsEsURL:        regSecretEsURL,
				dsSecretName:   "differentSecret",
				forceDSRestart: false,
				expectUpdateDS: true,
			},
		},
		{
			name: "no registration",
			fields: fields{
				secretExists:   false,
				dsClusterName:  defaultClusterName,
				dsEsURL:        vzconstants.DefaultOpensearchURL,
				dsSecretName:   defaultSecretName,
				forceDSRestart: false,
				expectUpdateDS: false,
			},
		},
		{
			name: "same registration",
			fields: fields{
				secretExists:   true,
				dsClusterName:  regSecretClusterName,
				dsEsURL:        regSecretEsURL,
				dsSecretName:   constants.MCRegistrationSecret,
				forceDSRestart: false,
				expectUpdateDS: false,
			},
		},
		{
			name: "same registration force DS restart",
			fields: fields{
				secretExists:   true,
				dsClusterName:  regSecretClusterName,
				dsEsURL:        regSecretEsURL,
				dsSecretName:   constants.MCRegistrationSecret,
				forceDSRestart: true,
				expectUpdateDS: true,
			},
		},
		{
			name: "new registration external ES",
			fields: fields{
				externalEs:     true,
				secretExists:   true,
				dsClusterName:  "",
				dsEsURL:        "",
				dsSecretName:   "",
				forceDSRestart: false,
				expectUpdateDS: true,
			},
		},
		{
			name: "no registration external ES",
			fields: fields{
				externalEs:     true,
				secretExists:   false,
				dsClusterName:  defaultClusterName,
				dsEsURL:        externalEsURL,
				dsSecretName:   externalEsSecret,
				forceDSRestart: false,
				expectUpdateDS: false,
			},
		},
		{
			name: "same registration external ES",
			fields: fields{
				externalEs:     true,
				secretExists:   true,
				dsClusterName:  regSecretClusterName,
				dsEsURL:        regSecretEsURL,
				dsSecretName:   constants.MCRegistrationSecret,
				forceDSRestart: false,
				expectUpdateDS: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsClusterName := tt.fields.dsClusterName
			dsEsURL := tt.fields.dsEsURL
			dsSecretName := tt.fields.dsSecretName
			expectUpdateDS := tt.fields.expectUpdateDS
			secretExists := tt.fields.secretExists

			// Managed cluster mocks
			mcMocker := gomock.NewController(t)
			mcMock := mocks.NewMockClient(mcMocker)

			// Managed Cluster - expect call to get the cluster registration secret.
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCRegistrationSecret}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
					if secretExists {
						secret.Name = constants.MCRegistrationSecret
						secret.Namespace = constants.VerrazzanoSystemNamespace
						secret.ResourceVersion = "secretVersion"
						secret.Data = map[string][]byte{}
						secret.Data[constants.ClusterNameData] = []byte(regSecretClusterName)
						secret.Data[constants.OpensearchURLData] = []byte(regSecretEsURL)
						return nil
					}
					return errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, constants.MCRegistrationSecret)
				})

			// Managed Cluster - expect call to get fluentd-es-config configmap
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: esConfigMapName}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, cm *corev1.ConfigMap) error {
					cm.Name = esConfigMapName
					cm.Namespace = constants.VerrazzanoSystemNamespace
					var data = make(map[string]string)
					data[esConfigMapURLKey] = dsEsURL
					data[esConfigMapSecretKey] = dsSecretName
					cm.Data = data
					return nil
				})

			// Managed Cluster - expect call to get the fluentd daemonset.
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: vzconstants.FluentdDaemonSetName}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, ds *appsv1.DaemonSet) error {
					ds.Name = vzconstants.FluentdDaemonSetName
					ds.Namespace = constants.VerrazzanoSystemNamespace
					ds.Spec = getTestDaemonSetSpec(dsClusterName, dsEsURL, dsSecretName)
					return nil
				})

			// we always call controllerutil.CreateOrUpdate in mcagent_test, which will do another get for fluentd
			// daemonset. However, update will only be called if we changed the daemonset
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: vzconstants.FluentdDaemonSetName}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, ds *appsv1.DaemonSet) error {
					ds.Name = vzconstants.FluentdDaemonSetName
					ds.Namespace = constants.VerrazzanoSystemNamespace
					ds.Spec = getTestDaemonSetSpec(dsClusterName, dsEsURL, dsSecretName)
					return nil
				})
			// update only when expected
			if expectUpdateDS {
				mcMock.EXPECT().
					Update(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, ds *appsv1.DaemonSet, opts ...client.UpdateOption) error {
						// for default secret (verrazzano-es-internal), the secret-volume should be optional, since it may
						// not exist on managed clusters.
						if dsSecretName == defaultSecretName {
							asserts.Equal(t, true, ds.Spec.Template.Spec.Volumes[0].Secret.Optional)
						}
						return nil
					})
			}

			// Make the request
			s := &Syncer{
				LocalClient: mcMock,
				Log:         zap.S().With("test"),
				Context:     context.TODO(),
			}
			s.configureLogging(tt.fields.forceDSRestart)

			// Validate the results
			mcMocker.Finish()
		})
	}
}
func getTestDaemonSetSpec(clusterName, esURL, secretName string) appsv1.DaemonSetSpec {
	var usernameKey string
	var passwordKey string
	if secretName == constants.MCRegistrationSecret {
		usernameKey = constants.OpensearchUsernameData
		passwordKey = constants.OpensearchPasswordData
	} else {
		usernameKey = constants.VerrazzanoUsernameData
		passwordKey = constants.VerrazzanoPasswordData
	}
	volumeIsOptional := secretName == defaultSecretName
	return appsv1.DaemonSetSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "fluentd",
						Env: []corev1.EnvVar{
							{
								Name:  constants.FluentdClusterNameEnvVar,
								Value: clusterName,
							},
							{
								Name:  constants.FluentdOpensearchURLEnvVar,
								Value: esURL,
							},
							{
								Name: constants.FluentdOpensearchUserEnvVar,
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: secretName,
										},
										Key: usernameKey,
										Optional: func(opt bool) *bool {
											return &opt
										}(true),
									},
								},
							},
							{
								Name: constants.FluentdOpensearchPwdEnvVar,
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: secretName,
										},
										Key: passwordKey,
										Optional: func(opt bool) *bool {
											return &opt
										}(true),
									},
								},
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "secret-volume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: secretName,
								Optional:   &volumeIsOptional,
							},
						},
					},
				},
			},
		},
	}
}

// Test_updateLoggingDaemonsetEnv tests updateLoggingDaemonsetEnv
// GIVEN a request for a specified ENV array
// WHEN the env array contains such an env
// THEN updates its value, append the env name/value if not found
func Test_updateLoggingDaemonsetEnv(t *testing.T) {
	oldEnvs := []corev1.EnvVar{
		{
			Name:  "CLUSTER_NAME",
			Value: defaultClusterName,
		},
		{
			Name:  "OPENSEARCH_URL",
			Value: vzconstants.DefaultOpensearchURL,
		},
		{
			Name:  "OPENSEARCH_USER",
			Value: "",
		},
		{
			Name:  "OPENSEARCH_PASSWORD",
			Value: "",
		},
		{
			Name:  "FLUENTD_CONF",
			Value: "fluentd.conf",
		},
	}
	const newClusterName = "newManagedClusterName"
	const newOpenURL = "https://myNewOpenURL"
	regSecret := corev1.Secret{
		Data: map[string][]byte{
			constants.ClusterNameData:        []byte(newClusterName),
			constants.OpensearchURLData:      []byte(newOpenURL),
			constants.OpensearchUsernameData: []byte("someuser"),
			constants.OpensearchPasswordData: []byte("somepassword"),
		},
	}
	newEnvs := updateLoggingDaemonsetEnv(regSecret, true, vzconstants.DefaultOpensearchURL, defaultSecretName, oldEnvs)
	asserts.NotNil(t, findEnv("FLUENTD_CONF", &newEnvs))
	asserts.Equal(t, newClusterName, findEnv("CLUSTER_NAME", &newEnvs).Value)
	asserts.Equal(t, newOpenURL, findEnv("OPENSEARCH_URL", &newEnvs).Value)
	asserts.Equal(t, constants.MCRegistrationSecret, findEnv("OPENSEARCH_USER", &newEnvs).ValueFrom.SecretKeyRef.Name)
	asserts.Equal(t, constants.MCRegistrationSecret, findEnv("OPENSEARCH_PASSWORD", &newEnvs).ValueFrom.SecretKeyRef.Name)
	// un-registration of setting secretVersion back to ""
	newEnvs = updateLoggingDaemonsetEnv(regSecret, false, vzconstants.DefaultOpensearchURL, defaultSecretName, newEnvs)
	asserts.NotNil(t, findEnv("FLUENTD_CONF", &newEnvs))
	asserts.Equal(t, defaultClusterName, findEnv("CLUSTER_NAME", &newEnvs).Value)
	asserts.Equal(t, vzconstants.DefaultOpensearchURL, findEnv("OPENSEARCH_URL", &newEnvs).Value)
	asserts.Equal(t, defaultSecretName, findEnv("OPENSEARCH_USER", &newEnvs).ValueFrom.SecretKeyRef.Name)
	asserts.Equal(t, defaultSecretName, findEnv("OPENSEARCH_PASSWORD", &newEnvs).ValueFrom.SecretKeyRef.Name)

}

// Test_updateLoggingDaemonsetVolumes tests updateLoggingDaemonsetVolumes
// GIVEN a request for a specified volume array
// WHEN the volume array contains such an volume
// THEN updates its value, append the volume name/value if not found
func Test_updateLoggingDaemonsetVolumes(t *testing.T) {
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
	newVols := updateLoggingDaemonsetVolumes(true, defaultSecretName, oldVols)
	asserts.NotNil(t, findVol("my-config", &newVols))
	asserts.NotNil(t, findVol("varlog", &newVols))
	asserts.Equal(t, constants.MCRegistrationSecret, findVol("secret-volume", &newVols).VolumeSource.Secret.SecretName)
	// un-registration of setting secretVersion back to ""
	newVols = updateLoggingDaemonsetVolumes(false, defaultSecretName, newVols)
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
