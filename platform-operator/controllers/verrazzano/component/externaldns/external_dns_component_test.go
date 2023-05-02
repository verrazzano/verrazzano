// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	corev1Cli "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const fooDomainSuffix = "foo.com"

// TestExternalDNSPostUninstall tests the PostUninstall fn
// GIVEN a call to PostUninstall
// WHEN ExternalDNS is installed
// THEN no errors are returned and the related resources are cleaned up
func TestExternalDNSPostUninstall(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).
		WithObjects(
			&rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName},
			},
			&rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{Name: clusterRoleBindingName},
			},
		).
		Build()

	// Verify they're there before PostInstall
	assert.NoError(t, client.Get(context.TODO(), types.NamespacedName{Name: clusterRoleName}, &rbacv1.ClusterRole{}))
	assert.NoError(t, client.Get(context.TODO(), types.NamespacedName{Name: clusterRoleName}, &rbacv1.ClusterRoleBinding{}))

	ednsComp := NewComponent()
	err := ednsComp.PostUninstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)

	// Verify they're gone after PostInstall
	err = client.Get(context.TODO(), types.NamespacedName{Name: clusterRoleName}, &rbacv1.ClusterRole{})
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
	err = client.Get(context.TODO(), types.NamespacedName{Name: clusterRoleName}, &rbacv1.ClusterRoleBinding{})
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
}

func TestValidateUpdate(t *testing.T) {
	var tests = []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
		{
			name: "disable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			new:     &vzapi.Verrazzano{},
			wantErr: true,
		},
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "default-to-external",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							External: &vzapi.External{Suffix: fooDomainSuffix},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
		{
			name: "oci-to-external",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							External: &vzapi.External{Suffix: fooDomainSuffix},
						},
					},
				},
			},
			wantErr: true, // For now, any changes to the DNS component are rejected
		},
		{
			name: "oci-to-wildcard",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							Wildcard: &vzapi.Wildcard{Domain: "xip.io"},
						},
					},
				},
			},
			wantErr: true, // For now, any changes to the DNS component are rejected
		},
		{
			name: "default-to-wildcard",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							Wildcard: &vzapi.Wildcard{Domain: "xip.io"},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
		{
			name: "wildcard-to-wildcard",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							Wildcard: &vzapi.Wildcard{Domain: "sslip.io"},
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							Wildcard: &vzapi.Wildcard{Domain: "xip.io"},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdateV1beta1(t *testing.T) {
	tests := []struct {
		name    string
		old     *v1beta1.Verrazzano
		new     *v1beta1.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						DNS: &v1beta1.DNSComponent{
							OCI: &v1beta1.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
		{
			name: "disable",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						DNS: &v1beta1.DNSComponent{
							OCI: &v1beta1.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			new:     &v1beta1.Verrazzano{},
			wantErr: true,
		},
		{
			name:    "no change",
			old:     &v1beta1.Verrazzano{},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
		{
			name: "default-to-external",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						DNS: &v1beta1.DNSComponent{
							External: &v1beta1.External{Suffix: fooDomainSuffix},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
		{
			name: "oci-to-external",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						DNS: &v1beta1.DNSComponent{
							OCI: &v1beta1.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						DNS: &v1beta1.DNSComponent{
							External: &v1beta1.External{Suffix: fooDomainSuffix},
						},
					},
				},
			},
			wantErr: true, // For now, any changes to the DNS component are rejected
		},
		{
			name: "oci-to-wildcard",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						DNS: &v1beta1.DNSComponent{
							OCI: &v1beta1.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						DNS: &v1beta1.DNSComponent{
							Wildcard: &v1beta1.Wildcard{Domain: "xip.io"},
						},
					},
				},
			},
			wantErr: true, // For now, any changes to the DNS component are rejected
		},
		{
			name: "default-to-wildcard",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						DNS: &v1beta1.DNSComponent{
							Wildcard: &v1beta1.Wildcard{Domain: "xip.io"},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
		{
			name: "wildcard-to-wildcard",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						DNS: &v1beta1.DNSComponent{
							Wildcard: &v1beta1.Wildcard{Domain: "sslip.io"},
						},
					},
				},
			},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						DNS: &v1beta1.DNSComponent{
							Wildcard: &v1beta1.Wildcard{Domain: "xip.io"},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdateV1Beta1(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIsAvailableDefaultNamespace tests the IsAvailable fn
// GIVEN a call to IsAvailable
// WHEN ExternalDNS is installed in the default ns and its pod is ready
// THEN it is reported as available
func TestIsAvailableDefaultNamespace(t *testing.T) {
	ns := ComponentNamespace

	k8sutil.GetCoreV1Func = func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset(
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
					Labels: map[string]string{
						constants.VerrazzanoManagedKey: ns,
					},
				},
			},
		).CoreV1(), nil
	}
	isLegacyNamespaceInstalledFunc = func(_ string, _ string) (found bool, err error) {
		return false, nil
	}
	defer func() {
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		isLegacyNamespaceInstalledFunc = helm.IsReleaseInstalled
	}()

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		newDeployment(ComponentName, ns, true), newPod(ComponentName, ns), newReplicaSet(ComponentName, ns),
	).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	_, availability := NewComponent().IsAvailable(ctx)
	assert.Equal(t, vzapi.ComponentAvailable, string(availability))
}

// TestIsAvailableLegacy tests the IsAvailable fn
// GIVEN a call to IsAvailable
// WHEN ExternalDNS is installed in the legacy ns and its pod is ready
// THEN it is reported as available
func TestIsAvailableLegacy(t *testing.T) {
	ns := legacyNamespace
	k8sutil.GetCoreV1Func = func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset(
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
					Labels: map[string]string{
						constants.VerrazzanoManagedKey: ns,
					},
				},
			},
		).CoreV1(), nil
	}
	isLegacyNamespaceInstalledFunc = func(_ string, _ string) (found bool, err error) {
		return true, nil
	}
	defer func() {
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		isLegacyNamespaceInstalledFunc = helm.IsReleaseInstalled
	}()

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		newDeployment(ComponentName, ns, true), newPod(ComponentName, ns), newReplicaSet(ComponentName, ns),
	).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	_, availability := NewComponent().IsAvailable(ctx)
	assert.Equal(t, vzapi.ComponentAvailable, string(availability))
}

