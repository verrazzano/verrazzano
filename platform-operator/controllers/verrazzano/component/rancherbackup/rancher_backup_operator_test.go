// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancherbackup

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = appsv1.AddToScheme(testScheme)
}

// TestisRancherBackupOperatorReady tests the isRancherBackupOperatorReady function for the Rancher Backup Operator
func TestIsRancherBackupOperatorReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Rancher Backup Operator deployment exists and there are available replicas
			// WHEN we call isRancherBackupOperatorReady
			// THEN the call returns true
			name: "Test IsReady when Rancher Backup Operator is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
						Labels:    map[string]string{"name": deploymentName},
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"name": deploymentName},
						},
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName + "-95d8c5d96-m6mbr",
						Labels: map[string]string{
							"pod-template-hash": "95d8c5d96",
							"name":              deploymentName,
						},
					},
				},
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   ComponentNamespace,
						Name:        deploymentName + "-95d8c5d96",
						Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
					},
				},
			).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Rancher Backup Operator deployment exists and there are no available replicas
			// WHEN we call isRancherBackupOperatorReady
			// THEN the call returns false
			name: "Test IsReady when Rancher Backup Operator deployment is not ready",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 0,
						Replicas:          1,
						UpdatedReplicas:   0,
					},
				}).Build(),
			expectTrue: false,
		},
		{
			// GIVEN the Rancher Backup Operator deployment does not exist
			// WHEN we call isRancherBackupOperatorReady
			// THEN the call returns false
			name:       "Test IsReady when Rancher Backup Operator deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)
			assert.Equal(t, tt.expectTrue, isRancherBackupOperatorReady(ctx))
		})
	}
}

// TestisRancherBackupOperatorReady tests the isRancherBackupOperatorReady function for the Rancher Backup Operator
func TestAppendOverrides(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(cli, &vzapi.Verrazzano{}, nil, false)

	defBOMPath := config.GetDefaultBOMFilePath()
	config.SetDefaultBomFilePath("testdata/test-bom.json")
	defer config.SetDefaultBomFilePath(defBOMPath)

	kvs := []bom.KeyValue{}
	kvs, err := AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	validateKeyValuePairs(t, kvs, "docker.io")

	privateRegistry := "foo.io"
	os.Setenv(constants.RegistryOverrideEnvVar, privateRegistry)
	defer os.Unsetenv(constants.RegistryOverrideEnvVar)

	kvs = []bom.KeyValue{}
	kvs, err = AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	validateKeyValuePairs(t, kvs, privateRegistry)
}

func validateKeyValuePairs(t *testing.T, kvs []bom.KeyValue, expectedRegistry string) {
	expectedRepo := fmt.Sprintf("%s/rancher/kubectl", expectedRegistry)
	assert.Len(t, kvs, 2)
	for _, kv := range kvs {
		switch kv.Key {
		case "global.kubectl.repository":
			assert.Equalf(t, expectedRepo, kv.Value, "Expected  %s in value, was: %s", expectedRegistry, kv.Value)
		case "global.kubectl.tag":
			continue
		default:
			assert.Fail(t, fmt.Sprintf("Unexpected key: %s", kv.Key))
		}
	}
}
