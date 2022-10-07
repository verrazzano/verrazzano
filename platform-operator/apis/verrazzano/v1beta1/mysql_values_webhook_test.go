// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta1

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

const (
	basePodSpecs = `
{
  "podSpec": {
    "affinity": {
      "podAffinity": {
        "preferredDuringSchedulingIgnoredDuringExecution": [
          {
            "weight": 100,
            "podAffinityTerm": {
              "labelSelector": {
                "matchLabels": {
                  "app.kubernetes.io/instance": "mysql-innodbcluster-mysql-mysql-server",
                  "app.kubernetes.io/name": "mysql-innodbcluster-mysql-server"
                }
              },
              "topologyKey": "kubernetes.io/hostname"
            }
          }
        ]
      }
    }
  },
  "router": {
    "podSpec": {
      "affinity": {
        "podAffinity": {
          "preferredDuringSchedulingIgnoredDuringExecution": [
            {
              "weight": 100,
              "podAffinityTerm": {
                "labelSelector": {
                  "matchLabels": {
                    "app.kubernetes.io/instance": "mysql-innodbcluster-mysql-router",
                    "app.kubernetes.io/name": "mysql-router"
                  }
                },
                "topologyKey": "kubernetes.io/hostname"
              }
            }
          ]
        }
      }
    }
  }
}`
	modifiedServerPodSpec = `
{
  "podSpec": {
    "affinity": {
      "podAffinity": {
        "preferredDuringSchedulingIgnoredDuringExecution": [
          {
            "weight": 200,
            "podAffinityTerm": {
              "labelSelector": {
                "matchLabels": {
                  "app.kubernetes.io/instance": "mysql-innodbcluster-mysql-mysql-server",
                  "app.kubernetes.io/name": "mysql-innodbcluster-mysql-server"
                }
              },
              "topologyKey": "kubernetes.io/hostname"
            }
          }
        ]
      }
    }
  },
  "router": {
    "podSpec": {
      "affinity": {
        "podAffinity": {
          "preferredDuringSchedulingIgnoredDuringExecution": [
            {
              "weight": 100,
              "podAffinityTerm": {
                "labelSelector": {
                  "matchLabels": {
                    "app.kubernetes.io/instance": "mysql-innodbcluster-mysql-router",
                    "app.kubernetes.io/name": "mysql-router"
                  }
                },
                "topologyKey": "kubernetes.io/hostname"
              }
            }
          ]
        }
      }
    }
  }
}`
	modifiedRouterPodSpec = `
{
  "podSpec": {
    "affinity": {
      "podAffinity": {
        "preferredDuringSchedulingIgnoredDuringExecution": [
          {
            "weight": 100,
            "podAffinityTerm": {
              "labelSelector": {
                "matchLabels": {
                  "app.kubernetes.io/instance": "mysql-innodbcluster-mysql-mysql-server",
                  "app.kubernetes.io/name": "mysql-innodbcluster-mysql-server"
                }
              },
              "topologyKey": "kubernetes.io/hostname"
            }
          }
        ]
      }
    }
  },
  "router": {
    "podSpec": {
      "affinity": {
        "podAffinity": {
          "preferredDuringSchedulingIgnoredDuringExecution": [
            {
              "weight": 200,
              "podAffinityTerm": {
                "labelSelector": {
                  "matchLabels": {
                    "app.kubernetes.io/instance": "mysql-innodbcluster-mysql-router",
                    "app.kubernetes.io/name": "mysql-router"
                  }
                },
                "topologyKey": "kubernetes.io/hostname"
              }
            }
          ]
        }
      }
    }
  }
}`

	noServerPodSpec = `
{
  "router": {
    "podSpec": {
      "affinity": {
        "podAffinity": {
          "preferredDuringSchedulingIgnoredDuringExecution": [
            {
              "weight": 100,
              "podAffinityTerm": {
                "labelSelector": {
                  "matchLabels": {
                    "app.kubernetes.io/instance": "mysql-innodbcluster-mysql-router",
                    "app.kubernetes.io/name": "mysql-router"
                  }
                },
                "topologyKey": "kubernetes.io/hostname"
              }
            }
          ]
        }
      }
    }
  }
}`

	noRouterPodSpec = `
{
  "podSpec": {
    "affinity": {
      "podAffinity": {
        "preferredDuringSchedulingIgnoredDuringExecution": [
          {
            "weight": 100,
            "podAffinityTerm": {
              "labelSelector": {
                "matchLabels": {
                  "app.kubernetes.io/instance": "mysql-innodbcluster-mysql-mysql-server",
                  "app.kubernetes.io/name": "mysql-innodbcluster-mysql-server"
                }
              },
              "topologyKey": "kubernetes.io/hostname"
            }
          }
        ]
      }
    }
  }
}`

)

// newMultiClusterApplicationConfigurationValidator creates a new MultiClusterApplicationConfigurationValidator
func newMysqlValuesValidatorWithObjects(initObjs ...client.Object) MysqlValuesValidator {
	scheme := newScheme()
	decoder, _ := admission.NewDecoder(scheme)
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjs...).Build()
	v := MysqlValuesValidator{client: cli, decoder: decoder}
	return v
}

