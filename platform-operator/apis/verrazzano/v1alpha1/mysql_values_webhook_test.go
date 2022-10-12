// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

const (
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
  }
}`
	modifiedRouterPodSpec = `
{
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

	noPodSpec = `
{
  "router": {
    "tls": {
      "useSelfSigned": true
	}
  }
}`
)

// newMultiClusterApplicationConfigurationValidator creates a new MultiClusterApplicationConfigurationValidator
func newMysqlValuesValidatorWithObjects(initObjs ...client.Object) MysqlValuesValidator {
	scheme := newScheme()
	decoder, _ := admission.NewDecoder(scheme)
	v := MysqlValuesValidator{decoder: decoder}
	return v
}

// newAdmissionRequest creates a new admissionRequest with the provided operation and objects.
func newAdmissionRequest(op admissionv1.Operation, obj interface{}) admission.Request {
	raw := runtime.RawExtension{}
	bytes, _ := json.Marshal(obj)
	raw.Raw = bytes
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Operation: op, Object: raw}}
	return req
}

// TestValidationWarningForServerPodSpec tests presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values specify a server podSpec
// THEN the admission request should be allowed but with a warning.
func TestValidationWarningForServerPodSpec(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorWithObjects()

	newVz := Verrazzano{
		Spec: VerrazzanoSpec{
			Version: MIN_VERSION,
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

	req := newAdmissionRequest(admissionv1.Update, newVz)
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected request to be allowed with warnings")
	asrt.Len(res.Warnings, 1, "Expected there to be one warning")
	asrt.Contains(res.Warnings[0], "Modifications to MySQL server pod specs do not trigger an automatic restart of the stateful set.", "expected specific warning about stateful set restart")
}

// TestValidationWarningForRouterPodSpec tests presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values specify a router podSpec
// THEN the admission request should be allowed but with a warning.
func TestValidationWarningForRouterPodSpec(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorWithObjects()
	newVz := Verrazzano{
		Spec: VerrazzanoSpec{
			Version: MIN_VERSION,
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

	req := newAdmissionRequest(admissionv1.Update, newVz)
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected request to be allowed with warnings")
	asrt.Len(res.Warnings, 0, "Expected there to be one warning")
}

// TestNoValidationWarningWithoutServerPodSpec tests not presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values do not specify a server podSpec
// THEN the admission request should be allowed
func TestNoValidationWarningWithoutServerPodSpec(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorWithObjects()
	newVz := Verrazzano{
		Spec: VerrazzanoSpec{
			Version: MIN_VERSION,
			Components: ComponentSpec{
				Keycloak: &KeycloakComponent{
					MySQL: MySQLComponent{
						InstallOverrides: InstallOverrides{
							ValueOverrides: []Overrides{{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(noPodSpec),
								},
							}},
						},
					},
				},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Update, newVz)
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected request to be allowed with warnings")
	asrt.Len(res.Warnings, 0, "Expected there to be one warning")
}
