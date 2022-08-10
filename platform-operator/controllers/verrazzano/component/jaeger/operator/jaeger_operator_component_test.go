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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

const (
	nameOverrideJSON          = "{\"nameOverride\": \"test\"}"
	fullnameOverrideJSON      = "{\"fullnameOverride\": \"testFullName\"}"
	serviceAccountNameJSON    = "{\"serviceAccount\": {\"name\": \"testServiceAccount\"}}"
	ingressJSON               = "{\"ingress\": {\"enabled\": true}}"
	validOverrideJSON         = "{\"serviceAccount\": {\"create\": false}}"
	defaultJaegerDisabledJSON = "{\"jaeger\":{\"create\": false}}"
	defaultJaegerEnabledJSON  = "{\"jaeger\":{\"create\": true}}"
	k8sAppNameLabel           = "app.kubernetes.io/name"
)

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
		cr         *vzapi.Verrazzano
		expectTrue bool
		dryRun     bool
	}{
		{
			// GIVEN the Jaeger Operator deployment does not exist
			// WHEN we call IsReady
			// THEN the call returns false
			name:       "Test IsReady when Jaeger Operator deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			cr:         &vzapi.Verrazzano{},
			expectTrue: false,
			dryRun:     true,
		},
		{
			// GIVEN the Jaeger Operator deployment does not exist, and dry run is false
			// WHEN we call IsReady
			// THEN the call returns false
			name:       "Test IsReady when Jaeger Operator deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			cr:         &vzapi.Verrazzano{},
			expectTrue: false,
			dryRun:     false,
		},
		//0XX
		{
			// GIVEN Jaeger operator, collector and query have no available pods,
			// WHEN we call IsReady,
			// THEN the call returns false.
			name: "Test IsReady when Jaeger Operator, Collector and Query are not available",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				getAllJaegerObjects(0, 0, 0)...,
			).Build(),
			cr:         &vzapi.Verrazzano{},
			expectTrue: false,
			dryRun:     true,
		},
		{
			// GIVEN Jaeger operator and collector have no available pods, but query has available pods,
			// WHEN we call IsReady,
			// THEN the call returns false.
			name: "Test IsReady when Jaeger Operator and Collector is not available but Query is available",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				getAllJaegerObjects(0, 0, 1)...,
			).Build(),
			cr:         &vzapi.Verrazzano{},
			expectTrue: false,
			dryRun:     true,
		},
		{
			// GIVEN Jaeger operator and query have no available pods, but collector has available pods
			// WHEN we call IsReady
			// THEN the call returns false
			name: "Test IsReady when Jaeger Operator and Query is not available but Collector is available",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				getAllJaegerObjects(0, 1, 0)...,
			).Build(),
			cr:         &vzapi.Verrazzano{},
			expectTrue: false,
			dryRun:     true,
		},
		{
			// GIVEN Jaeger operator has no available pods, but collector and query have available pods
			// WHEN we call IsReady,
			// THEN the call returns false.
			name: "Test IsReady when Jaeger Operator is not available but Query and Collector is available",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				getAllJaegerObjects(0, 1, 1)...,
			).Build(),
			cr:         &vzapi.Verrazzano{},
			expectTrue: false,
			dryRun:     true,
		},
		//1XX
		{
			// GIVEN Jaeger operator has available pods but collector and query have no available pods,
			// WHEN we call IsReady,
			// THEN the call returns false.
			name: "Test IsReady when Jaeger Operator is available but Collector and Query are not available",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				getAllJaegerObjects(1, 0, 0)...,
			).Build(),
			cr:         &vzapi.Verrazzano{},
			expectTrue: false,
			dryRun:     true,
		},
		{
			// GIVEN Jaeger operator and query have available pods but collector has no available pods,
			// WHEN we call IsReady,
			// THEN the call returns false.
			name: "Test IsReady when Jaeger Operator and Query are available but Collector is not available",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				getAllJaegerObjects(1, 0, 1)...,
			).Build(),
			cr:         &vzapi.Verrazzano{},
			expectTrue: false,
			dryRun:     true,
		},
		{
			// GIVEN Jaeger operator and collector have available pods but query has no available pods,
			// WHEN we call IsReady,
			// THEN the call returns false.
			name: "Test IsReady when Jaeger Operator and Collector is available but Query is not available",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				getAllJaegerObjects(1, 1, 0)...,
			).Build(),
			cr:         &vzapi.Verrazzano{},
			expectTrue: false,
			dryRun:     true,
		},
		{
			// GIVEN Jaeger operator, collector and query have available pods,
			// WHEN we call IsReady,
			// THEN the call returns false.
			name: "Test IsReady when Jaeger Operator, Collector and Query pods are available",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				getAllJaegerObjects(1, 1, 1)...,
			).Build(),
			cr:         &vzapi.Verrazzano{},
			expectTrue: true,
			dryRun:     true,
		},
		{
			// GIVEN Jaeger operator has available pods and VZ managed default jaeger CR is disabled,
			// WHEN we call IsReady,
			// THEN the call returns true.
			name: "Test IsReady when Jaeger Operator is available but default Jaeger CR is disabled",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				getJaegerOperatorObjects(1)...,
			).Build(),
			cr:         getVZCRWithDefaultJaegerOverride(defaultJaegerDisabledJSON),
			expectTrue: true,
			dryRun:     true,
		},
		{
			// GIVEN Jaeger operator has available pods and VZ managed default jaeger CR is explicitly enabled without
			//       deployments for collector and query components,
			// WHEN we call IsReady,
			// THEN the call returns false.
			name: "Test IsReady when Jaeger Operator is available and default Jaeger CR is enabled without query and collector deployments",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				getJaegerOperatorObjects(1)...,
			).Build(),
			cr:         getVZCRWithDefaultJaegerOverride(defaultJaegerEnabledJSON),
			expectTrue: false,
			dryRun:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, tt.cr, tt.dryRun)
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

