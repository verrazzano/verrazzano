// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"reflect"
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestChooseCompState tests chooseCompState
// GIVEN componentStatus
// WHEN chooseCompState is called
// THEN corresponding state is returned based on componentStatus state
func TestChooseCompState(t *testing.T) {
	type args struct {
		componentStatus *vzapi.ComponentStatusDetails
	}
	tests := []struct {
		name string
		args args
		want componentInstallState
	}{
		{
			"TestChooseCompState when state is CompStateDisabled",
			args{
				componentStatus: &vzapi.ComponentStatusDetails{
					State: vzapi.CompStateDisabled,
				},
			},
			compStateInstallInitDisabled,
		},
		{
			"TestChooseCompState when state is CompStateDisabled",
			args{
				componentStatus: &vzapi.ComponentStatusDetails{
					State: vzapi.CompStateDisabled,
				},
			},
			compStateInstallInitDisabled,
		},
		{
			"TestChooseCompState when state is CompStatePreInstalling",
			args{
				componentStatus: &vzapi.ComponentStatusDetails{
					State: vzapi.CompStatePreInstalling,
				},
			},
			compStateWriteInstallStartedStatus,
		},
		{
			"TestChooseCompState when state is CompStateInstalling",
			args{
				componentStatus: &vzapi.ComponentStatusDetails{
					State: vzapi.CompStateInstalling,
				},
			},
			compStateWriteInstallStartedStatus,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chooseCompState(tt.args.componentStatus); got != tt.want {
				t.Errorf("chooseCompState() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestReconcilerInstallSingleComponent tests installSingleComponent
//
// GIVEN reconciler object
// WHEN installSingleComponent is called
// THEN corresponding action is take based on installSingleComponent state
//
//	and Result of a Reconciler invocation is returned
func TestReconcilerInstallSingleComponent(t *testing.T) {
	type args struct {
		spiCtx         spi.ComponentContext
		installContext *componentTrackerContext
		comp           spi.Component
		preUpgrade     bool
	}
	compContext := &componentTrackerContext{
		installState: compStateInstallInitDisabled,
	}
	compCtxWithPreInstall := &componentTrackerContext{
		installState: compStatePreInstall,
	}
	compCtxWithInstall := &componentTrackerContext{
		installState: compStateInstall,
	}
	compCtxWithWait := &componentTrackerContext{
		installState: compStateInstallWaitReady,
	}
	compCtxWithPostInstall := &componentTrackerContext{
		installState: compStatePostInstall,
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
	vzWithInvalidVersion := &vzapi.Verrazzano{
		Status: vzapi.VerrazzanoStatus{
			Components: map[string]*vzapi.ComponentStatusDetails{
				rancher.ComponentName: {
					State: vzapi.ComponentAvailable,
				},
			},
			Version: "b1.2",
		},
	}
	tests := []struct {
		name      string
		k8sClient client.Client
		args      args
		want      controllerruntime.Result
	}{
		{
			"TestReconcilerInstallSingleComponent with invalid vz version",
			mockClient,
			args{
				spi.NewFakeContext(mockClient, vzWithInvalidVersion, nil, true),
				compContext,
				rancher.NewComponent(),
				false,
			},
			controllerruntime.Result{},
		},
		{
			"TestReconcilerInstallSingleComponent when dependencies are not met with compStatePreInstall",
			mockClient,
			args{
				spi.NewFakeContext(mockClient, vz, nil, true),
				compCtxWithPreInstall,
				rancher.NewComponent(),
				false,
			},
			controllerruntime.Result{Requeue: true},
		},
		{
			"TestReconcilerInstallSingleComponent when dependencies are met with compStatePreInstall",
			mockClient,
			args{
				spi.NewFakeContext(mockClient, vz, nil, true),
				compCtxWithInstall,
				rancher.NewComponent(),
				false,
			},
			controllerruntime.Result{Requeue: true},
		},
		{
			"TestReconcilerInstallSingleComponent with compStateInstallWaitReady state",
			mockClient,
			args{
				spi.NewFakeContext(mockClient, vz, nil, true),
				compCtxWithWait,
				rancher.NewComponent(),
				false,
			},
			controllerruntime.Result{Requeue: true},
		},
		{
			"TestReconcilerInstallSingleComponent postInstall state",
			mockClient,
			args{
				spi.NewFakeContext(mockClient, vz, nil, true),
				compCtxWithPostInstall,
				rancher.NewComponent(),
				false,
			},
			controllerruntime.Result{Requeue: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newVerrazzanoReconciler(tt.k8sClient)
			if got := r.installSingleComponent(tt.args.spiCtx, tt.args.installContext, tt.args.comp, tt.args.preUpgrade); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("installSingleComponent() = %v, want %v", got, tt.want)
			}
		})
	}
}
