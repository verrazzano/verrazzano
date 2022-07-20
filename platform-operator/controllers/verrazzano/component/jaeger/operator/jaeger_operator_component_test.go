// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../../manifests/profiles"

var enabled = true
var jaegerEnabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			JaegerOperator: &vzapi.JaegerOperatorComponent{
				Enabled: &enabled,
			},
		},
	},
}

type ingressTestStruct struct {
	name   string
	spec   *vzapi.Verrazzano
	client client.Client
	err    error
}

// TestIsEnabled tests the IsEnabled function for the Jaeger Operator component
func TestIsEnabled(t *testing.T) {
	falseValue := false
	tests := []struct {
		name       string
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			// GIVEN a default Verrazzano custom resource
			// WHEN we call IsReady on the Jaeger Operator component
			// THEN the call returns false
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with the Jaeger Operator enabled
			// WHEN we call IsReady on the Jaeger Operator component
			// THEN the call returns true
			name:       "Test IsEnabled when Jaeger Operator component set to enabled",
			actualCR:   *jaegerEnabledCR,
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Jaeger Operator disabled
			// WHEN we call IsReady on the Jaeger Operator component
			// THEN the call returns false
			name: "Test IsEnabled when Jaeger Operator component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			expectTrue: false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, false, profilesRelativePath)
			assert.Equal(t, tt.expectTrue, NewComponent().IsEnabled(ctx.EffectiveCR()))
		})
	}
}

// TestGetMinVerrazzanoVersion tests whether the Jaeger Operator component
// is enabled only for VZ version 1.3.0 and above.
func TestGetMinVerrazzanoVersion(t *testing.T) {
	assert.Equal(t, constants.VerrazzanoVersion1_3_0, NewComponent().GetMinVerrazzanoVersion())
}

// TestGetDependencies tests whether cert-manager component is a dependency
// that needs to be installed prior to Jaeger operator
func TestGetDependencies(t *testing.T) {
	assert.Equal(t, []string{"cert-manager"}, NewComponent().GetDependencies())
}

// TestIsReady tests the IsReady function for the Jaeger Operator
func TestIsReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
		dryRun     bool
	}{
		{
			// GIVEN the Jaeger Operator deployment exists and there are available replicas
			// WHEN we call IsReady
			// THEN the call returns true
			name: "Test IsReady when Jaeger Operator is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
						Labels:    map[string]string{"name": ComponentName},
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"name": ComponentName},
						},
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName + "-95d8c5d96-m6mbr",
						Labels: map[string]string{
							"pod-template-hash": "95d8c5d96",
							"name":              ComponentName,
						},
					},
				},
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   ComponentNamespace,
						Name:        deploymentName + "-95d8c5d96",
						Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
					},
				},
			).Build(),
			expectTrue: true,
			dryRun:     true,
		},
		{
			// GIVEN the Jaeger Operator deployment exists and there are no available replicas
			// WHEN we call isJaegerOperatorReady
			// THEN the call returns false
			name: "Test IsReady when Jaeger Operator deployment is not ready",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 0,
						Replicas:          1,
						UpdatedReplicas:   0,
					},
				}).Build(),
			expectTrue: false,
			dryRun:     true,
		},
		{
			// GIVEN the Jaeger Operator deployment does not exist
			// WHEN we call IsReady
			// THEN the call returns false
			name:       "Test IsReady when Jaeger Operator deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
			dryRun:     true,
		},
		{
			// GIVEN the Jaeger Operator deployment does not exist, and dry run is false
			// WHEN we call IsReady
			// THEN the call returns false
			name:       "Test IsReady when Jaeger Operator deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
			dryRun:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, tt.dryRun)
			assert.Equal(t, tt.expectTrue, NewComponent().IsReady(ctx))
		})
	}
}

// TestPreInstall tests the PreInstall function for various scenarios.
func TestPreInstall(t *testing.T) {
	for _, tt := range getPreInstallTests() {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, tt.spec, false)
			err := NewComponent().PreInstall(ctx)
			if tt.err != nil {
				assert.Error(t, err)
				assert.IsTypef(t, tt.err, err, "")
			} else {
				assert.NoError(t, err)
			}
			ns := corev1.Namespace{}
			err = tt.client.Get(context.TODO(), types.NamespacedName{Name: ComponentNamespace}, &ns)
			assert.NoError(t, err)
		})
	}
}

// TestPostInstall tests the component PostInstall function
func TestPostInstall(t *testing.T) {
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../../..",
	})

	for _, tt := range getIngressTests(false) {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, tt.spec, false, profilesRelativePath)
			err := NewComponent().PostInstall(ctx)
			if tt.err != nil {
				assert.Equal(t, tt.err, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateInstall tests the validation of the Jaeger Operator installation and the Verrazzano CR
func TestValidateInstall(t *testing.T) {
	tests := []struct {
		name        string
		vz          vzapi.Verrazzano
		expectError bool
	}{
		// GIVEN a default Verrazzano CR,
		// WHEN we call the ValidateInstall function,
		// THEN no error is returned.
		{
			name:        "test nothing enabled",
			vz:          vzapi.Verrazzano{},
			expectError: false,
		},
		// GIVEN a Verrazzano CR with Jaeger Component enabled,
		// WHEN we call the ValidateInstall function
		// THEN no error is returned.
		{
			name: "test jaeger operator enabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{Enabled: &trueValue},
					},
				},
			},
			expectError: false,
		},
		// GIVEN a Verrazzano CR with Jaeger Component disabled,
		// WHEN we call the ValidateInstall function
		// THEN no error is returned.
		{
			name: "test jaeger operator disabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{Enabled: &falseValue},
					},
				},
			},
			expectError: false,
		},
	}
	c := jaegerOperatorComponent{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateInstall(&tt.vz)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateUpdate tests the Jaeger Operator ValidateUpdate function
func TestValidateUpdate(t *testing.T) {
	oldVZ := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				JaegerOperator: &vzapi.JaegerOperatorComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	newVZ := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				JaegerOperator: &vzapi.JaegerOperatorComponent{
					Enabled: &falseValue,
				},
			},
		},
	}
	// GIVEN a default Verrazzano custom resource with Jaeger Operator component enabled,
	// WHEN we try to update Verrazzano CR to disable Jaeger Component,
	// THEN an error is returned.
	assert.Error(t, NewComponent().ValidateUpdate(&oldVZ, &newVZ))
	// GIVEN a default Verrazzano custom resource with Jaeger Operator component enabled,
	// WHEN we try to update Verrazzano CR with no changes,
	// THEN no error is returned.
	assert.NoError(t, NewComponent().ValidateUpdate(&oldVZ, &oldVZ))
}

