// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/util/openapi"
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
		// TODO: Add test cases.
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

func Test_locateField(t *testing.T) {
	type args struct {
		document   openapi.Resources
		res        *unstructured.Unstructured
		fieldPaths [][]string
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := locateField(tt.args.document, tt.args.res, tt.args.fieldPaths)
			if got != tt.want {
				t.Errorf("locateField() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("locateField() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_locateContainersField(t *testing.T) {
	type args struct {
		document openapi.Resources
		res      *unstructured.Unstructured
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := locateContainersField(tt.args.document, tt.args.res)
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
		document openapi.Resources
		res      *unstructured.Unstructured
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := locateVolumesField(tt.args.document, tt.args.res)
			if got != tt.want {
				t.Errorf("locateVolumesField() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("locateVolumesField() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_locateVolumeMountsField(t *testing.T) {
	type args struct {
		document openapi.Resources
		res      *unstructured.Unstructured
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := locateVolumeMountsField(tt.args.document, tt.args.res)
			if got != tt.want {
				t.Errorf("locateVolumeMountsField() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("locateVolumeMountsField() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
