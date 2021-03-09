// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	corev1 "k8s.io/api/core/v1"
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
// THEN ensure that there are no calls to sync any multi-cluste resources
func TestProcessAgentThreadNoProjects(t *testing.T) {
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
			secret.ObjectMeta = validSecret.ObjectMeta
			secret.Data = validSecret.Data
			return nil
		})

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
			return (errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, name.Name))
		})

	// Do not expect any further calls because the registration secret no longer exists

	// Make the request
	s := &Syncer{
		AdminClient:         adminMock,
		LocalClient:         mcMock,
		Log:                 log,
		ManagedClusterName:  testClusterName,
		Context:             context.TODO(),
		AgentSecretFound:    true,
		AgentSecretValid:    true,
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
	testEnvs := []corev1.EnvVar{}
	asserts.Equal(t, "", getEnvValue(testEnvs, managedClusterNameEnvName), "expected cluster name")
	testEnvs = []corev1.EnvVar{
		{
			Name:  managedClusterNameEnvName,
			Value: "cluster1",
		},
	}
	asserts.Equal(t, "cluster1", getEnvValue(testEnvs, managedClusterNameEnvName), "expected cluster name")
	testEnvs = []corev1.EnvVar{
		{
			Name:  "env1",
			Value: "value1",
		},
		{
			Name:  managedClusterNameEnvName,
			Value: "cluster1",
		},
	}
	asserts.Equal(t, "cluster1", getEnvValue(testEnvs, managedClusterNameEnvName), "expected cluster name")
}

// Test_getEnvValue tests updateEnvValue
// GIVEN a request for a specified ENV name/value
// WHEN the env array contains such an env
// THEN updates its value, append the env name/value if not found
func Test_updateEnvValue(t *testing.T) {
	testEnvs := []corev1.EnvVar{}
	newValue := "cluster2"
	newEnvs := updateEnvValue(testEnvs, managedClusterNameEnvName, newValue)
	asserts.Equal(t, managedClusterNameEnvName, newEnvs[0].Name, "expected env")
	asserts.Equal(t, newValue, newEnvs[0].Value, "expected env value")
	testEnvs = []corev1.EnvVar{
		{
			Name:  managedClusterNameEnvName,
			Value: "cluster1",
		},
	}
	newValue = "cluster2"
	newEnvs = updateEnvValue(testEnvs, managedClusterNameEnvName, newValue)
	asserts.Equal(t, managedClusterNameEnvName, newEnvs[0].Name, "expected env")
	asserts.Equal(t, newValue, newEnvs[0].Value, "expected env value")
	testEnvs = []corev1.EnvVar{
		{
			Name:  "env1",
			Value: "value1",
		},
		{
			Name:  managedClusterNameEnvName,
			Value: "cluster1",
		},
	}
	newEnvs = updateEnvValue(testEnvs, managedClusterNameEnvName, newValue)
	asserts.Equal(t, managedClusterNameEnvName, newEnvs[1].Name, "expected env")
	asserts.Equal(t, newValue, newEnvs[1].Value, "expected env value")
}

func getTestDeploymentSpec(clusterName, esSecretVersion string) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Env: []corev1.EnvVar{
							{
								Name:  managedClusterNameEnvName,
								Value: clusterName,
							},
							{
								Name:  elasticsearchSecretVersionEnvName,
								Value: esSecretVersion,
							},
						},
					},
				},
			},
		},
	}
}

