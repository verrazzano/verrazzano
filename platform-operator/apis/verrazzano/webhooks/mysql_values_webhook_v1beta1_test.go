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
	replicasOverrides = `
{
  "routerInstances": 3,
  "serverInstances": 3
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
	v := MysqlValuesValidatorV1beta1{decoder: decoder, BomVersion: MinVersion}
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
func newAdmissionRequest(op admissionv1.Operation, obj interface{}, oldObj interface{}) admission.Request {
	objRaw := encodeRawBytes(obj)
	oldObjRaw := encodeRawBytes(oldObj)
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Operation: op, Object: objRaw, OldObject: oldObjRaw}}
	return req
}

func encodeRawBytes(obj interface{}) runtime.RawExtension {
	raw := runtime.RawExtension{}
	bytes, _ := json.Marshal(obj)
	raw.Raw = bytes
	return raw
}

// TestValidationWarningForServerPodSpecV1beta1 tests presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values specify a server podSpec
// THEN the admission request should be allowed but with a warning.
func TestValidationWarningForServerPodSpecV1beta1(t *testing.T) {
	tests := []struct {
		name       string
		shouldWarn bool
		newVz      *v1beta1.Verrazzano
		oldVz      *v1beta1.Verrazzano
	}{
		{
			name:       "No Version Set",
			shouldWarn: true,
			newVz:      &v1beta1.Verrazzano{},
			oldVz:      &v1beta1.Verrazzano{},
		},
		{
			name:       "New version set, Old Version Not Set",
			shouldWarn: true,
			newVz:      &v1beta1.Verrazzano{Spec: v1beta1.VerrazzanoSpec{Version: MinVersion}},
			oldVz:      &v1beta1.Verrazzano{},
		},
		{
			name:       "New version NOT set, Old Status Version Set",
			shouldWarn: true,
			newVz:      &v1beta1.Verrazzano{Spec: v1beta1.VerrazzanoSpec{}},
			oldVz:      &v1beta1.Verrazzano{Status: v1beta1.VerrazzanoStatus{Version: MinVersion}},
		},
		{
			name:       "New version NOT set, Old Status Version Set",
			shouldWarn: true,
			newVz:      &v1beta1.Verrazzano{Spec: v1beta1.VerrazzanoSpec{}},
			oldVz:      &v1beta1.Verrazzano{Status: v1beta1.VerrazzanoStatus{Version: MinVersion}},
		},
		{
			name:       "New version set below min version",
			shouldWarn: false,
			newVz:      &v1beta1.Verrazzano{Spec: v1beta1.VerrazzanoSpec{Version: "v1.4.1"}},
			oldVz:      &v1beta1.Verrazzano{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Test: %s", test.name)
			asrt := assert.New(t)
			m := newMysqlValuesValidatorV1alpha1()
			test.newVz.Spec.Components.Keycloak = &v1beta1.KeycloakComponent{
				MySQL: v1beta1.MySQLComponent{
					InstallOverrides: v1beta1.InstallOverrides{
						ValueOverrides: []v1beta1.Overrides{{
							Values: &apiextensionsv1.JSON{
								Raw: []byte(modifiedServerPodSpec),
							},
						}},
					},
				},
			}
			req := newAdmissionRequest(admissionv1.Update, test.newVz, test.oldVz)
			res := m.Handle(context.TODO(), req)
			asrt.True(res.Allowed, allowedFailureMessage)
			if test.shouldWarn {
				asrt.Len(res.Warnings, 1, expectedWarningFailureMessage)
				asrt.Contains(res.Warnings[0], "Modifications to MySQL server pod specs do not trigger an automatic restart of the stateful set.", "expected specific warning about stateful set restart")
			} else {
				asrt.Len(res.Warnings, 0, noWarningsFailureMessage)
			}
		})
	}
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
			Version: MinVersion,
			Components: v1beta1.ComponentSpec{
				Keycloak: &v1beta1.KeycloakComponent{
					MySQL: v1beta1.MySQLComponent{
						InstallOverrides: v1beta1.InstallOverrides{
							ValueOverrides: []v1beta1.Overrides{
								{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(replicasOverrides),
									},
								},
								{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(modifiedRouterPodSpec),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Update, newVz, &v1beta1.Verrazzano{})
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, allowedFailureMessage)
	asrt.Len(res.Warnings, 0, noWarningsFailureMessage)
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
			Version: MinVersion,
			Components: v1beta1.ComponentSpec{
				Keycloak: &v1beta1.KeycloakComponent{
					MySQL: v1beta1.MySQLComponent{
						InstallOverrides: v1beta1.InstallOverrides{
							ValueOverrides: []v1beta1.Overrides{
								{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(replicasOverrides),
									},
								},
								{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(noPodSpec),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Update, newVz, &v1beta1.Verrazzano{})
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, allowedFailureMessage)
	asrt.Len(res.Warnings, 0, noWarningsFailureMessage)
}

// TestNoValidationWarningWithEmptyVZ object tests not presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values do not specify a server podSpec
// THEN the admission request should be allowed with ""

func TestNoValidationWarningWithEmptyVZ(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorV1beta1()
	newVz := v1beta1.Verrazzano{}
	req := newAdmissionRequest(admissionv1.Create, newVz, &v1beta1.Verrazzano{})
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, allowedFailureMessage)
	asrt.Len(res.Warnings, 0, noWarningsFailureMessage)
}
