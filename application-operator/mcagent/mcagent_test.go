// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
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
	clustersapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var validSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      constants.MCAgentSecret,
		Namespace: constants.VerrazzanoSystemNamespace,
	},
	Data: map[string][]byte{constants.ClusterNameData: []byte("cluster1"), mcconstants.KubeconfigKey: []byte("kubeconfig")},
}

const testManagedPrometheusHost = "prometheus"
const testManagedThanosQueryStoreAPIHost = "thanos-query-store.example.com"

// TestReconcileAgentSecretDeleted tests agent thread when the registration secret is deleted
// GIVEN a request to process the agent loop
// WHEN the agent secret has been deleted
// THEN ensure that there are no calls to get VerrazzanoProject resources
func TestReconcileAgentSecretDeleted(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Override createAdminClient to return the mock
	originalAdminClientFunc := getAdminClientFunc
	defer func() {
		getAdminClientFunc = originalAdminClientFunc
	}()
	getAdminClientFunc = func(secret *corev1.Secret) (client.Client, error) {
		return adminMock, nil
	}

	// Managed Cluster - expect call to get the agent secret.
	expectAgentSecretNotFound(mcMock)

	// Do not expect any further calls because the agent secret no longer exists

	// Make the request
	r := &Reconciler{
		Client:       mcMock,
		Log:          log,
		Scheme:       newTestScheme(),
		AgentChannel: nil,
	}

	_, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}})

	// Validate the results
	// asserts.Equal(t, false, s.AgentSecretFound)
	// asserts.Equal(t, false, s.AgentSecretValid)

	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

func expectAgentSecretNotFound(mock *mocks.MockClient) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
			return errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, name.Name)
		})
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

// TestReconcile_AgentSecretPresenceAndValidity
// GIVEN a request to reconcile the agent
// WHEN no new VerrazzanoProjects resources exists
// THEN all the expected K8S calls are made and that there are no calls to sync any multi-cluster resources
// for various cases of agent secret presence and validity, and agent state configmap presence
func TestReconcile_AgentSecretPresenceAndValidity(t *testing.T) {
	assert := asserts.New(t)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}}
	type fields struct {
		AgentSecretFound         bool
		AgentSecretValid         bool
		AgentStateConfigMapFound bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"agent secret found not valid", fields{AgentSecretFound: true, AgentSecretValid: false, AgentStateConfigMapFound: false}, true},
		{"agent secret not found - no op expected", fields{AgentSecretFound: false, AgentSecretValid: false, AgentStateConfigMapFound: false}, false},
		{"agent secret found and valid but no configmap - should create configmap", fields{AgentSecretFound: true, AgentSecretValid: true, AgentStateConfigMapFound: false}, false},
		{"agent secret found and valid and configmap exists", fields{AgentSecretFound: true, AgentSecretValid: true, AgentStateConfigMapFound: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Managed cluster mocks
			mcMocker := gomock.NewController(t)
			mcMock := mocks.NewMockClient(mcMocker)

			// admin cluster mocks
			adminMocker := gomock.NewController(t)
			adminMock := mocks.NewMockClient(adminMocker)
			adminStatusMock := mocks.NewMockStatusWriter(adminMocker)

			// Override createAdminClient to return the mock
			originalAdminClientFunc := getAdminClientFunc
			defer func() {
				getAdminClientFunc = originalAdminClientFunc
			}()
			getAdminClientFunc = func(secret *corev1.Secret) (client.Client, error) {
				return adminMock, nil
			}

			secretToUse := validSecret
			if !tt.fields.AgentSecretValid {
				secretToUse.Data = map[string][]byte{}
			}
			if tt.fields.AgentSecretFound {
				expectAgentSecretFound(mcMock, secretToUse)
			} else {
				expectAgentSecretNotFound(mcMock)
			}
			if tt.fields.AgentSecretFound && tt.fields.AgentSecretValid {
				clusterName := string(secretToUse.Data[constants.ClusterNameData])
				if tt.fields.AgentStateConfigMapFound {
					// Managed Cluster - expect call to get the agent state config map at the beginning of reconcile
					expectAgentStateConfigMapFound(mcMock, clusterName)
				} else {
					expectAgentStateConfigMapNotFound(mcMock)
					expectAgentStateConfigMapCreated(mcMock, clusterName)
				}
				expectAllCallsNoApps(adminMock, mcMock, adminStatusMock, clusterName, assert)
			}
			r := Reconciler{
				Client: mcMock,
				Scheme: newTestScheme(),
				Log:    zap.S(),
			}

			_, err := r.Reconcile(context.TODO(), req)
			if !tt.wantErr {
				asserts.NoError(t, err)
			} else {
				asserts.NotNil(t, err)
			}
			adminMocker.Finish()
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
		Get(gomock.Any(), vmcName, gomock.Not(gomock.Nil()), gomock.Any()).
		Return(errors.NewNotFound(schema.GroupResource{Group: "clusters.verrazzano.io", Resource: "VerrazzanoManagedCluster"}, vmcName.Name))
}

func expectAdminVMCStatusUpdateSuccess(adminMock *mocks.MockClient, vmcName types.NamespacedName, adminStatusMock *mocks.MockStatusWriter, assert *asserts.Assertions) {
	expectGetVMC(adminMock, vmcName, "")
	adminMock.EXPECT().Status().Return(adminStatusMock)
	adminStatusMock.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersapi.VerrazzanoManagedCluster{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.UpdateOption) error {
			assert.Equal(vmcName.Namespace, vmc.Namespace)
			assert.Equal(vmcName.Name, vmc.Name)
			assert.NotNil(vmc.Status)
			assert.NotNil(vmc.Status.LastAgentConnectTime)
			assert.NotNil(vmc.Status.APIUrl)
			assert.Equal(testManagedPrometheusHost, vmc.Status.PrometheusHost)
			assert.Equal(testManagedThanosQueryStoreAPIHost, vmc.Status.ThanosQueryStore)
			return nil
		})
}

