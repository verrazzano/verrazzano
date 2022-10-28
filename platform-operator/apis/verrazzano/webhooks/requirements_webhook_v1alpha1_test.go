// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

// newRequirementsValidatorV1alpha1 creates a new MultiClusterConfigmapValidator
func newRequirementsValidatorV1alpha1(objects []client.Object) RequirementsValidatorV1alpha1 {
	scheme := newV1alpha1Scheme()
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	decoder, _ := admission.NewDecoder(scheme)
	v := RequirementsValidatorV1alpha1{client: client, decoder: decoder}
	return v
}

// TestPrerequisiteValidationWarningForV1alpha1 tests presenting a user warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the nodes do not meet the prerequisites
// THEN the admission request should be allowed but with a warning.
func TestPrerequisiteValidationWarningForV1alpha1(t *testing.T) {
	asrt := assert.New(t)
	var nodes []client.Object
	nodes = append(nodes, node("node1", "1", "32G", "40G"))
	m := newRequirementsValidatorV1alpha1(nodes)
	vz := &v1alpha1.Verrazzano{Spec: v1alpha1.VerrazzanoSpec{Profile: v1alpha1.Dev}}
	req := newAdmissionRequest(admissionv1.Update, vz, vz)
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, allowedFailureMessage)
	asrt.Len(res.Warnings, 2, expectedWarningFailureMessage)
	asrt.Contains(res.Warnings[0], "minimum required CPUs for dev profile is 2 but the CPUs on node node1 is 1")
	asrt.Contains(res.Warnings[1], "minimum required ephemeral storage for dev profile is 100G but the ephemeral storage on node node1 is 40G")
}

// TestPrerequisiteValidationNoWarningForV1alpha1 tests presenting a user with no warning
// GIVEN a call to validate a Verrazzano resource
// WHEN the nodes meet the prerequisites
// THEN the admission request should be allowed without a warning.
func TestPrerequisiteValidationNoWarningForV1alpha1(t *testing.T) {
	asrt := assert.New(t)
	var nodes []client.Object
	nodes = append(nodes, node("node1", "4", "32G", "140G"), node("node2", "5", "32G", "140G"), node("node3", "6", "32G", "140G"))
	m := newRequirementsValidatorV1alpha1(nodes)
	vz := &v1alpha1.Verrazzano{Spec: v1alpha1.VerrazzanoSpec{Profile: v1alpha1.Prod}}
	req := newAdmissionRequest(admissionv1.Update, vz, vz)
	res := m.Handle(context.TODO(), req)
	asrt.True(res.Allowed, allowedFailureMessage)
	asrt.Len(res.Warnings, 0, noWarningsFailureMessage)
}

func node(name string, cpu string, memory string, ephemeralStorage string) *v1.Node {
	return &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: v1.NodeStatus{
			Allocatable: v1.ResourceList{
				"cpu":               resource.MustParse(cpu),
				"memory":            resource.MustParse(memory),
				"ephemeral-storage": resource.MustParse(ephemeralStorage),
			},
		},
	}
}
