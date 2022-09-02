// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

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

func Test_externalDNSComponent_ValidateUpdate(t *testing.T) {
	tests := []struct {
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
							External: &vzapi.External{Suffix: "foo.com"},
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
							External: &vzapi.External{Suffix: "foo.com"},
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

func Test_externalDNSComponent_ValidateUpdateV1beta1(t *testing.T) {
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
							External: &v1beta1.External{Suffix: "foo.com"},
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
							External: &v1beta1.External{Suffix: "foo.com"},
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