func expectCASyncSuccess(localMock, adminMock *mocks.MockClient, assert *asserts.Assertions, testClusterName string) {
	localRegistrationSecret := types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCRegistrationSecret}
	adminCASecret := types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: constants.VerrazzanoLocalCABundleSecret}
	adminRegSecret := types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: getRegistrationSecretName(testClusterName)}
	localIngressTLSSecret := types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VerrazzanoIngressTLSSecret}

	// Managed Cluster - expect call to get the tls-ca-additional secret. Return not found since it
	// is ok for it to be not present.
	localMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: vzconstants.RancherSystemNamespace, Name: vzconstants.AdditionalTLS}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
			return errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, name.Name)
		})
	adminMock.EXPECT().
		Get(gomock.Any(), adminCASecret, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
			secret.Name = adminCASecret.Name
			secret.Namespace = adminCASecret.Namespace
			return nil
		})
	adminMock.EXPECT().
		Get(gomock.Any(), adminRegSecret, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
			secret.Name = adminRegSecret.Name
			secret.Namespace = adminRegSecret.Namespace
			return nil
		})
	localMock.EXPECT().
		Get(gomock.Any(), localRegistrationSecret, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
			secret.Name = localRegistrationSecret.Name
			secret.Namespace = localRegistrationSecret.Namespace
			return nil
		})
	localMock.EXPECT().
		Get(gomock.Any(), localIngressTLSSecret, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
			secret.Name = localIngressTLSSecret.Name
			secret.Namespace = localIngressTLSSecret.Namespace
			secret.Data = map[string][]byte{mcconstants.CaCrtKey: []byte("somekey")}
			return nil
		})

	vmcName := types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: testClusterName}
	clusterCASecret := "clusterCASecret"
	expectGetVMC(adminMock, vmcName, clusterCASecret)
	adminClusterCASecret := types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: clusterCASecret}
	adminMock.EXPECT().
		Get(gomock.Any(), adminClusterCASecret, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
			secret.Name = adminClusterCASecret.Name
			secret.Namespace = adminClusterCASecret.Namespace
			// make the value equal to the managed cluster CA - we are not looking for updates to this secret
			secret.Data = map[string][]byte{keyCaCrtNoDot: []byte("somekey")}
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
	expectGetThanosQueryHostCalled(localClientMock)
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
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *networkingv1.Ingress, opts ...client.GetOption) error {
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
	expectGetIngress(mock, constants.VerrazzanoSystemNamespace, constants.VzPrometheusIngress, testManagedPrometheusHost)
}

func expectGetThanosQueryHostCalled(mock *mocks.MockClient) {
	// Expect a call to get the Thanos query ingress and return the host.
	expectGetIngress(mock, constants.VerrazzanoSystemNamespace, vzconstants.ThanosQueryStoreIngress, testManagedThanosQueryStoreAPIHost)
}

// Expects a call to get an ingress with the given name and namespace, and returns an ingress with the specified
// ingressHost
func expectGetIngress(mock *mocks.MockClient, ingressNamespace string, ingressName string, ingressHost string) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ingressNamespace, Name: ingressName}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ingress *networkingv1.Ingress, opts ...client.GetOption) error {
			ingress.TypeMeta = metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "ingress"}
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			ingress.Spec.Rules = []networkingv1.IngressRule{{
				Host: ingressHost,
			}}
			return nil
		})
}

func expectGetManifestSecretNotFound(mock *mocks.MockClient, clusterName string) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: getManifestSecretName(clusterName)}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
			return errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, name.Name)
		})
}

