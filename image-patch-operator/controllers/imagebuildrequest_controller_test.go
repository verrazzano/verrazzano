package controllers

import (
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestImageBuildRequestReconciler_Reconcile(t *testing.T) {
	type fields struct {
		Client client.Client
		Log    logr.Logger
		Scheme *runtime.Scheme
	}
	type args struct {
		req controllerruntime.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    controllerruntime.Result
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ImageBuildRequestReconciler{
				Client: tt.fields.Client,
				Log:    tt.fields.Log,
				Scheme: tt.fields.Scheme,
			}
			got, err := r.Reconcile(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reconcile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImageBuildRequestReconciler_SetupWithManager(t *testing.T) {
	type fields struct {
		Client client.Client
		Log    logr.Logger
		Scheme *runtime.Scheme
	}
	type args struct {
		mgr controllerruntime.Manager
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ImageBuildRequestReconciler{
				Client: tt.fields.Client,
				Log:    tt.fields.Log,
				Scheme: tt.fields.Scheme,
			}
			if err := r.SetupWithManager(tt.args.mgr); (err != nil) != tt.wantErr {
				t.Errorf("SetupWithManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
