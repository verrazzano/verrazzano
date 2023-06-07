// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentoperator

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

var (
	testScheme = runtime.NewScheme()
)

const (
	testBomFilePath        = "../../testdata/test_bom.json"
	invalidTestBomFilePath = "../../testdata/invalid_test_bom.json"
)

// TestGetOverrides tests getOverrides functions for the FluentOperator component
// WHEN I call GetOverrides function with Verrazzano CR object
// THEN all installed overrides available in Verrazzano CR for the FluentOperator are returned.
func TestGetOverrides(t *testing.T) {
	type args struct {
		object runtime.Object
	}
	ref := &corev1.ConfigMapKeySelector{
		Key: "testOverride",
	}
	oV1Beta1 := v1beta1.InstallOverrides{
		ValueOverrides: []v1beta1.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	oV1Alpha1 := v1alpha1.InstallOverrides{
		ValueOverrides: []v1alpha1.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			"TestGetOverrides v1alpha1",
			args{
				&v1alpha1.Verrazzano{
					Spec: v1alpha1.VerrazzanoSpec{
						Components: v1alpha1.ComponentSpec{
							FluentOperator: &v1alpha1.FluentOperatorComponent{
								InstallOverrides: oV1Alpha1,
							},
						},
					},
				},
			},
			oV1Alpha1.ValueOverrides,
		},
		{
			"TestGetOverrides v1beta1",
			args{
				&v1beta1.Verrazzano{
					Spec: v1beta1.VerrazzanoSpec{
						Components: v1beta1.ComponentSpec{
							FluentOperator: &v1beta1.FluentOperatorComponent{
								InstallOverrides: oV1Beta1,
							},
						},
					},
				},
			},
			oV1Beta1.ValueOverrides,
		},
		{
			"Empty overrides when component is nil",
			args{&v1beta1.Verrazzano{}},
			[]v1beta1.Overrides{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getOverrides(tt.args.object); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOverrides() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAppendOverrides tests appendOverrides functions for the FluentOperator component
// GIVEN a FluentOperator component
// WHEN I call AppendOverrides function with FluentOperator context and slice of key-value
// THEN slice of the array is updated with FluentOperator Overrides.
func TestAppendOverrides(t *testing.T) {
	testCR := v1alpha1.Verrazzano{}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.
		Scheme).Build()
	fakeContext := spi.NewFakeContext(fakeClient, &testCR, nil, false)
	emptyString := ""
	config.TestHelmConfigDir = "../../../../helm_config"
	defer func() { config.TestHelmConfigDir = "" }()
	expectedKVS := []bom.KeyValue{
		{Key: fluentOperatorImageTag, Value: "v2.2.0-20230526122409-3662eb4"},
		{Key: fluentOperatorImageKey, Value: "ghcr.io/verrazzano/fluent-operator"},
		{Key: fluentBitImageTag, Value: "v2.0.11-20230526122435-3bff26487"},
		{Key: fluentBitImageKey, Value: "ghcr.io/verrazzano/fluent-bit"},
		{Key: fluentOperatorInitTag, Value: "8"},
		{Key: fluentOperatorInitImageKey, Value: "ghcr.io/oracle/oraclelinux"},
	}
	type args struct {
		ctx     spi.ComponentContext
		in1     string
		in2     string
		in3     string
		bomFile string
		kvs     []bom.KeyValue
	}
	tests := []struct {
		name    string
		args    args
		want    []bom.KeyValue
		wantErr bool
	}{
		{
			"With all overrides values",
			args{
				fakeContext,
				emptyString,
				emptyString,
				emptyString,
				testBomFilePath,
				[]bom.KeyValue{},
			},
			expectedKVS,
			false,
		},
		{
			"With invalid BOM file",
			args{
				fakeContext,
				emptyString,
				emptyString,
				emptyString,
				invalidTestBomFilePath,
				[]bom.KeyValue{},
			},
			[]bom.KeyValue{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetDefaultBomFilePath(tt.args.bomFile)
			defer func() {
				config.SetDefaultBomFilePath("")
			}()
			got, err := appendOverrides(tt.args.ctx, tt.args.in1, tt.args.in2, tt.args.in3, tt.args.kvs)
			if (err != nil) != tt.wantErr {
				t.Errorf("appendOverrides() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tt.want) > 1 {
				// exclude the file override from the list as last override is the temp file override.
				valuesOverride := got[:len(got)-1]
				if !reflect.DeepEqual(valuesOverride, tt.want) {
					t.Errorf("appendOverrides() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// TestApplyFluentBitConfigMap tests applyFluentBitConfigMap functions for the FluentOperator component
// WHEN I call ApplyFluentBitConfigMap function with FluentOperator context
// THEN nil is returned if Fluentbit ConfigMap is successfully created; otherwise, error is returned.
func TestApplyFluentBitConfigMap(t *testing.T) {
	type args struct {
		compContext     spi.ComponentContext
		testManifestDir string
	}
	testCR := v1alpha1.Verrazzano{}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	fakeContext := spi.NewFakeContext(fakeClient, &testCR, nil, false)
	defer func() {
		config.TestThirdPartyManifestDir = ""
	}()
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"test with valid configmap map",
			args{
				fakeContext,
				"../../../../thirdparty/manifests",
			},
			false,
		},
		{
			"test with inValid configmap map",

			args{
				fakeContext,
				"test",
			},
			true,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			config.TestThirdPartyManifestDir = tt.args.testManifestDir
			if err := applyFluentBitConfigMap(tt.args.compContext); (err != nil) != tt.wantErr {
				t.Errorf("applyFluentBitConfigMap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIsFluentOperatorReady tests isFluentOperatorReady functions for the FluentOperator component
// WHEN I call isFluentOperatorReady function with FluentOperator context
// THEN true is returned, if both FluentOperator Deployment and Fluentbit DaemonSet are ready; otherwise, false will be returned.
func TestIsFluentOperatorReady(t *testing.T) {
	type args struct {
		context spi.ComponentContext
	}
	testCR := v1alpha1.Verrazzano{}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		getFluentOperatorDeployment(ComponentName, map[string]string{"app": "fluent-operator"}, false),
		getFluentbitDaemonset(fluentbitDaemonSet, map[string]string{"app": "fluent-bit"}, false),
		getTestPod(ComponentName, ComponentNamespace, ComponentName),
		getTestReplicaSet(ComponentName, ComponentNamespace),
		getTestControllerRevision(fluentbitDaemonSet, ComponentNamespace),
		getTestDaemonSetPod(fluentbitDaemonSet, ComponentNamespace, fluentbitDaemonSet),
	).Build()
	fakeClientWithNotReadyStatus := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		getFluentOperatorDeployment(ComponentName, map[string]string{"app": "fluent-operator"}, true),
		getFluentbitDaemonset(fluentbitDaemonSet, map[string]string{"app": "fluent-bit"}, true),
	).Build()
	fakeContext := spi.NewFakeContext(fakeClient, &testCR, nil, false)
	fakeCtxWithNotReadyStatus := spi.NewFakeContext(fakeClientWithNotReadyStatus, &testCR, nil, false)
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"When fluentOperator is ready",
			args{
				fakeContext,
			},
			true,
		},
		{
			"When fluentOperator is not ready",
			args{
				fakeCtxWithNotReadyStatus,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFluentOperatorReady(tt.args.context); got != tt.want {
				t.Errorf("isFluentOperatorReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

// getFluentOperatorDeployment returns FluentOperator deployment for the Unit tests.
func getFluentOperatorDeployment(name string, labels map[string]string, notReady bool) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          1,
			AvailableReplicas: 1,
			UpdatedReplicas:   1,
		},
	}

	if notReady {
		deployment.Status = appsv1.DeploymentStatus{
			Replicas:          1,
			AvailableReplicas: 0,
			UpdatedReplicas:   0,
		}
	}
	return deployment
}

// getTestPod returns FluentOperator pod for the Unit tests.
func getTestPod(name, namespace, labelApp string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name + "-95d8c5d97-m6mbr",
			Labels: map[string]string{
				"pod-template-hash": "95d8c5d97",
				"app":               labelApp,
			},
		},
	}
}

// getTestDaemonSetPod returns Fluentbit DaemonSet pod for the Unit tests.
func getTestDaemonSetPod(name, namespace, labelApp string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name + "-95d8c5d97-m6mbr",
			Labels: map[string]string{
				"controller-revision-hash": "95d8c5d97",
				"app":                      labelApp,
			},
		},
	}
}

// getTestReplicaSet returns FluentOperator ReplicaSet for the Unit tests.
func getTestReplicaSet(name, namespace string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name + "-95d8c5d97",
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
	}
}

// getTestControllerRevision returns Fluentbit ControllerRevision for the Unit tests.
func getTestControllerRevision(name, namespace string) *appsv1.ControllerRevision {
	return &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name + "-95d8c5d97",
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
		Revision: 1,
	}
}

// getFluentbitDaemonset returns Fluentbit DaemonSet for the Unit tests.
func getFluentbitDaemonset(name string, labels map[string]string, notReady bool) *appsv1.DaemonSet {
	deployment := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
		Status: appsv1.DaemonSetStatus{
			NumberAvailable:        1,
			UpdatedNumberScheduled: 1,
		},
	}
	if notReady {
		deployment.Status = appsv1.DaemonSetStatus{
			NumberAvailable:        0,
			UpdatedNumberScheduled: 1,
		}
	}
	return deployment
}
