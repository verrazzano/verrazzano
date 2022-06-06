// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	vmoconst "github.com/verrazzano/verrazzano-monitoring-operator/pkg/constants"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testBomFilePath = "../../../testdata/test_bom.json"
)

var (
	testScheme = runtime.NewScheme()

	falseValue = false
	trueValue  = true
)

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

// TestIsPrometheusOperatorReady tests the isPrometheusOperatorReady function for the Prometheus Operator
func TestIsPrometheusOperatorReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Prometheus Operator deployment exists and there are available replicas
			// WHEN we call isPrometheusOperatorReady
			// THEN the call returns true
			name: "Test IsReady when Prometheus Operator is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				}).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Prometheus Operator deployment exists and there are no available replicas
			// WHEN we call isPrometheusOperatorReady
			// THEN the call returns false
			name: "Test IsReady when Prometheus Operator deployment is not ready",
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
			// GIVEN the Prometheus Operator deployment does not exist
			// WHEN we call isPrometheusOperatorReady
			// THEN the call returns false
			name:       "Test IsReady when Prometheus Operator deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, false)
			assert.Equal(t, tt.expectTrue, isPrometheusOperatorReady(ctx))
		})
	}
}

// TestAppendOverrides tests that helm overrides are set properly
func TestAppendOverrides(t *testing.T) {
	oldBomPath := config.GetDefaultBOMFilePath()
	config.SetDefaultBomFilePath(testBomFilePath)
	defer config.SetDefaultBomFilePath(oldBomPath)

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	kvs := make([]bom.KeyValue, 0)

	// GIVEN a Verrazzano CR with the CertManager component enabled
	// WHEN the AppendOverrides function is called
	// THEN the key/value slice contains the expected helm override keys and values
	// AND the admission webhook cert manager helm override is set to true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Enabled: &trueValue,
				},
			},
		},
	}

	ctx := spi.NewFakeContext(client, vz, false)

	var err error
	kvs, err = AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 22)

	assert.Equal(t, "ghcr.io/verrazzano/prometheus-config-reloader", bom.FindKV(kvs, "prometheusOperator.prometheusConfigReloader.image.repository"))
	assert.NotEmpty(t, bom.FindKV(kvs, "prometheusOperator.prometheusConfigReloader.image.tag"))

	assert.Equal(t, "ghcr.io/verrazzano/alertmanager", bom.FindKV(kvs, "alertmanager.alertmanagerSpec.image.repository"))
	assert.NotEmpty(t, bom.FindKV(kvs, "alertmanager.alertmanagerSpec.image.tag"))

	assert.True(t, strings.HasPrefix(bom.FindKV(kvs, "prometheusOperator.alertmanagerDefaultBaseImage"), "ghcr.io/verrazzano/alertmanager:"))
	assert.True(t, strings.HasPrefix(bom.FindKV(kvs, "prometheusOperator.prometheusDefaultBaseImage"), "ghcr.io/verrazzano/prometheus:"))

	assert.Equal(t, "true", bom.FindKV(kvs, "prometheusOperator.admissionWebhooks.certManager.enabled"))

	// GIVEN a Verrazzano CR with the CertManager component disabled
	// WHEN the AppendOverrides function is called
	// THEN the key/value slice contains the expected helm override keys and values
	// AND the admission webhook cert manager helm override is set to false
	vz = &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Enabled: &falseValue,
				},
			},
		},
	}

	ctx = spi.NewFakeContext(client, vz, false)
	kvs = make([]bom.KeyValue, 0)

	kvs, err = AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 22)

	assert.Equal(t, "false", bom.FindKV(kvs, "prometheusOperator.admissionWebhooks.certManager.enabled"))
}

// TestPreInstallUpgrade tests the preInstallUpgrade function.
func TestPreInstallUpgrade(t *testing.T) {
	// GIVEN the Prometheus Operator is being installed or upgraded
	// WHEN the preInstallUpgrade function is called
	// THEN the component namespace is created in the cluster
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)

	err := preInstallUpgrade(ctx)
	assert.NoError(t, err)

	ns := v1.Namespace{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: ComponentNamespace}, &ns)
	assert.NoError(t, err)
}

// TestPreUpgrade tests the preUpgrade function.
func TestPostInstallUpgrade(t *testing.T) {
	// GIVEN the Prometheus Operator is being installed or upgraded
	// WHEN the postInstallUpgrade function is called
	// THEN the function does not return an error
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)

	err := postInstallUpgrade(ctx)
	assert.NoError(t, err)
}

