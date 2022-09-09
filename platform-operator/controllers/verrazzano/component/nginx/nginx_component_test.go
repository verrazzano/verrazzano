// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"context"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/test/ip"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testExternalIP = ip.RandomIP()

const (
	invalidExternalIPOverrideJSON = `
{
	"controller": {
		"service": {
			"externalIPs": ["1231"]
		}
	}
}
`
	validExternalIPOverrideJSON = `
{
	"controller": {
		"service": {
			"externalIPs": ["1.1.1.1"]
		}
	}
}
`
)

func Test_nginxComponent_ValidateUpdate(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "change-type-to-nodeport-without-externalIPs",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: vzapi.NodePort,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "change-type-to-nodeport-with-externalIPs",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: vzapi.NodePort,
							NGINXInstallArgs: []vzapi.InstallArgs{
								{
									Name:      nginxExternalIPKey,
									ValueList: []string{testExternalIP},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-type-from-nodeport",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: vzapi.NodePort,
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: vzapi.LoadBalancer,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-install-args",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							NGINXInstallArgs: []vzapi.InstallArgs{{Name: "foo", Value: "bar"}},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-ports",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Ports: []corev1.ServicePort{{Name: "https2", NodePort: 30057}},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
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

func Test_nginxComponent_ValidateUpdateV1Beta1(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *v1beta1.Verrazzano
		new     *v1beta1.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						IngressNGINX: &v1beta1.IngressNginxComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						IngressNGINX: &v1beta1.IngressNginxComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "change-type-to-nodeport-without-externalIPs",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						IngressNGINX: &v1beta1.IngressNginxComponent{
							Type: v1beta1.NodePort,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-type-to-nodeport-with-externalIPs-valid",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						IngressNGINX: &v1beta1.IngressNginxComponent{
							Type: v1beta1.NodePort,
							InstallOverrides: v1beta1.InstallOverrides{
								ValueOverrides: []v1beta1.Overrides{
									{
										Values: &apiextensionsv1.JSON{
											Raw: []byte(validExternalIPOverrideJSON),
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-type-to-nodeport-with-externalIPs-invalid",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						IngressNGINX: &v1beta1.IngressNginxComponent{
							Type: v1beta1.NodePort,
							InstallOverrides: v1beta1.InstallOverrides{
								ValueOverrides: []v1beta1.Overrides{
									{
										Values: &apiextensionsv1.JSON{
											Raw: []byte(invalidExternalIPOverrideJSON),
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "change-type-from-nodeport",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						IngressNGINX: &v1beta1.IngressNginxComponent{
							Type: v1beta1.NodePort,
						},
					},
				},
			},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						IngressNGINX: &v1beta1.IngressNginxComponent{
							Type: v1beta1.LoadBalancer,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-ports",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						IngressNGINX: &v1beta1.IngressNginxComponent{
							Ports: []corev1.ServicePort{{Name: "https2", NodePort: 30057}},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "no change",
			old:     &v1beta1.Verrazzano{},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
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

func Test_nginxComponent_ValidateInstall(t *testing.T) {
	tests := []struct {
		name    string
		vz      *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "NginxInstallArgsEmpty",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: vzapi.NodePort,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "NginxInstallArgsMissingKey",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: vzapi.NodePort,
							NGINXInstallArgs: []vzapi.InstallArgs{
								{
									Name:      "foo",
									ValueList: []string{testExternalIP},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "NginxInstallArgsMissingIP",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: vzapi.NodePort,
							NGINXInstallArgs: []vzapi.InstallArgs{
								{
									Name:  nginxExternalIPKey,
									Value: "1.1.1.1.1",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "NginxInstallArgsMissingIPInList",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: vzapi.NodePort,
							NGINXInstallArgs: []vzapi.InstallArgs{
								{
									Name:      nginxExternalIPKey,
									ValueList: []string{""},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "NginxInstallArgsInvalidIP",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: vzapi.NodePort,
							NGINXInstallArgs: []vzapi.InstallArgs{
								{Name: nginxExternalIPKey},
								{ValueList: []string{testExternalIP + ".1"}},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "NginxInstallArgsValidConfig",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: vzapi.NodePort,
							NGINXInstallArgs: []vzapi.InstallArgs{
								{
									Name:      nginxExternalIPKey,
									ValueList: []string{testExternalIP},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateInstall(tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_nginxComponent_ValidateInstallV1Beta1(t *testing.T) {
	tests := []struct {
		name    string
		vz      *v1beta1.Verrazzano
		wantErr bool
	}{
		{
			name: "NginxInstallOverridesEmpty",
			vz: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						IngressNGINX: &v1beta1.IngressNginxComponent{
							Type: v1beta1.NodePort,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "NginxInstallOverridesInvalid",
			vz: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						IngressNGINX: &v1beta1.IngressNginxComponent{
							Type: v1beta1.NodePort,
							InstallOverrides: v1beta1.InstallOverrides{
								ValueOverrides: []v1beta1.Overrides{
									{
										Values: &apiextensionsv1.JSON{
											Raw: []byte(invalidExternalIPOverrideJSON),
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "NginxInstallOverridesValid",
			vz: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						IngressNGINX: &v1beta1.IngressNginxComponent{
							Type: v1beta1.NodePort,
							InstallOverrides: v1beta1.InstallOverrides{
								ValueOverrides: []v1beta1.Overrides{
									{
										Values: &apiextensionsv1.JSON{
											Raw: []byte(validExternalIPOverrideJSON),
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateInstallV1Beta1(tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstallV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPostUninstall tests the PostUninstall function
// GIVEN a call to PostUninstall
//
//	WHEN the ingress-nginx namespace exists with a finalizer
//	THEN true is returned and ingress-nginx namespace is deleted
func TestPostUninstall(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:       ComponentNamespace,
				Finalizers: []string{"fake-finalizer"},
			},
		},
	).Build()

	var nComp nginxComponent
	compContext := spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)
	assert.NoError(t, nComp.PostUninstall(compContext))

	// Validate that the namespace does not exist
	ns := corev1.Namespace{}
	err := compContext.Client().Get(context.TODO(), types.NamespacedName{Name: ComponentNamespace}, &ns)
	assert.True(t, errors.IsNotFound(err))
}