// TestPostUpgrade tests the component PostUpgrade function
func TestPostUpgrade(t *testing.T) {
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../../..",
	})

	for _, tt := range getIngressTests(true) {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, tt.spec, false, profilesRelativePath)
			err := NewComponent().PostUpgrade(ctx)
			if tt.err != nil {
				assert.Equal(t, tt.err, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMonitorOverrides tests the monitor overrides function
func TestMonitorOverrides(t *testing.T) {
	tests := []struct {
		name       string
		actualCR   *vzapi.Verrazzano
		expectTrue bool
	}{
		// GIVEN a default Verrazzano custom resource,
		// WHEN we call MonitorOverrides on the Jaeger component,
		// THEN it returns false
		{
			"Monitor changes should be false by default when VZ spec does not have a Jaeger Component section",
			&vzapi.Verrazzano{},
			false,
		},
		// GIVEN a Verrazzano custom resource with a Jaeger Component in the spec section,
		// WHEN we call MonitorOverrides on the Jaeger component,
		// THEN it returns true
		{
			"Monitor changes should be true by default when VZ spec has a Jaeger Component section",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{},
					},
				},
			},
			true,
		},
		// GIVEN a Verrazzano custom resource with a Jaeger Component in the spec section
		//       with monitor changes flag explicitly set to true,
		// WHEN we call MonitorOverrides on the Jaeger component,
		// THEN it returns true
		{
			"Monitor changes should be true when set explicitly in the VZ CR",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &trueValue,
							},
						},
					},
				},
			},
			true,
		},
		// GIVEN a Verrazzano custom resource with a Jaeger Component in the spec section
		//       with monitor changes flag explicitly set to false,
		// WHEN we call MonitorOverrides on the Jaeger component,
		// THEN it returns false
		{
			"Monitor changes should be false when set explicitly in the VZ CR",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &falseValue,
							},
						},
					},
				},
			},
			false,
		},
	}
	client := createFakeClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(client, tt.actualCR, false, profilesRelativePath)
			if tt.expectTrue {
				assert.True(t, NewComponent().MonitorOverrides(ctx), tt.name)
			} else {
				assert.False(t, NewComponent().MonitorOverrides(ctx), tt.name)
			}
		})
	}
}

func getIngressTests(isUpgradeOperation bool) []ingressTestStruct {
	// TLS certificate check is done only during post install, and skipped during
	// post upgrade phase. Conditionally adding the expected error based on whether
	// it is testing the installation flow or the upgrade flow.
	var certificateErr error = ctrlerrors.RetryableError{
		Source:    deploymentName,
		Operation: "Check if certificates are ready",
	}
	if isUpgradeOperation {
		certificateErr = nil
	}
	return []ingressTestStruct{
		{
			// GIVEN a default Verrazzano custom resource with ingress controller running,
			// WHEN we call PostInstall/PostUpgrade on the Jaeger component,
			// THEN an error is returned as the ingress resource cannot be created.
			"should return error when ingress service is not up",
			&vzapi.Verrazzano{},
			createFakeClient(),
			fmt.Errorf("Failed create/update Jaeger ingress: Failed building DNS domain name: services \"ingress-controller-ingress-nginx-controller\" not found"),
		},
		{
			// GIVEN a default Verrazzano custom resource, with ingress controller running,
			// WHEN we call PostInstall/PostUpgrade on the Jaeger component,
			// THEN an error is returned as the certificates cannot be created.
			"should return error when ingress service is up and cert manager is enabled",
			&vzapi.Verrazzano{},
			createFakeClient(vzIngressService),
			certificateErr,
		},
		{
			// GIVEN a default Verrazzano custom resource, with ingress controller running and cert manager disabled,
			// WHEN we call PostInstall/PostUpgrade on the Jaeger component,
			// THEN no error is returned.
			"should not return error when ingress service is up and cert manager is disabled",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			createFakeClient(vzIngressService),
			nil,
		},
		{
			// GIVEN a default Verrazzano custom resource using an external DNS configuration, with ingress controller
			//       running and cert manager disabled,
			// WHEN we call PostInstall/PostUpgrade on the Jaeger component,
			// THEN no error is returned.
			"should not return error when ingress service is up, cert manager is disabled and external OCI DNS is used",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &falseValue,
						},
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								DNSZoneOCID:            "somezoneocid",
								DNSZoneCompartmentOCID: "somenewocid",
								OCIConfigSecret:        globalconst.VerrazzanoESInternal,
								DNSZoneName:            "newzone.dns.io",
							},
						},
					},
				},
			},
			createFakeClient(vzIngressService),
			nil,
		},
	}
}
