// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	admissionv1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

// newRequirementsValidatorV1beta1 creates a new RequirementsValidatorV1beta1
func newRequirementsValidatorV1beta1(objects []client.Object) RequirementsValidatorV1beta1 {
	scheme := newScheme()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	decoder, _ := admission.NewDecoder(scheme)
	v := RequirementsValidatorV1beta1{client: c, decoder: decoder}
	return v
}

// TestPrerequisiteValidationWarningForV1beta1 tests presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the nodes do not meet the prerequisites
// THEN the admission request should be allowed but with a warning.
func TestPrerequisiteValidationWarningForV1beta1(t *testing.T) {
	asrt := assert.New(t)
	var nodes []client.Object
	nodes = append(nodes, node("node1", "3", "16G", "400G"))
	m := newRequirementsValidatorV1beta1(nodes)
	vz := &v1beta1.Verrazzano{Spec: v1beta1.VerrazzanoSpec{Profile: v1beta1.Prod}}
	req := newAdmissionRequest(admissionv1.Update, vz, vz)
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, allowedFailureMessage)
	asrt.Len(res.Warnings, 2, expectedWarningFailureMessage)
	asrt.Contains(res.Warnings[0], "minimum required CPUs is 4 but the CPUs on node node1 is 3")
	asrt.Contains(res.Warnings[1], "minimum required memory is 32G but the memory on node node1 is 16G")
}

// TestPrerequisiteValidationNoWarningForV1beta1 tests presenting a user with no warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the nodes meet the prerequisites
// THEN the admission request should be allowed without a warning.
func TestPrerequisiteValidationNoWarningForV1beta1(t *testing.T) {
	asrt := assert.New(t)
	var nodes []client.Object
	nodes = append(nodes, node("node1", "3", "16G", "100G"), node("node2", "5", "32G", "140G"))
	m := newRequirementsValidatorV1beta1(nodes)
	vz := &v1beta1.Verrazzano{Spec: v1beta1.VerrazzanoSpec{Profile: v1beta1.Dev}}
	req := newAdmissionRequest(admissionv1.Update, vz, vz)
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, allowedFailureMessage)
	asrt.Len(res.Warnings, 0, noWarningsFailureMessage)
}