// TestGetIngressAndCertificateNames tests getting Jaeger ingress names and certificate names
func TestGetIngressAndCertificateNames(t *testing.T) {
	tests := []struct {
		name      string
		actualCR  vzapi.Verrazzano
		ingresses []types.NamespacedName
		certs     []types.NamespacedName
	}{
		{
			// GIVEN a Verrazzano custom resource with Jaeger Operator component enabled
			// WHEN we call GetIngressNames and GetCertificateNames on the Jaeger Operator component
			// THEN we expect to find the Jaeger ingress and certs
			name: "Test GetIngressNames and GetCertificateNames when Jaeger Operator set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			ingresses: jaegerIngressNames,
			certs:     certificates,
		},
		{
			// GIVEN a Verrazzano custom resource with Jaeger Operator enabled and OpenSearch disabled
			// WHEN we call GetIngressNames and GetCertificateNames on the Jaeger Operator component
			// THEN we do not expect to find the Jaeger ingress and certs
			name: "Test GetIngressNames and GetCertificateNames when OpenSearch is disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							Enabled: &trueValue,
						},
						Elasticsearch: &vzapi.ElasticsearchComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			ingresses: []types.NamespacedName{},
			certs:     []types.NamespacedName{},
		},
		{
			// GIVEN a Verrazzano custom resource with Jaeger operator is enabled and instance is disabled
			// WHEN we call GetIngressNames and GetCertificateNames on the Jaeger Operator component
			// THEN we do not expect to find the Jaeger ingress and certs
			name: "Test GetIngressNames and GetCertificateNames when Jaeger instance is disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							Enabled: &trueValue,
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &trueValue,
								ValueOverrides: []vzapi.Overrides{
									{
										Values: &apiextensionsv1.JSON{
											Raw: []byte(jaegerDisabledJSON),
										},
									},
								},
							},
						},
					},
				},
			},
			ingresses: []types.NamespacedName{},
			certs:     []types.NamespacedName{},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, false)
			assert.Equal(t, tt.ingresses, NewComponent().GetIngressNames(ctx))
			assert.Equal(t, tt.certs, NewComponent().GetCertificateNames(ctx))
		})
	}
}