// TestAppendIstioOverrides tests that the Istio overrides get applied
func TestAppendIstioOverrides(t *testing.T) {
	annotationKey := "annKey"
	volumeMountKey := "vmKey"
	volumeKey := "volKey"
	tests := []struct {
		name            string
		expectOverrides []bom.KeyValue
	}{
		{
			name: "test expect overrides",
			expectOverrides: []bom.KeyValue{
				{
					Key:   fmt.Sprintf(`%s.traffic\.sidecar\.istio\.io/includeOutboundIPRanges`, annotationKey),
					Value: "0.0.0.0/32",
				},
				{
					Key:   fmt.Sprintf(`%s.proxy\.istio\.io/config`, annotationKey),
					Value: `{"proxyMetadata":{ "OUTPUT_CERTS": "/etc/istio-output-certs"}}`,
				},
				{
					Key:   fmt.Sprintf(`%s.sidecar\.istio\.io/userVolumeMount`, annotationKey),
					Value: `[{"name": "istio-certs-dir", "mountPath": "/etc/istio-output-certs"}]`,
				},
				{
					Key:   fmt.Sprintf("%s[0].name", volumeMountKey),
					Value: istioVolumeName,
				},
				{
					Key:   fmt.Sprintf("%s[0].mountPath", volumeMountKey),
					Value: vmoconst.IstioCertsMountPath,
				},
				{
					Key:   fmt.Sprintf("%s[0].name", volumeKey),
					Value: istioVolumeName,
				},
				{
					Key:   fmt.Sprintf("%s[0].emptyDir.medium", volumeKey),
					Value: string(v1.StorageMediumMemory),
				},
			},
		},
	}
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, name types.NamespacedName, service *v1.Service) error {
			service.Spec.ClusterIP = "0.0.0.0"
			return nil
		})
	vz := vzapi.Verrazzano{}
	ctx := spi.NewFakeContext(mock, &vz, false)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kvs, err := appendIstioOverrides(ctx, annotationKey, volumeMountKey, volumeKey, []bom.KeyValue{})

			assert.Equal(t, len(tt.expectOverrides), len(kvs))

			for _, kvsVal := range kvs {
				found := false
				for _, expVal := range tt.expectOverrides {
					if expVal == kvsVal {
						found = true
						break
					}
				}
				assert.True(t, found, fmt.Sprintf("Could not find key %s, value %s in expected key value pairs", kvsVal.Key, kvsVal.Value))
			}
			assert.NoError(t, err)
		})
	}
}

// TestValidatePrometheusOperator tests the validation of the Prometheus Operator installation and the Verrazzano CR
func TestValidatePrometheusOperator(t *testing.T) {
	tests := []struct {
		name        string
		vz          vzapi.Verrazzano
		expectError bool
	}{
		{
			name:        "test nothing enabled",
			vz:          vzapi.Verrazzano{},
			expectError: false,
		},
		{
			name: "test only Prometheus enabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Prometheus:         &vzapi.PrometheusComponent{Enabled: &trueValue},
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &falseValue},
					},
				},
			},
			expectError: true,
		},
		{
			name: "test only Prometheus Operator enabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Prometheus:         &vzapi.PrometheusComponent{Enabled: &falseValue},
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &trueValue},
					},
				},
			},
			expectError: false,
		},
		{
			name: "test all enabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Prometheus:         &vzapi.PrometheusComponent{Enabled: &trueValue},
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &trueValue},
					},
				},
			},
			expectError: false,
		},
		{
			name: "test all disabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Prometheus:         &vzapi.PrometheusComponent{Enabled: &falseValue},
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &falseValue},
					},
				},
			},
			expectError: false,
		},
	}
	c := prometheusComponent{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.validatePrometheusOperator(&tt.vz)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// erroringFakeClient wraps a k8s client and returns an error when Update is called
type erroringFakeClient struct {
	client.Client
}

// Update always returns an error - used to simulate an error updating a resource
func (e *erroringFakeClient) Update(_ context.Context, _ client.Object, _ ...client.UpdateOption) error {
	return errors.NewConflict(schema.GroupResource{}, "", nil)
}

// TestRemoveOldClaimFromPrometheusVolume tests the removeOldClaimFromPrometheusVolume function
func TestRemoveOldClaimFromPrometheusVolume(t *testing.T) {
	const volumeName = "pvc-5ab58a05-71f9-4f09-8911-a5c029f6305f"

	// GIVEN a persistent volume that has a released status and a claim that references vmi-system-prometheus
	// WHEN the removeOldClaimFromPrometheusVolume function is called
	// THEN the persistent volume is updated and the claim is removed
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel: constants.PrometheusStorageLabelValue,
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				ClaimRef: &corev1.ObjectReference{
					Name:      constants.VMISystemPrometheusVolumeClaim,
					Namespace: constants.VerrazzanoSystemNamespace,
				},
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeReleased,
			},
		}).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)

	err := removeOldClaimFromPrometheusVolume(ctx)
	assert.NoError(t, err)

	// validate that the ClaimRef is now nil
	pv := &corev1.PersistentVolume{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: volumeName}, pv)
	assert.NoError(t, err)
	assert.Nil(t, pv.Spec.ClaimRef)

	// GIVEN no persistent volumes
	// WHEN the removeOldClaimFromPrometheusVolume function is called
	// THEN no error is returned
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
			},
		}).Build()
	ctx = spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)

	err = removeOldClaimFromPrometheusVolume(ctx)
	assert.NoError(t, err)

	// GIVEN a persistent volume that is bound and has a claim that references vmi-system-prometheus
	// WHEN the removeOldClaimFromPrometheusVolume function is called
	// THEN the persistent volume is not updated
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel: constants.PrometheusStorageLabelValue,
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				ClaimRef: &corev1.ObjectReference{
					Name:      constants.VMISystemPrometheusVolumeClaim,
					Namespace: constants.VerrazzanoSystemNamespace,
				},
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeBound,
			},
		}).Build()
	ctx = spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)

	err = removeOldClaimFromPrometheusVolume(ctx)
	assert.NoError(t, err)

	// validate that the ClaimRef is not nil
	pv = &corev1.PersistentVolume{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: volumeName}, pv)
	assert.NoError(t, err)
	assert.NotNil(t, pv.Spec.ClaimRef)

	// GIVEN a persistent volume that has a released status and a claim that references vmi-system-prometheus
	// WHEN the removeOldClaimFromPrometheusVolume function is called and the call to update the volume fails
	// THEN an error is returned
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel: constants.PrometheusStorageLabelValue,
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				ClaimRef: &corev1.ObjectReference{
					Name:      constants.VMISystemPrometheusVolumeClaim,
					Namespace: constants.VerrazzanoSystemNamespace,
				},
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeReleased,
			},
		}).Build()
	erroringClient := &erroringFakeClient{Client: client}
	ctx = spi.NewFakeContext(erroringClient, &vzapi.Verrazzano{}, false)

	// validate that the expected error is returned
	err = removeOldClaimFromPrometheusVolume(ctx)
	assert.ErrorContains(t, err, "Failed removing claim")
}