// TestDryRun tests the behavior when DryRun is enabled, mainly for code coverage
// GIVEN a call to overridden Component methods with a ComponentContext
//
//	WHEN the ComponentContext has DryRun set to true
//	THEN no error is returned
func TestDryRun(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, true)

	assert.NoError(t, fakeComponent.PreInstall(ctx))
	assert.True(t, fakeComponent.IsReady(ctx))
	assert.NoError(t, fakeComponent.PostUninstall(ctx))
}

// TestMonitorOverrides tests the MonitorOverrides fn
// GIVEN a call to MonitorOverrides
// THEN it returns true if DNS is defined or the value of DNS.MonitorChanges is true
func TestMonitorOverrides(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()

	cr := &vzapi.Verrazzano{}
	ctx := spi.NewFakeContext(client, cr, nil, true)
	assert.False(t, fakeComponent.MonitorOverrides(ctx))

	cr.Spec.Components.DNS = &vzapi.DNSComponent{}
	ctx = spi.NewFakeContext(client, cr, nil, true)
	assert.True(t, fakeComponent.MonitorOverrides(ctx))

	falseVal := false
	cr.Spec.Components.DNS.MonitorChanges = &falseVal
	ctx = spi.NewFakeContext(client, cr, nil, true)
	assert.False(t, fakeComponent.MonitorOverrides(ctx))

	trueVal := true
	cr.Spec.Components.DNS.MonitorChanges = &trueVal
	ctx = spi.NewFakeContext(client, cr, nil, true)
	assert.True(t, fakeComponent.MonitorOverrides(ctx))
}
