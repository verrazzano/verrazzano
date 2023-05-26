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
	profileDir             = "../../../../../manifests/profiles"
	testBomFilePath        = "../../testdata/test_bom.json"
	invalidTestBomFilePath = "../../testdata/invalid_test_bom.json"
)

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

func TestAppendOverrides(t *testing.T) {
	testCR := v1alpha1.Verrazzano{}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	fakeContext := spi.NewFakeContext(fakeClient, &testCR, nil, false)
	emptyString := ""

	expectedKVS := []bom.KeyValue{
		{Key: fluentOperatorImageKey, Value: "iad.ocir.io/odsbuilddev/sandboxes/test/fluent-operator"},
		{Key: fluentOperatorImageTag, Value: "v1"},
		{Key: fluentBitImageKey, Value: "iad.ocir.io/odsbuilddev/sandboxes/test/fluent-bit"},
		{Key: fluentBitImageTag, Value: "v1"},
		{Key: fluentOperatorInitImageKey, Value: "ghcr.io/oracle/oraclelinux"},
		{Key: fluentOperatorInitTag, Value: "8"},
		{Key: "image.pullSecrets.enabled", Value: "true"},
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("appendOverrides() got = %v, want %v", got, tt.want)
			}
		})
	}
}

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

	if !notReady {
		deployment.Status = appsv1.DeploymentStatus{
			Replicas:          1,
			AvailableReplicas: 0,
			UpdatedReplicas:   0,
		}
	}
	return deployment
}

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
	if !notReady {
		deployment.Status = appsv1.DaemonSetStatus{
			NumberAvailable:        0,
			UpdatedNumberScheduled: 1,
		}
	}
	return deployment
}

func TestIsFluentOperatorReady(t *testing.T) {
	type args struct {
		context spi.ComponentContext
	}
	testCR := v1alpha1.Verrazzano{}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		getFluentOperatorDeployment("testFluentOperator", map[string]string{"app": "fluent-operator"}, false),
		getFluentbitDaemonset("testFluentbit", map[string]string{"app": "fluent-bit"}, false),
	).Build()
	fakeClientWithNotReadyStatus := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		getFluentOperatorDeployment("testFluentOperator", map[string]string{"app": "fluent-operator"}, true),
		getFluentbitDaemonset("testFluentbit", map[string]string{"app": "fluent-bit"}, true),
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
			false,
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