// newAdmissionRequest creates a new admissionRequest with the provided operation and objects.
func newAdmissionRequest(op admissionv1.Operation, obj interface{}, oldObj interface{}) admission.Request {
	raw := runtime.RawExtension{}
	bytes, _ := json.Marshal(obj)
	raw.Raw = bytes
	oldRaw := runtime.RawExtension{}
	oldBytes, _ := json.Marshal(oldObj)
	oldRaw.Raw = oldBytes
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Operation: op, Object: raw, OldObject: oldRaw}}
	return req
}


// TestValidationWarningForServerPodSpec tests presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values specify a server podSpec
// THEN the admission request should be allowed but with a warning.
func TestValidationWarningForServerPodSpec(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorWithObjects()
	oldVz := Verrazzano{
		Spec: VerrazzanoSpec{
			Version:    MIN_VERSION,
			Components: ComponentSpec{
				Keycloak: &KeycloakComponent{
					MySQL: MySQLComponent{
						InstallOverrides: InstallOverrides{
							ValueOverrides: []Overrides{{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(basePodSpecs),
								},
							}},
						},
					},
				},
			},
		},
	}

	newVz := Verrazzano{
		Spec: VerrazzanoSpec{
			Version:    MIN_VERSION,
			Components: ComponentSpec{
				Keycloak: &KeycloakComponent{
					MySQL: MySQLComponent{
						InstallOverrides: InstallOverrides{
							ValueOverrides: []Overrides{{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(modifiedServerPodSpec),
								},
							}},
						},
					},
				},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Update, newVz, oldVz)
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected request to be allowed with warnings")
	asrt.Len(res.Warnings, 1, "Expected there to be one warning")
	asrt.Contains(res.Warnings[0], "Modifications to pod specs do not trigger an automatic restart of a stateful set.", "expected specific warning about stateful set restart")
}

// TestValidationWarningForRouterPodSpec tests presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values specify a router podSpec
// THEN the admission request should be allowed but with a warning.
func TestValidationWarningForRouterPodSpec(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorWithObjects()
	oldVz := Verrazzano{
		Spec: VerrazzanoSpec{
			Version:    MIN_VERSION,
			Components: ComponentSpec{
				Keycloak: &KeycloakComponent{
					MySQL: MySQLComponent{
						InstallOverrides: InstallOverrides{
							ValueOverrides: []Overrides{{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(basePodSpecs),
								},
							}},
						},
					},
				},
			},
		},
	}

	newVz := Verrazzano{
		Spec: VerrazzanoSpec{
			Version:    MIN_VERSION,
			Components: ComponentSpec{
				Keycloak: &KeycloakComponent{
					MySQL: MySQLComponent{
						InstallOverrides: InstallOverrides{
							ValueOverrides: []Overrides{{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(modifiedRouterPodSpec),
								},
							}},
						},
					},
				},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Update, newVz, oldVz)
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected request to be allowed with warnings")
	asrt.Len(res.Warnings, 1, "Expected there to be one warning")
	asrt.Contains(res.Warnings[0], "Modifications to pod specs do not trigger an automatic restart of a stateful set.", "expected specific warning about stateful set restart")
}

// TestNoValidationWarningWithoutServerPodSpec tests not presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values do not specify a server podSpec
// THEN the admission request should be allowed
func TestNoValidationWarningWithoutServerPodSpec(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorWithObjects()
	oldVz := Verrazzano{
		Spec: VerrazzanoSpec{
			Version:    MIN_VERSION,
			Components: ComponentSpec{
				Keycloak: &KeycloakComponent{
					MySQL: MySQLComponent{
						InstallOverrides: InstallOverrides{
							ValueOverrides: []Overrides{{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(basePodSpecs),
								},
							}},
						},
					},
				},
			},
		},
	}

	newVz := Verrazzano{
		Spec: VerrazzanoSpec{
			Version:    MIN_VERSION,
			Components: ComponentSpec{
				Keycloak: &KeycloakComponent{
					MySQL: MySQLComponent{
						InstallOverrides: InstallOverrides{
							ValueOverrides: []Overrides{{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(noServerPodSpec),
								},
							}},
						},
					},
				},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Update, newVz, oldVz)
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected request to be allowed with warnings")
	asrt.Len(res.Warnings, 0, "Expected there to be no warnings")
}

// TestNoValidationWarningWithoutRouterPodSpec tests not presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values do not specify a router podSpec
// THEN the admission request should be allowed
func TestNoValidationWarningWithoutRouterPodSpec(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorWithObjects()
	oldVz := Verrazzano{
		Spec: VerrazzanoSpec{
			Version:    MIN_VERSION,
			Components: ComponentSpec{
				Keycloak: &KeycloakComponent{
					MySQL: MySQLComponent{
						InstallOverrides: InstallOverrides{
							ValueOverrides: []Overrides{{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(basePodSpecs),
								},
							}},
						},
					},
				},
			},
		},
	}

	newVz := Verrazzano{
		Spec: VerrazzanoSpec{
			Version:    MIN_VERSION,
			Components: ComponentSpec{
				Keycloak: &KeycloakComponent{
					MySQL: MySQLComponent{
						InstallOverrides: InstallOverrides{
							ValueOverrides: []Overrides{{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(noRouterPodSpec),
								},
							}},
						},
					},
				},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Update, newVz, oldVz)
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected request to be allowed with warnings")
	asrt.Len(res.Warnings, 0, "Expected there to be no warnings")
}
