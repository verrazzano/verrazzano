// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	admissionv1 "k8s.io/api/admission/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
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

// newMysqlValuesValidatorV1beta1 creates a new MysqlValuesValidatorV1beta1
func newMysqlValuesValidatorV1beta1() MysqlValuesValidatorV1beta1 {
	scheme := newScheme()
	decoder, _ := admission.NewDecoder(scheme)
	v := MysqlValuesValidatorV1beta1{decoder: decoder}
	return v
}

// newV1alpha1Scheme creates a new scheme that includes this package's object for use by client
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	v1beta1.AddToScheme(scheme)
	clientgoscheme.AddToScheme(scheme)
	return scheme
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

// TestValidationWarningForServerPodSpecV1beta1 tests presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values specify a server podSpec
// THEN the admission request should be allowed but with a warning.
func TestValidationWarningForServerPodSpecV1beta1(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorV1beta1()

	newVz := v1beta1.Verrazzano{
		Spec: v1beta1.VerrazzanoSpec{
			Version: MIN_VERSION,
			Components: v1beta1.ComponentSpec{
				Keycloak: &v1beta1.KeycloakComponent{
					MySQL: v1beta1.MySQLComponent{
						InstallOverrides: v1beta1.InstallOverrides{
							ValueOverrides: []v1beta1.Overrides{{
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

// TestNoValidationWarningForRouterPodSpecV1beta1 tests presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values specify a router podSpec
// THEN the admission request should be allowed with no warning.
func TestNoValidationWarningForRouterPodSpecV1beta1(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorV1beta1()
	newVz := v1beta1.Verrazzano{
		Spec: v1beta1.VerrazzanoSpec{
			Version: MIN_VERSION,
			Components: v1beta1.ComponentSpec{
				Keycloak: &v1beta1.KeycloakComponent{
					MySQL: v1beta1.MySQLComponent{
						InstallOverrides: v1beta1.InstallOverrides{
							ValueOverrides: []v1beta1.Overrides{{
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
func TestNoValidationWarningWithoutServerPodSpecV1beta1(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorV1beta1()
	newVz := v1beta1.Verrazzano{
		Spec: v1beta1.VerrazzanoSpec{
			Version: MIN_VERSION,
			Components: v1beta1.ComponentSpec{
				Keycloak: &v1beta1.KeycloakComponent{
					MySQL: v1beta1.MySQLComponent{
						InstallOverrides: v1beta1.InstallOverrides{
							ValueOverrides: []v1beta1.Overrides{{
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
