// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

func Test_certManagerComponent_ValidateUpdate(t *testing.T) {
	disabled := false
	var tests = []struct {
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
						CertManager: &vzapi.CertManagerComponent{
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
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "update",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								CA: vzapi.CA{
									SecretName:               "newsecret",
									ClusterResourceNamespace: "ns",
								},
							},
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
		{
			name: "updateInvalidBothConfigured",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								CA: vzapi.CA{
									SecretName:               "newsecret",
									ClusterResourceNamespace: "ns",
								},
								Acme: vzapi.Acme{
									Provider:     "letsencrypt",
									EmailAddress: "joeblow@foo.com",
									Environment:  "staging",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "validLetsEncryptStaging",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     "letsencrypt",
									EmailAddress: "joeblow@foo.com",
									Environment:  "staging",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "validLetsEncryptProviderCaseInsensitivity",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     "LETSENCRYPT",
									EmailAddress: "joeblow@foo.com",
									Environment:  "staging",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "validLetsEncryptStagingCaseInsensitivity",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     vzapi.LetsEncrypt,
									EmailAddress: "joeblow@foo.com",
									Environment:  "STAGING",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "validLetsEncryptProdCaseInsensitivity",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     vzapi.LetsEncrypt,
									EmailAddress: "joeblow@foo.com",
									Environment:  "PRODUCTION",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "validLetsEncryptDefaultStagingEnv",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     vzapi.LetsEncrypt,
									EmailAddress: "joeblow@foo.com",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "validLetsEncryptProd",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     "letsencrypt",
									EmailAddress: "joeblow@foo.com",
									Environment:  "production",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalidACMEProvider",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     "blah",
									EmailAddress: "joeblow@foo.com",
									Environment:  "production",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalidLetsEncryptEnv",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     "letsencrypt",
									EmailAddress: "joeblow@foo.com",
									Environment:  "myenv",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalidACMEEmail",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     "letsencrypt",
									EmailAddress: "joeblow",
									Environment:  "staging",
								},
							},
						},
					},
				},
			},
			wantErr: true,
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
