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
					Name:      "loggingVolume",
					SubPath:   loggingKey,
					ReadOnly:  true,
				},
			},
			want: unstructured.Unstructured{
				Object: map[string]interface{}{
					"mountPath": loggingMountPath,
					"name":      "loggingVolume",
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

func Test_appendSliceOfInterface(t *testing.T) {
	type args struct {
		aSlice []interface{}
		bSlice []interface{}
	}

	var empty []interface{}
	var noDuplicateMounts = []corev1.VolumeMount{
		{
			Name:      "test-volume-1",
			MountPath: "test/mount/path/1",
		},
		{
			Name:      "test-volume-2",
			MountPath: "test/mount/path/2",
		},
	}
	var noDuplicates = make([]interface{}, len(noDuplicateMounts))
	for _, v := range noDuplicateMounts {
		noDuplicates = append(noDuplicates, v)
	}

	var duplicateMounts1 = []corev1.VolumeMount{
		{
			Name:      "test-volume-1",
			MountPath: "test/mount/path/1",
		},
		{
			Name:      "test-volume-2",
			MountPath: "test/mount/path/2",
		},
	}
	var duplicates1 = make([]interface{}, 0)
	for _, v := range duplicateMounts1 {
		duplicates1 = append(duplicates1, v)
	}

	var duplicateMounts2 = []corev1.VolumeMount{
		{
			Name:      "test-volume-2",
			MountPath: "test/mount/path/2",
		},
		{
			Name:      "test-volume-3",
			MountPath: "test/mount/path/3",
		},
	}
	var duplicates2 = make([]interface{}, 0)
	for _, v := range duplicateMounts2 {
		duplicates2 = append(duplicates2, v)
	}

	var duplicateMountsWant = []corev1.VolumeMount{
		{
			Name:      "test-volume-2",
			MountPath: "test/mount/path/2",
		},
		{
			Name:      "test-volume-3",
			MountPath: "test/mount/path/3",
		},
		{
			Name:      "test-volume-1",
			MountPath: "test/mount/path/1",
		},
	}
	var duplicatesWant = make([]interface{}, 0)
	for _, v := range duplicateMountsWant {
		duplicatesWant = append(duplicatesWant, v)
	}

	tests := []struct {
		name string
		args args
		want []interface{}
	}{
		{
			name: "append slice both empty",
			args: args{
				aSlice: empty,
				bSlice: empty,
			},
			want: make([]interface{}, 0),
		},
		{
			name: "append slice no duplicates",
			args: args{
				aSlice: empty,
				bSlice: noDuplicates,
			},
			want: noDuplicates,
		},
		{
			name: "append slice with duplicates",
			args: args{
				aSlice: duplicates1,
				bSlice: duplicates2,
			},
			want: duplicatesWant,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appendSliceOfInterface(tt.args.aSlice, tt.args.bSlice); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("appendSliceOfInterface() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_locateContainersField(t *testing.T) {
	type args struct {
		res *unstructured.Unstructured
	}

	// Create Deployment resource
	deploymentResource := unstructured.Unstructured{}
	deploymentResource.SetAPIVersion("apps/v1")
	deploymentResource.SetKind("Deployment")

	// Create Pod resource
	podResource := unstructured.Unstructured{}
	podResource.SetAPIVersion("v1")
	podResource.SetKind("Pod")

	// Create ContainerizedWorkload resource
	containerizedWorkloadResource := unstructured.Unstructured{}
	containerizedWorkloadResource.SetAPIVersion("v1")
	containerizedWorkloadResource.SetKind("ContainerizedWorkload")

	// Create StatefuleSet resource
	statefuleSetResource := unstructured.Unstructured{}
	statefuleSetResource.SetAPIVersion("apps/v1")
	statefuleSetResource.SetKind("StatefuleSet")

	// Create DaemonSet resource
	daemonSetResource := unstructured.Unstructured{}
	daemonSetResource.SetAPIVersion("apps/v1")
	daemonSetResource.SetKind("DaemonSet")

	// Create Secret resource
	secretResource := unstructured.Unstructured{}
	secretResource.SetAPIVersion("v1")
	secretResource.SetKind("Secret")

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
				res: &podResource,
			},
			want:  true,
			want1: []string{"spec", "containers"},
		},
		{
			name: "containerizedWorkload",
			args: args{
				res: &containerizedWorkloadResource,
			},
			want:  true,
			want1: []string{"spec", "containers"},
		},
		{
			name: "statefuleSet_test",
			args: args{
				res: &statefuleSetResource,
			},
			want:  true,
			want1: []string{"spec", "template", "spec", "containers"},
		},
		{
			name: "daemonSet_test",
			args: args{
				res: &daemonSetResource,
			},
			want:  true,
			want1: []string{"spec", "template", "spec", "containers"},
		},
		{
			name: "secret_test",
			args: args{
				res: &secretResource,
			},
			want:  false,
			want1: nil,
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
		res *unstructured.Unstructured
	}

	// Create Deployment resource
	deploymentResource := unstructured.Unstructured{}
	deploymentResource.SetAPIVersion("apps/v1")
	deploymentResource.SetKind("Deployment")

	// Create Pod resource
	podResource := unstructured.Unstructured{}
	podResource.SetAPIVersion("v1")
	podResource.SetKind("Pod")

	// Create ContainerizedWorkload resource
	containerizedWorkloadResource := unstructured.Unstructured{}
	containerizedWorkloadResource.SetAPIVersion("v1")
	containerizedWorkloadResource.SetKind("ContainerizedWorkload")

	// Create StatefuleSet resource
	statefuleSetResource := unstructured.Unstructured{}
	statefuleSetResource.SetAPIVersion("apps/v1")
	statefuleSetResource.SetKind("StatefuleSet")

	// Create DaemonSet resource
	daemonSetResource := unstructured.Unstructured{}
	daemonSetResource.SetAPIVersion("apps/v1")
	daemonSetResource.SetKind("DaemonSet")

	// Create Secret resource
	secretResource := unstructured.Unstructured{}
	secretResource.SetAPIVersion("v1")
	secretResource.SetKind("Secret")

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
			want1: []string{"spec", "template", "spec", "volumes"},
		},
		{
			name: "pod_test",
			args: args{
				res: &podResource,
			},
			want:  true,
			want1: []string{"spec", "volumes"},
		},
		{
			name: "containerizedWorkload",
			args: args{
				res: &containerizedWorkloadResource,
			},
			want:  true,
			want1: []string{"spec", "volumes"},
		},
		{
			name: "statefuleSet_test",
			args: args{
				res: &statefuleSetResource,
			},
			want:  true,
			want1: []string{"spec", "template", "spec", "volumes"},
		},
		{
			name: "daemonSet_test",
			args: args{
				res: &daemonSetResource,
			},
			want:  true,
			want1: []string{"spec", "template", "spec", "volumes"},
		},
		{
			name: "secret_test",
			args: args{
				res: &secretResource,
			},
			want:  false,
			want1: nil,
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
