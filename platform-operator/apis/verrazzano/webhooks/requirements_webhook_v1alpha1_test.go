// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
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
	asrt.Contains(res.Warnings[0], "minimum required CPUs is 2 but the CPUs on node node1 is 1")
	asrt.Contains(res.Warnings[1], "minimum required ephemeral storage is 100G but the ephemeral storage on node node1 is 40G")
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

// TestValidateInstallOS tests presenting a user with a warning related to number of OS nodes
// GIVEN a call to validate a Verrazzano resource
// WHEN the nodes don't meet the prerequisites
// THEN the admission request should be allowed with a warning.
func TestValidateInstallOS(t *testing.T) {
	asrt := assert.New(t)
	trueVal := true
	tests := []struct {
		name    string
		vzCR    *v1alpha1.Verrazzano
		wantStr string
	}{
		{ //The master nodes are less then 3 in a multi node cluster and so the appropriate warning should be displayed
			name: "case-1",
			vzCR: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Elasticsearch: &v1alpha1.ElasticsearchComponent{
							Enabled: &(trueVal),
							Nodes:   []v1alpha1.OpenSearchNode{{Name: "node1", Replicas: 1, Roles: []vmov1.NodeRole{vmov1.DataRole, vmov1.MasterRole}}, {Name: "node2", Replicas: 1, Roles: []vmov1.NodeRole{vmov1.MasterRole, vmov1.DataRole}}, {Name: "node3", Replicas: 8, Roles: []vmov1.NodeRole{vmov1.IngestRole}}},
						},
					},
				},
			},
			wantStr: "Number of master nodes should be at least 3 in a multi node cluster",
		},
		{ //The data nodes are less then 2 in a multi node cluster and so the appropriate warning should be displayed
			name: "case-2",
			vzCR: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Elasticsearch: &v1alpha1.ElasticsearchComponent{
							Enabled: &(trueVal),
							Nodes:   []v1alpha1.OpenSearchNode{{Name: "node1", Replicas: 1, Roles: []vmov1.NodeRole{vmov1.DataRole}}, {Name: "node2", Replicas: 3, Roles: []vmov1.NodeRole{vmov1.MasterRole}}, {Name: "node3", Replicas: 1, Roles: []vmov1.NodeRole{vmov1.IngestRole}}},
						},
					},
				},
			},
			wantStr: "Number of data nodes should be at least 2 in a multi node cluster",
		},
		{ //The ingest nodes are less then 1 in a multi node cluster and so the appropriate warning should be displayed
			name: "case-3",
			vzCR: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Elasticsearch: &v1alpha1.ElasticsearchComponent{
							Enabled: &(trueVal),
							Nodes:   []v1alpha1.OpenSearchNode{{Name: "node1", Replicas: 1, Roles: []vmov1.NodeRole{vmov1.DataRole, vmov1.MasterRole}}, {Name: "node2", Replicas: 2, Roles: []vmov1.NodeRole{vmov1.MasterRole, vmov1.DataRole}}, {Name: "node3", Replicas: 0, Roles: []vmov1.NodeRole{vmov1.IngestRole}}},
						},
					},
				},
			},
			wantStr: "Number of ingest nodes should be at least 1 in a multi node cluster",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nodes []client.Object
			nodes = append(nodes, node("node1", "4", "32G", "140G"), node("node2", "5", "32G", "140G"), node("node3", "6", "32G", "140G"))
			m := newRequirementsValidatorV1alpha1(nodes)
			req := newAdmissionRequest(admissionv1.Update, tt.vzCR, tt.vzCR)
			res := m.Handle(context.TODO(), req)
			asrt.Contains(res.Warnings[0], tt.wantStr)
		})
	}
}

// TestValidateUpdateOS tests presenting a user with a warning related to scaling of OS nodes
// GIVEN a call to validate a Verrazzano resource
// WHEN the nodes don't meet the prerequisites
// THEN the admission request should be allowed with a warning.
func TestValidateUpdateOS(t *testing.T) {
	asrt := assert.New(t)
	trueVal := true
	tests := []struct {
		name    string
		old     *v1alpha1.Verrazzano
		new     *v1alpha1.Verrazzano
		wantStr string
	}{
		{ //The master nodes are being scaled down by more than half and so appropriate warning should be given
			name: "case-1",
			old: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Elasticsearch: &v1alpha1.ElasticsearchComponent{
							Enabled: &(trueVal),
							Nodes:   []v1alpha1.OpenSearchNode{{Name: "node1", Replicas: 5, Roles: []vmov1.NodeRole{vmov1.DataRole}}, {Name: "node2", Replicas: 7, Roles: []vmov1.NodeRole{vmov1.MasterRole}}, {Name: "node3", Replicas: 1, Roles: []vmov1.NodeRole{vmov1.IngestRole}}},
						},
					},
				},
			},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Elasticsearch: &v1alpha1.ElasticsearchComponent{
							Enabled: &(trueVal),
							Nodes:   []v1alpha1.OpenSearchNode{{Name: "node1", Replicas: 1, Roles: []vmov1.NodeRole{vmov1.DataRole, vmov1.MasterRole}}, {Name: "node2", Replicas: 2, Roles: []vmov1.NodeRole{vmov1.MasterRole, vmov1.DataRole}}, {Name: "node3", Replicas: 8, Roles: []vmov1.NodeRole{vmov1.IngestRole}}},
						},
					},
				},
			},
			wantStr: "The number of master nodes shouldn't be scaled down by more than half at once",
		},
		{ //The data nodes are being scaled down by more than half and so appropriate warning should be given
			name: "case-2",
			old: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Elasticsearch: &v1alpha1.ElasticsearchComponent{
							Enabled: &(trueVal),
							Nodes:   []v1alpha1.OpenSearchNode{{Name: "node1", Replicas: 5, Roles: []vmov1.NodeRole{vmov1.DataRole}}, {Name: "node2", Replicas: 7, Roles: []vmov1.NodeRole{vmov1.MasterRole}}, {Name: "node3", Replicas: 1, Roles: []vmov1.NodeRole{vmov1.IngestRole}}},
						},
					},
				},
			},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Elasticsearch: &v1alpha1.ElasticsearchComponent{
							Enabled: &(trueVal),
							Nodes:   []v1alpha1.OpenSearchNode{{Name: "node1", Replicas: 2, Roles: []vmov1.NodeRole{vmov1.DataRole, vmov1.MasterRole}}, {Name: "node2", Replicas: 2, Roles: []vmov1.NodeRole{vmov1.MasterRole}}, {Name: "node3", Replicas: 1, Roles: []vmov1.NodeRole{vmov1.IngestRole}}},
						},
					},
				},
			},
			wantStr: "The number of data nodes shouldn't be scaled down by more than half at once",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nodes []client.Object
			nodes = append(nodes, node("node1", "4", "32G", "140G"), node("node2", "5", "32G", "140G"), node("node3", "6", "32G", "140G"))
			m := newRequirementsValidatorV1alpha1(nodes)
			req := newAdmissionRequest(admissionv1.Update, tt.new, tt.old)
			res := m.Handle(context.TODO(), req)
			asrt.Contains(res.Warnings[0], tt.wantStr)
		})
	}
}