// TestResetVolumeReclaimPolicy tests the resetVolumeReclaimPolicy function
func TestResetVolumeReclaimPolicy(t *testing.T) {
	const volumeName = "pvc-5ab58a05-71f9-4f09-8911-a5c029f6305f"

	// GIVEN a persistent volume that has a bound status
	// WHEN the resetVolumeReclaimPolicy function is called
	// THEN the persistent volume reclaim policy is reset to the original value
	// AND the old-reclaim-policy label is removed
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel:       constants.PrometheusStorageLabelValue,
					constants.OldReclaimPolicyLabel: string(corev1.PersistentVolumeReclaimDelete),
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeBound,
			},
		}).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)

	err := resetVolumeReclaimPolicy(ctx)
	assert.NoError(t, err)

	// validate that the reclaim policy is now "Delete" and the label has been removed
	pv := &corev1.PersistentVolume{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: volumeName}, pv)
	assert.NoError(t, err)
	assert.Equal(t, corev1.PersistentVolumeReclaimDelete, pv.Spec.PersistentVolumeReclaimPolicy)
	assert.NotContains(t, pv.Labels, constants.OldReclaimPolicyLabel)

	// GIVEN a persistent volume that has an available status
	// WHEN the resetVolumeReclaimPolicy function is called
	// THEN the persistent volume is not updated
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel:       constants.PrometheusStorageLabelValue,
					constants.OldReclaimPolicyLabel: string(corev1.PersistentVolumeReclaimDelete),
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeAvailable,
			},
		}).Build()
	ctx = spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)

	err = resetVolumeReclaimPolicy(ctx)
	assert.NoError(t, err)

	// validate that the reclaim policy has not changed and that the label still exists
	pv = &corev1.PersistentVolume{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: volumeName}, pv)
	assert.NoError(t, err)
	assert.Equal(t, corev1.PersistentVolumeReclaimRetain, pv.Spec.PersistentVolumeReclaimPolicy)
	assert.Contains(t, pv.Labels, constants.OldReclaimPolicyLabel)

	// GIVEN a persistent volume that has a bound status
	// WHEN the resetVolumeReclaimPolicy function is called and the call to update the volume fails
	// THEN an error is returned
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel:       constants.PrometheusStorageLabelValue,
					constants.OldReclaimPolicyLabel: string(corev1.PersistentVolumeReclaimDelete),
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeBound,
			},
		}).Build()
	erroringClient := &erroringFakeClient{Client: client}
	ctx = spi.NewFakeContext(erroringClient, &vzapi.Verrazzano{}, false)

	// validate that the expected error is returned
	err = resetVolumeReclaimPolicy(ctx)
	assert.ErrorContains(t, err, "Failed resetting reclaim policy")
}

