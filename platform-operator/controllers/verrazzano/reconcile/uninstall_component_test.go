// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"reflect"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/helm"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestReconcilerUninstallSingleComponent tests uninstallSingleComponent
func TestReconcilerUninstallSingleComponent(t *testing.T) {
	type args struct {
		spiCtx           spi.ComponentContext
		UninstallContext *componentTrackerContext
		comp             spi.Component
	}
	compContext := &componentTrackerContext{
		state: compStateUninstallStart,
	}
	compCtxWithUninstall := &componentTrackerContext{
		state: compStateUninstall,
	}

	mockClient := fake.NewClientBuilder().Build()
	vz := &vzapi.Verrazzano{
		Status: vzapi.VerrazzanoStatus{
			Components: map[string]*vzapi.ComponentStatusDetails{
				rancher.ComponentName: {
					State: vzapi.ComponentAvailable,
				},
			},
		},
	}
	defer helm.SetDefaultRunner()
	helm.SetCmdRunner(vzos.GenericTestRunner{
		StdOut: []byte(""),
		StdErr: []byte("not found"),
		Err:    fmt.Errorf(unExpectedError),
	})
	k8sutil.GetCoreV1Func = common.MockGetCoreV1()
	k8sutil.GetDynamicClientFunc = common.MockDynamicClient()
	defer func() {
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
	}()
	tests := []struct {
		name      string
		k8sClient client.Client
		args      args
		want      controllerruntime.Result
		wantErr   bool
	}{
		// GIVEN VZ reconciler object
		// WHEN uninstallSingleComponent is called with invalid vz version
		// THEN no error is returned with empty result of reconcile invocation
		{
			"TestReconcilerUninstallSingleComponent with invalid vz version",
			mockClient,
			args{
				spi.NewFakeContext(mockClient, vz, nil, false),
				compContext,
				rancher.NewComponent(),
			},
			controllerruntime.Result{},
			false,
		},
		// GIVEN VZ reconciler object
		// WHEN uninstallSingleComponent is called
		// THEN error is returned with empty result of reconcile invocation if error occurs while uninstalling vz
		{
			"TestReconcilerUninstallSingleComponent when error occurs while deleting resource",
			mockClient,
			args{
				spi.NewFakeContext(mockClient, vz, nil, true),
				compCtxWithUninstall,
				rancher.NewComponent(),
			},
			controllerruntime.Result{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newVerrazzanoReconciler(tt.k8sClient)
			got, err := r.uninstallSingleComponent(tt.args.spiCtx, tt.args.UninstallContext, tt.args.comp)
			if (err != nil) != tt.wantErr {
				t.Errorf("uninstallSingleComponent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("uninstallSingleComponent() got = %v, want %v", got, tt.want)
			}
		})
	}
}
