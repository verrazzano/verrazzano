// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

// newMysqlValuesValidatorV1alpha1 creates a new MysqlValuesValidatorV1alpha1
func newMysqlValuesValidatorV1alpha1() MysqlValuesValidatorV1alpha1 {
	scheme := newV1alpha1Scheme()
	decoder, _ := admission.NewDecoder(scheme)
	v := MysqlValuesValidatorV1alpha1{BomVersion: MinVersion}
	v.InjectDecoder(decoder)
	return v
}

// newV1alpha1Scheme creates a new scheme that includes this package's object for use by client
func newV1alpha1Scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	clientgoscheme.AddToScheme(scheme)
	return scheme
}

// TestValidationWarningForServerPodSpec tests presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values specify a server podSpec
// THEN the admission request should be allowed but with a warning.
func TestValidationWarningForServerPodSpecV1alpha1(t *testing.T) {
	tests := []struct {
		name       string
		shouldWarn bool
		newVz      *v1alpha1.Verrazzano
		oldVz      *v1alpha1.Verrazzano
	}{
		{
			name:       "No Version Set",
			shouldWarn: true,
			newVz:      &v1alpha1.Verrazzano{},
			oldVz:      &v1alpha1.Verrazzano{},
		},
		{
			name:       "New version set, Old Version Not Set",
			shouldWarn: true,
			newVz:      &v1alpha1.Verrazzano{Spec: v1alpha1.VerrazzanoSpec{Version: MinVersion}},
			oldVz:      &v1alpha1.Verrazzano{},
		},
		{
			name:       "New version NOT set, Old Status Version Set",
			shouldWarn: true,
			newVz:      &v1alpha1.Verrazzano{Spec: v1alpha1.VerrazzanoSpec{}},
			oldVz:      &v1alpha1.Verrazzano{Status: v1alpha1.VerrazzanoStatus{Version: MinVersion}},
		},
		{
			name:       "New version NOT set, Old Status Version Set",
			shouldWarn: true,
			newVz:      &v1alpha1.Verrazzano{Spec: v1alpha1.VerrazzanoSpec{}},
			oldVz:      &v1alpha1.Verrazzano{Status: v1alpha1.VerrazzanoStatus{Version: MinVersion}},
		},
		{
			name:       "New version set below min version",
			shouldWarn: false,
			newVz:      &v1alpha1.Verrazzano{Spec: v1alpha1.VerrazzanoSpec{Version: "v1.4.1"}},
			oldVz:      &v1alpha1.Verrazzano{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Test: %s", test.name)
			asrt := assert.New(t)
			m := newMysqlValuesValidatorV1alpha1()
			test.newVz.Spec.Components.Keycloak = &v1alpha1.KeycloakComponent{
				MySQL: v1alpha1.MySQLComponent{
					InstallOverrides: v1alpha1.InstallOverrides{
						ValueOverrides: []v1alpha1.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(replicasOverrides),
								},
							},
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(modifiedServerPodSpec),
								},
							},
						},
					},
				},
			}
			req := newAdmissionRequest(admissionv1.Update, test.newVz, test.oldVz)
			res := m.Handle(context.TODO(), req)
			asrt.True(res.Allowed, "Expected request to be allowed")
			if test.shouldWarn {
				asrt.Len(res.Warnings, 1, "Expected there to be one warning")
				asrt.Contains(res.Warnings[0], "Modifications to MySQL server pod specs do not trigger an automatic restart of the stateful set.", "expected specific warning about stateful set restart")
			} else {
				asrt.Len(res.Warnings, 0, "Expected there to be no warnings")
			}
		})
	}
}

// TestNoValidationWarningForRouterPodSpecV1alpha1 tests presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values specify a router podSpec
// THEN the admission request should be allowed with no warning.
func TestNoValidationWarningForRouterPodSpecV1alpha1(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorV1alpha1()
	newVz := v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Version: MinVersion,
			Components: v1alpha1.ComponentSpec{
				Keycloak: &v1alpha1.KeycloakComponent{
					MySQL: v1alpha1.MySQLComponent{
						InstallOverrides: v1alpha1.InstallOverrides{
							ValueOverrides: []v1alpha1.Overrides{{
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

	req := newAdmissionRequest(admissionv1.Update, newVz, &v1alpha1.Verrazzano{})
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected request to be allowed with warnings")
	asrt.Len(res.Warnings, 0, "Expected there to be one warning")
}

// TestNoValidationWarningWithoutServerPodSpec tests not presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the override values do not specify a server podSpec
// THEN the admission request should be allowed
func TestNoValidationWarningWithoutServerPodSpecV1alpha1(t *testing.T) {
	asrt := assert.New(t)
	m := newMysqlValuesValidatorV1alpha1()
	newVz := v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Version: MinVersion,
			Components: v1alpha1.ComponentSpec{
				Keycloak: &v1alpha1.KeycloakComponent{
					MySQL: v1alpha1.MySQLComponent{
						InstallOverrides: v1alpha1.InstallOverrides{
							ValueOverrides: []v1alpha1.Overrides{
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

	req := newAdmissionRequest(admissionv1.Update, newVz, &v1alpha1.Verrazzano{})
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected request to be allowed with warnings")
	asrt.Len(res.Warnings, 0, "Expected there to be one warning")
}
