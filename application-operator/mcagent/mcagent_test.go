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

	// expect the VMC to be retrieved to check for deletion
	clusterCASecret := "clusterCASecret"
	adminMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: "cluster1"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, vmc *platformopclusters.VerrazzanoManagedCluster) error {
			vmc.DeletionTimestamp = nil
			vmc.Name = vmcName.Name
			vmc.Namespace = vmcName.Namespace
			vmc.Spec.CASecret = clusterCASecret
			return nil
		})

	adminMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: getManifestSecretName("cluster1")}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			return errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, name.Name)
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
