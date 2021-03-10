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
		AdminClient:        adminMock,
		LocalClient:        mcMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
		AgentSecretFound:   true,
	}
	err := s.ProcessAgentThread()

	// Validate the results
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

func Test_getClusterNameEnvValue(t *testing.T) {
	testEnvs := []corev1.EnvVar{}
	asserts.Equal(t, "", getClusterNameEnvValue(testEnvs), "expected cluster name")
	testEnvs = []corev1.EnvVar{
		{
			Name:  managedClusterNameEnvName,
			Value: "cluster1",
		},
	}
	asserts.Equal(t, "cluster1", getClusterNameEnvValue(testEnvs), "expected cluster name")
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
	asserts.Equal(t, "cluster1", getClusterNameEnvValue(testEnvs), "expected cluster name")
}

func Test_updateClusterNameEnvValue(t *testing.T) {
	testEnvs := []corev1.EnvVar{}
	newValue := "cluster2"
	newEnvs := updateClusterNameEnvValue(testEnvs, newValue)
	asserts.Equal(t, managedClusterNameEnvName, newEnvs[0].Name, "expected env")
	asserts.Equal(t, newValue, newEnvs[0].Value, "expected env value")
	testEnvs = []corev1.EnvVar{
		{
			Name:  managedClusterNameEnvName,
			Value: "cluster1",
		},
	}
	newValue = "cluster2"
	newEnvs = updateClusterNameEnvValue(testEnvs, newValue)
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
	newEnvs = updateClusterNameEnvValue(testEnvs, newValue)
	asserts.Equal(t, managedClusterNameEnvName, newEnvs[1].Name, "expected env")
	asserts.Equal(t, newValue, newEnvs[1].Value, "expected env value")
}

func getTestDeploymentSpec(clusterName string) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Env: []corev1.EnvVar{
							{
								Name:  "MANAGED_CLUSTER_NAME",
								Value: clusterName,
							},
						},
					},
				},
			},
		},
	}
}

func TestSyncer_configureBeats(t *testing.T) {
	type fields struct {
		oldClusterName string
		newClusterName string
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldClusterName := tt.fields.oldClusterName
			newClusterName := tt.fields.newClusterName

			// Managed cluster mocks
			mcMocker := gomock.NewController(t)
			mcMock := mocks.NewMockClient(mcMocker)

			// Managed Cluster - expect call to get the verrazzano operator deployment.
			mcMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: "verrazzano-operator"}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
					deployment.Name = "verrazzano-operator"
					deployment.Namespace = constants.VerrazzanoSystemNamespace
					deployment.Spec = getTestDeploymentSpec(oldClusterName)
					return nil
				})

			// update only when registration is updated
			if oldClusterName != newClusterName {
				// Managed Cluster - expect another call to get the verrazzano operator deployment prior to updating it
				mcMock.EXPECT().
					Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: "verrazzano-operator"}, gomock.Not(gomock.Nil())).
					DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *appsv1.Deployment) error {
						deployment.Name = "verrazzano-operator"
						deployment.Namespace = constants.VerrazzanoSystemNamespace
						deployment.Spec = getTestDeploymentSpec(oldClusterName)
						return nil
					})

				// Managed Cluster - expect call to update the verrazzano operator deployment.
				mcMock.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, deployment *appsv1.Deployment) error {
						asserts.Equal(t, newClusterName, getClusterNameEnvValue(deployment.Spec.Template.Spec.Containers[0].Env), "expected env value")
						return nil
					})
			}

			// Make the request
			s := &Syncer{
				LocalClient:        mcMock,
				Log:                ctrl.Log.WithName("test"),
				ManagedClusterName: newClusterName,
				Context:            context.TODO(),
			}
			s.configureBeats()

			// Validate the results
			mcMocker.Finish()
		})
	}
}