// TestAppendResourceRequestOverrides tests the appendResourceRequestOverrides function
func TestAppendResourceRequestOverrides(t *testing.T) {
	const (
		storageSize = "1Gi"
		memorySize  = "128Mi"
	)
	clientNoPV := fake.NewClientBuilder().WithScheme(testScheme).Build()
	clientWithPV := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pvc-5ab58a05-71f9-4f09-8911-a5c029f6305f",
				Labels: map[string]string{
					constants.StorageForLabel: constants.PrometheusStorageLabelValue,
				},
			},
		}).Build()

	tests := []struct {
		name            string
		client          client.Client
		request         common.ResourceRequestValues
		expectOverrides []bom.KeyValue
		expectError     bool
	}{
		{
			// GIVEN a resource request with both storage and memory set and there are no existing Prometheus persistent volumes
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and the expected key/value overrides are returned
			name:   "both storage and memory set, no existing Prometheus persistent volume",
			client: clientNoPV,
			request: common.ResourceRequestValues{
				Storage: storageSize,
				Memory:  memorySize,
			},
			expectOverrides: []bom.KeyValue{
				{
					Key:   "prometheus.prometheusSpec.storageSpec.disableMountSubPath",
					Value: "true",
				},
				{
					Key:   "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage",
					Value: storageSize,
				},
				{
					Key:   "prometheus.prometheusSpec.resources.requests.memory",
					Value: memorySize,
				},
			},
			expectError: false,
		},
		{
			// GIVEN a resource request with both storage and memory set and there is an existing Prometheus persistent volume
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and the expected key/value overrides are returned
			name:   "both storage and memory set, and existing Prometheus persistent volume",
			client: clientWithPV,
			request: common.ResourceRequestValues{
				Storage: storageSize,
				Memory:  memorySize,
			},
			expectOverrides: []bom.KeyValue{
				{
					Key:   "prometheus.prometheusSpec.storageSpec.disableMountSubPath",
					Value: "true",
				},
				{
					Key:   "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage",
					Value: storageSize,
				},
				{
					Key:   `prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.selector.matchLabels.verrazzano\.io/storage-for`,
					Value: "prometheus",
				},
				{
					Key:   "prometheus.prometheusSpec.resources.requests.memory",
					Value: memorySize,
				},
			},
			expectError: false,
		},
		{
			// GIVEN a resource request with no storage or memory requests
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and no key/value overrides are returned
			name:            "neither storage nor memory set",
			client:          clientNoPV,
			request:         common.ResourceRequestValues{},
			expectOverrides: []bom.KeyValue{},
			expectError:     false,
		},
		{
			// GIVEN a resource request with only storage set and there are no existing Prometheus persistent volumes
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and the expected key/value overrides are returned
			name:   "only storage set, no persistent volumes",
			client: clientNoPV,
			request: common.ResourceRequestValues{
				Storage: storageSize,
			},
			expectOverrides: []bom.KeyValue{
				{
					Key:   "prometheus.prometheusSpec.storageSpec.disableMountSubPath",
					Value: "true",
				},
				{
					Key:   "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage",
					Value: storageSize,
				},
			},
			expectError: false,
		},
		{
			// GIVEN a resource request with only storage set and there are is an existing Prometheus persistent volume
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and the expected key/value overrides are returned
			name:   "only storage set, and persistent volume exists",
			client: clientWithPV,
			request: common.ResourceRequestValues{
				Storage: storageSize,
			},
			expectOverrides: []bom.KeyValue{
				{
					Key:   "prometheus.prometheusSpec.storageSpec.disableMountSubPath",
					Value: "true",
				},
				{
					Key:   "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage",
					Value: storageSize,
				},
				{
					Key:   `prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.selector.matchLabels.verrazzano\.io/storage-for`,
					Value: "prometheus",
				},
			},
			expectError: false,
		},
		{
			// GIVEN a resource request with only memory set
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and the expected key/value overrides are returned
			name:   "only memory set",
			client: clientNoPV,
			request: common.ResourceRequestValues{
				Memory: memorySize,
			},
			expectOverrides: []bom.KeyValue{
				{
					Key:   "prometheus.prometheusSpec.resources.requests.memory",
					Value: memorySize,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, false)

			kvs, err := appendResourceRequestOverrides(ctx, &tt.request, []bom.KeyValue{})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectOverrides, kvs)
		})
	}
}
