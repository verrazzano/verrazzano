// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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
			name: "change-type",
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
							Ports: []v1.ServicePort{{Name: "https2", NodePort: 30057}},
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
									ValueList: []string{"1.1.1.1"},
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
									Name:  "controller.service.externalIPs",
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
									Name:      "controller.service.externalIPs",
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
								{Name: "controller.service.externalIPs"},
								{ValueList: []string{"1.1.1.1.1"}},
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
									Name:      "controller.service.externalIPs",
									ValueList: []string{"1.1.1.1"},
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