// TestSyncer_configureBeats tests configuring beats by updating verrazzano operator deployment
// GIVEN a request to configure the beats
// WHEN the cluster name in registration secret has been changed or the elasticsearch secret has been updated
// THEN ensure that verrazzano operator deployment is updated
func TestSyncer_configureBeats(t *testing.T) {
	type fields struct {
		oldClusterName     string
		newClusterName     string
		oldESSecretVersion string
		newESSecretVersion string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "new registration",
			fields: fields{
				oldClusterName: "",
				newClusterName: "testCluster1",
			},
		},
		{
			name: "delete registration",
			fields: fields{
				oldClusterName: "testCluster1",
				newClusterName: "",
			},
		},
		{
			name: "update registration",
			fields: fields{
				oldClusterName: "testCluster1",
				newClusterName: "testCluster2",
			},
		},
		{
			name: "no registration",
			fields: fields{
				oldClusterName: "",
				newClusterName: "",
			},
		},
		{
			name: "same registration",
			fields: fields{
				oldClusterName: "testCluster1",
				newClusterName: "testCluster1",
			},
		},
		{
			name: "update ES secret",
			fields: fields{
				oldESSecretVersion: "secret v1",
				newESSecretVersion: "secret v2",
			},
		},
		{
			name: "same ES secret",
			fields: fields{
				oldESSecretVersion: "secret v1",
				newESSecretVersion: "secret v1",
			},
		},
		{
			name: "update both",
			fields: fields{
				oldClusterName:     "testCluster1",
				newClusterName:     "testCluster1",
				oldESSecretVersion: "secret v1",
				newESSecretVersion: "secret v2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldClusterName := tt.fields.oldClusterName
			newClusterName := tt.fields.newClusterName
			oldESSecretVersion := tt.fields.oldESSecretVersion
			newESSecretVersion := tt.fields.newESSecretVersion

			// Managed cluster mocks
			mcMocker := gomock.NewController(t)
			mcMock := mocks.NewMockClient(mcMocker)

			// Managed Cluster - expect call to get the cluster registration secret.
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.MCRegistrationSecret}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
					secret.Name = constants.MCRegistrationSecret
					secret.Namespace = constants.VerrazzanoSystemNamespace
					secret.Data = map[string][]byte{constants.ClusterNameData: []byte(newClusterName)}
					return nil
				})

			// Managed Cluster - expect call to get the ES secret.
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.ElasticsearchSecretName}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
					secret.Name = constants.ElasticsearchSecretName
					secret.Namespace = constants.VerrazzanoSystemNamespace
					secret.ResourceVersion = newESSecretVersion
					return nil
				})

			// Managed Cluster - expect call to get the verrazzano operator deployment.
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: "verrazzano-operator"}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
					deployment.Name = "verrazzano-operator"
					deployment.Namespace = constants.VerrazzanoSystemNamespace
					deployment.Spec = getTestDeploymentSpec(oldClusterName, oldESSecretVersion)
					return nil
				})

			// update only when registration is updated
			if oldClusterName != newClusterName || oldESSecretVersion != newESSecretVersion {
				// Managed Cluster - expect another call to get the verrazzano operator deployment prior to updating it
				mcMock.EXPECT().
					Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: "verrazzano-operator"}, gomock.Not(gomock.Nil())).
					DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
						deployment.Name = "verrazzano-operator"
						deployment.Namespace = constants.VerrazzanoSystemNamespace
						deployment.Spec = getTestDeploymentSpec(oldClusterName, oldESSecretVersion)
						return nil
					})

				// Managed Cluster - expect call to update the verrazzano operator deployment.
				mcMock.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, deployment *appsv1.Deployment) error {
						asserts.Equal(t, newClusterName, getEnvValue(deployment.Spec.Template.Spec.Containers[0].Env, managedClusterNameEnvName), "expected env value for "+managedClusterNameEnvName)
						asserts.Equal(t, newESSecretVersion, getEnvValue(deployment.Spec.Template.Spec.Containers[0].Env, elasticsearchSecretVersionEnvName), "expected env value for "+elasticsearchSecretVersionEnvName)
						return nil
					})
			}

			// Make the request
			s := &Syncer{
				LocalClient: mcMock,
				Log:         ctrl.Log.WithName("test"),
				Context:     context.TODO(),
			}
			s.configureBeats()

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
