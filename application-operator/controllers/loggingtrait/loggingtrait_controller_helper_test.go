// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Test_struct2Unmarshal(t *testing.T) {
	type args struct {
		obj interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    unstructured.Unstructured
		wantErr bool
	}{
		{
			name: "volumeMountJSON",
			args: args{
				obj: &corev1.VolumeMount{
					MountPath: loggingMountPath,
					Name:      loggingVolume,
					SubPath:   loggingKey,
					ReadOnly:  true,
				},
			},
			want: unstructured.Unstructured{
				Object: map[string]interface{}{
					"mountPath": loggingMountPath,
					"name":      loggingVolume,
					"subPath":   loggingKey,
					"readOnly":  true,
				},
			},
			wantErr: false,
		},
		{
			name: "nilJSON",
			args: args{
				obj: nil,
			},
			want: unstructured.Unstructured{
				Object: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := struct2Unmarshal(tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("struct2Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("struct2Unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}


func Test_locateContainersField(t *testing.T) {
	type args struct {
		res      *unstructured.Unstructured
	}

	// Create Deployment resource
	deploymentResource := unstructured.Unstructured{}
	deploymentResource.SetAPIVersion("apps/v1")
	deploymentResource.SetKind("Deployment")

	// Create Pod resource
	podResource := unstructured.Unstructured{}
	podResource.SetAPIVersion("v1")
	podResource.SetKind("Pod")

	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		{
			name: "deployment_test",
			args: args{
				res: &deploymentResource,
			},
			want:  true,
			want1: []string{"spec", "template", "spec", "containers"},
		},
		{
			name: "pod_test",
			args: args{
				res:      &podResource,
			},
			want:  true,
			want1: []string{"spec", "containers"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := locateContainersField(tt.args.res)
			if got != tt.want {
				t.Errorf("locateContainersField() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("locateContainersField() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_locateVolumesField(t *testing.T) {
	type args struct {
		res      *unstructured.Unstructured
	}

	// Create Deployment resource
	deploymentResource := unstructured.Unstructured{}
	deploymentResource.SetAPIVersion("apps/v1")
	deploymentResource.SetKind("Deployment")

	// Create Pod resource
	podResource := unstructured.Unstructured{}
	podResource.SetAPIVersion("v1")
	podResource.SetKind("Pod")

	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		{
			name: "deployment_test",
			args: args{
				res:      &deploymentResource,
			},
			want:  true,
			want1: []string{"spec", "template", "spec", "volumes"},
		},
		{
			name: "pod_test",
			args: args{
				res:      &podResource,
			},
			want:  true,
			want1: []string{"spec", "volumes"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := locateVolumesField(tt.args.res)
			if got != tt.want {
				t.Errorf("locateVolumesField() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("locateVolumesField() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