func expectListNamespacesReturnNone(mock *mocks.MockClient) {
	mock.EXPECT().
		List(gomock.Any(), &corev1.NamespaceList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *corev1.NamespaceList, opts ...client.ListOption) error {
			return nil
		})

}

func expectListProjectsReturnNone(mock *mocks.MockClient) {
	mock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...client.ListOption) error {
			return nil
		})
}

func expectAgentStateConfigMapNotFound(mock *mocks.MockClient) {
	// called once for reading the existing data, and another time during update
	mock.EXPECT().
		Get(gomock.Any(), mcAgentStateConfigMapName, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, cm *corev1.ConfigMap, opts ...client.GetOption) error {
			return errors.NewNotFound(schema.GroupResource{Group: "", Resource: "ConfigMap"}, name.Name)
		})
}

func expectAgentStateConfigMapFound(mock *mocks.MockClient, clusterName string) {
	// called at the beginning for reading the existing data, and another get to determine if update is needed
	mock.EXPECT().
		Get(gomock.Any(), mcAgentStateConfigMapName, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, cm *corev1.ConfigMap, opts ...client.GetOption) error {
			cm.Name = mcAgentStateConfigMapName.Name
			cm.Namespace = mcAgentStateConfigMapName.Namespace
			cm.Data = map[string]string{}
			if clusterName != "" {
				cm.Data[constants.ClusterNameData] = clusterName
			}
			return nil
		}).Times(2)
}

func expectAgentStateConfigMapCreated(mock *mocks.MockClient, clusterName string) {
	// expect a get where the configmap is not found
	expectAgentStateConfigMapNotFound(mock)

	// expect a create of the config map
	mock.EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
		DoAndReturn(func(ctx context.Context, cm *corev1.ConfigMap, opts ...client.GetOption) error {
			cm.Name = mcAgentStateConfigMapName.Name
			cm.Namespace = mcAgentStateConfigMapName.Namespace
			if clusterName != "" {
				cm.Data = map[string]string{constants.ClusterNameData: clusterName}
			}
			return nil
		})
}

func expectGetVMC(mock *mocks.MockClient, vmcName types.NamespacedName, caSecretName string) {
	mock.EXPECT().
		Get(gomock.Any(), vmcName, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, vmc *clustersapi.VerrazzanoManagedCluster, opts ...client.GetOption) error {
			vmc.DeletionTimestamp = nil
			vmc.Name = vmcName.Name
			vmc.Namespace = vmcName.Namespace
			if caSecretName != "" {
				vmc.Spec.CASecret = caSecretName
			}
			return nil
		})
}

func expectAgentSecretFound(mock *mocks.MockClient, secretToUse corev1.Secret) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCAgentSecret}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
			secret.ObjectMeta = secretToUse.ObjectMeta
			secret.Data = secretToUse.Data
			return nil
		})
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

// expectAllCallsNoApps expects all the calls that a reconcile of the MC agent would result in, if
// there are no applications or VerrazzanoProjects. It does not include the initial call to get the agent secret.
func expectAllCallsNoApps(adminMock *mocks.MockClient, mcMock *mocks.MockClient, adminStatusMock *mocks.MockStatusWriter, clusterName string, assert *asserts.Assertions) {
	// Admin Cluster - expect a get followed by status update on VMC to record last agent connect time
	vmcName := types.NamespacedName{Name: clusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}
	expectGetAPIServerURLCalled(mcMock)
	expectGetPrometheusHostCalled(mcMock)
	expectGetThanosQueryHostCalled(mcMock)
	expectAdminVMCStatusUpdateSuccess(adminMock, vmcName, adminStatusMock, assert)

	// Managed Cluster - expect call to get MC app config CRD - return exists
	expectGetMCAppConfigCRD(mcMock)

	// Admin Cluster - expect call to list VerrazzanoProject objects - return an empty list
	expectListProjectsReturnNone(adminMock)
	// Managed Cluster - expect call to list VerrazzanoProject objects - return an empty list
	expectListProjectsReturnNone(mcMock)

	expectServiceAndPodMonitorsList(mcMock, assert)

	// Managed Cluster - expect call to list Namespace objects - return an empty list
	expectListNamespacesReturnNone(mcMock)

	expectCASyncSuccess(mcMock, adminMock, assert, "cluster1")

	// expect the VMC to be retrieved to check for deletion
	clusterCASecret := "clusterCASecret"
	expectGetVMC(adminMock, vmcName, clusterCASecret)
	expectGetManifestSecretNotFound(adminMock, clusterName)
}

func expectGetMCAppConfigCRD(mock *mocks.MockClient) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: mcAppConfCRDName}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, crd *apiextv1.CustomResourceDefinition, opts ...client.GetOption) error {
			crd.Name = mcAppConfCRDName
			return nil
		})
}
