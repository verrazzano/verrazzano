// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"testing"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzContext "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/context"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestReconcilerCheckComponentReadyState tests checkComponentReadyState
func TestReconcilerCheckComponentReadyState(t *testing.T) {
	temp := unitTesting
	defer func() {
		unitTesting = temp
	}()
	unitTesting = false
	k8sClient := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	context, _ := vzContext.NewVerrazzanoContext(vzlog.DefaultLogger(), k8sClient, &v1alpha1.Verrazzano{}, true)
	contextCompReady, _ := vzContext.NewVerrazzanoContext(vzlog.DefaultLogger(), k8sClient, &v1alpha1.Verrazzano{
		Status: v1alpha1.VerrazzanoStatus{
			Components: map[string]*v1alpha1.ComponentStatusDetails{
				rancher.ComponentName: {
					State: v1alpha1.CompStateReady,
				},
			},
		},
	}, true)
	contextCompNotReady, _ := vzContext.NewVerrazzanoContext(vzlog.DefaultLogger(), k8sClient, &v1alpha1.Verrazzano{
		Status: v1alpha1.VerrazzanoStatus{
			Components: map[string]*v1alpha1.ComponentStatusDetails{
				rancher.ComponentName: {
					State: v1alpha1.ComponentAvailable,
				},
			},
		},
	}, true)
	tests := []struct {
		name           string
		vzContext      vzContext.VerrazzanoContext
		k8sClient      client.Client
		setProfileFunc func()
		want           bool
		wantErr        bool
	}{
		// GIVEN VZ Reconciler
		// WHEN checkComponentReadyState is called
		// THEN false is returned with no error if any error occurs while fetching context
		{
			"TestReconcilerCheckComponentReadyState when failed to get context",
			context,
			k8sClient,
			nil,
			false,
			true,
		},
		// GIVEN VZ Reconciler
		// WHEN checkComponentReadyState is called
		// THEN false is returned with no error if component is disabled
		{
			"TestReconcilerCheckComponentReadyState when component is disabled",
			contextCompNotReady,
			k8sClient,
			func() {
				config.TestProfilesDir = relativeProfilesDir
			},
			false,
			false,
		},
		// GIVEN VZ Reconciler
		// WHEN checkComponentReadyState is called
		// THEN true is returned with no error if there is no error
		{
			"TestReconcilerCheckComponentReadyState when no error",
			contextCompReady,
			k8sClient,
			func() {
				config.TestProfilesDir = relativeProfilesDir
			},
			true,
			false,
		},
	}
	defer func() { config.TestProfilesDir = "" }()
	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{rancher.NewComponent()}
	})
	defer registry.ResetGetComponentsFn()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newVerrazzanoReconciler(tt.k8sClient)
			if tt.setProfileFunc != nil {
				tt.setProfileFunc()
			}
			got, err := r.checkComponentReadyState(tt.vzContext)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkComponentReadyState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkComponentReadyState() got = %v, want %v", got, tt.want)
			}
		})
	}
}