// TestValidateInstall tests the validation of the Jaeger Operator installation and the Verrazzano CR
func TestValidateInstall(t *testing.T) {
	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithRuntimeObjects().Build(), nil
	}
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
		// GIVEN a Verrazzano CR with Jaeger Component enabled and nameOverride set,
		// WHEN we call the ValidateInstall function
		// THEN an error is returned.
		{
			name: "test jaeger operator override name",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							Enabled: &trueValue,
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &trueValue,
								ValueOverrides: []vzapi.Overrides{
									{
										Values: &apiextensionsv1.JSON{
											Raw: []byte(nameOverrideJSON),
										},
									},
								},
							},
						},
					},
				},
			},
			expectError: true,
		},
		// GIVEN a Verrazzano CR with Jaeger Component enabled and fullNameOverride value set,
		// WHEN we call the ValidateInstall function,
		// THEN an error is returned.
		{
			name: "test jaeger operator override full name",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							Enabled: &trueValue,
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &trueValue,
								ValueOverrides: []vzapi.Overrides{
									{
										Values: &apiextensionsv1.JSON{
											Raw: []byte(fullnameOverrideJSON),
										},
									},
								},
							},
						},
					},
				},
			},
			expectError: true,
		},
		// GIVEN a Verrazzano CR with Jaeger Component enabled and valid override value set,
		// WHEN we call the ValidateInstall function
		// THEN no error is returned.
		{
			name: "test jaeger operator override allowed value",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							Enabled: &trueValue,
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &trueValue,
								ValueOverrides: []vzapi.Overrides{
									{
										Values: &apiextensionsv1.JSON{
											Raw: []byte(validOverrideJSON),
										},
									},
								},
							},
						},
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
	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithRuntimeObjects().Build(), nil
	}
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
	tests := []struct {
		name        string
		oldVZ       vzapi.Verrazzano
		newVZ       vzapi.Verrazzano
		expectError bool
	}{
		// GIVEN a default Verrazzano custom resource with Jaeger Operator component enabled,
		// WHEN we try to update Verrazzano CR to disable Jaeger Component,
		// THEN an error is returned.
		{
			name:        "test disable jaeger operator post installation",
			oldVZ:       oldVZ,
			newVZ:       newVZ,
			expectError: true,
		},
		// GIVEN a default Verrazzano custom resource with Jaeger Operator component enabled,
		// WHEN we try to update Verrazzano CR with no changes,
		// THEN no error is returned.
		{
			name:        "test jaeger operator with no changes",
			oldVZ:       oldVZ,
			newVZ:       oldVZ,
			expectError: false,
		},
		// GIVEN a Verrazzano CR with Jaeger Component enabled and service account name override value set,
		// WHEN we call the ValidateInstall function
		// THEN an error is returned.
		{
			name:  "test jaeger operator override service account name",
			oldVZ: oldVZ,
			newVZ: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							Enabled: &trueValue,
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &trueValue,
								ValueOverrides: []vzapi.Overrides{
									{
										Values: &apiextensionsv1.JSON{
											Raw: []byte(serviceAccountNameJSON),
										},
									},
								},
							},
						},
					},
				},
			},
			expectError: true,
		},
		// GIVEN a Verrazzano CR with Jaeger Component enabled and ingress override value set,
		// WHEN we call the ValidateInstall function
		// THEN an error is returned.
		{
			name:  "test jaeger operator override ingress setting",
			oldVZ: oldVZ,
			newVZ: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							Enabled: &trueValue,
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &trueValue,
								ValueOverrides: []vzapi.Overrides{
									{
										Values: &apiextensionsv1.JSON{
											Raw: []byte(ingressJSON),
										},
									},
								},
							},
						},
					},
				},
			},
			expectError: true,
		},
	}
	c := jaegerOperatorComponent{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateUpdate(&tt.oldVZ, &tt.newVZ)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
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

func getAllJaegerObjects(operatorReplicas, collectorReplicas, queryReplicas int32) []client.Object {
	allJaegerObjects := append(getJaegerOperatorObjects(operatorReplicas), getJaegerCollectorObjects(collectorReplicas)...)
	allJaegerObjects = append(allJaegerObjects, getJaegerQueryObjects(queryReplicas)...)
	return allJaegerObjects
}

// getJaegerOperatorObjects returns the K8S objects for the Jaeger Operator component.
func getJaegerOperatorObjects(availableReplicas int32) []client.Object {
	return []client.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      deploymentName,
				Labels:    map[string]string{k8sAppNameLabel: ComponentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{k8sAppNameLabel: ComponentName},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: availableReplicas,
				ReadyReplicas:     availableReplicas,
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
					k8sAppNameLabel:     ComponentName,
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
	}
}

// getJaegerCollectorObjects returns the K8S objects for the Jaeger Collector component.
func getJaegerCollectorObjects(availableReplicas int32) []client.Object {
	return []client.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      JaegerCollectorDeploymentName,
				Labels:    map[string]string{k8sAppNameLabel: JaegerCollectorDeploymentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{k8sAppNameLabel: JaegerCollectorDeploymentName},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: availableReplicas,
				ReadyReplicas:     availableReplicas,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      JaegerCollectorDeploymentName + "-95d8c4c96-m6ncr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c4c96",
					k8sAppNameLabel:     JaegerCollectorDeploymentName,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        JaegerCollectorDeploymentName + "-95d8c4c96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
	}
}

// getJaegerQueryObjects returns the K8S objects for the Jaeger Query component.
func getJaegerQueryObjects(availableReplicas int32) []client.Object {
	return []client.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      JaegerQueryDeploymentName,
				Labels:    map[string]string{k8sAppNameLabel: JaegerQueryDeploymentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{k8sAppNameLabel: JaegerQueryDeploymentName},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: availableReplicas,
				ReadyReplicas:     availableReplicas,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      JaegerQueryDeploymentName + "-95d8c3b96-m689r",
				Labels: map[string]string{
					"pod-template-hash": "95d8c3b96",
					k8sAppNameLabel:     JaegerQueryDeploymentName,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        JaegerQueryDeploymentName + "-95d8c3b96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
	}
}

// getVZCRWithDefaultJaegerOverride returns VZ with the given Jaeger CR overrides applied
func getVZCRWithDefaultJaegerOverride(jaegerCROverride string) *vzapi.Verrazzano {
	return &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				JaegerOperator: &vzapi.JaegerOperatorComponent{
					Enabled: &enabled,
					InstallOverrides: vzapi.InstallOverrides{
						MonitorChanges: &trueValue,
						ValueOverrides: []vzapi.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(jaegerCROverride),
								},
							},
						},
					},
				},
			},
		},
	}
}
