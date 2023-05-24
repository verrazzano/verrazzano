// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentoperator

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

var (
	testScheme = runtime.NewScheme()
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
			if got := GetOverrides(tt.args.object); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOverrides() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_appendOverrides(t *testing.T) {
	//testCR := v1alpha1.Verrazzano{}
	//fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	//fakeContext := spi.NewFakeContext(fakeClient, &testCR, nil, false, profileDir)
	type args struct {
		ctx spi.ComponentContext
		in1 string
		in2 string
		in3 string
		kvs []bom.KeyValue
	}
	tests := []struct {
		name    string
		args    args
		want    []bom.KeyValue
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func Test_applyFluentBitConfigMap(t *testing.T) {
	type args struct {
		compContext spi.ComponentContext
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := applyFluentBitConfigMap(tt.args.compContext); (err != nil) != tt.wantErr {
				t.Errorf("applyFluentBitConfigMap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_applyOpenSearchClusterOutputs(t *testing.T) {
	type args struct {
		compContext spi.ComponentContext
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := applyOpenSearchClusterOutputs(tt.args.compContext); (err != nil) != tt.wantErr {
				t.Errorf("applyOpenSearchClusterOutputs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_isFluentOperatorReady(t *testing.T) {
	type args struct {
		context spi.ComponentContext
	}
	tests := []struct {
		name string
		args args
		want bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFluentOperatorReady(tt.args.context); got != tt.want {
				t.Errorf("isFluentOperatorReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
