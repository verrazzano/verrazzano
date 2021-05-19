// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

var testManagedCluster = v1alpha1.VerrazzanoManagedCluster{
	ObjectMeta: metav1.ObjectMeta{
		Name: "test-managed-cluster-name",
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	},
	Spec:       v1alpha1.VerrazzanoManagedClusterSpec{
		PrometheusSecret: "test-prometheus-secret",
		ManagedClusterManifestSecret: "test-cluster-manifest-secret",
		ServiceAccount: "test-service-account",
	},
}

var testProject = VerrazzanoProject{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	},
	Spec: VerrazzanoProjectSpec{
		Placement: Placement{
			Clusters: []Cluster{{Name: "test-managed-cluster-name"}},
		},
		Template: ProjectTemplate{
			Namespaces: []NamespaceTemplate{
				{
					Metadata: metav1.ObjectMeta{
						Name: "newNS1",
					},
				},
			},
		},
	},
}

var testNetworkPolicy = VerrazzanoProject{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	},
	Spec: VerrazzanoProjectSpec{
		Placement: Placement{
			Clusters: []Cluster{{Name: "test-managed-cluster-name"}},
		},
		Template: ProjectTemplate{
			Namespaces: []NamespaceTemplate{
				{
					Metadata: metav1.ObjectMeta{
						Name: "ns1",
					},
				},
			},
			NetworkPolicies: []NetworkPolicyTemplate{
				{
					Metadata: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "net1",
					},
					Spec: netv1.NetworkPolicySpec{},
				}},
		},
	},
}

// TestVerrazzanoProject tests the validation of VerrazzanoProject resource
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject is properly formed
// THEN the validation should succeed
func TestVerrazzanoProject(t *testing.T) {
	asrt := assert.New(t)
	v := newVerrazzanoProjectValidator()

	// Test data
	testVP := testProject
	testMC := testManagedCluster
	asrt.NoError(v.client.Create(context.TODO(), &testMC))

	req := newAdmissionRequest(admissionv1beta1.Create, testVP)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected project validation to succeed.")

	req = newAdmissionRequest(admissionv1beta1.Update, testVP)
	res = v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected project validation to succeed.")
}

// TestInvalidNamespace tests the validation of VerrazzanoProject resource
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject contains an invalid namespace
// THEN the validation should fail
func TestInvalidNamespace(t *testing.T) {
	asrt := assert.New(t)
	v := newVerrazzanoProjectValidator()

	// Test data
	testVP := testProject
	testVP.Namespace = "invalid-namespace"
	testMC := testManagedCluster
	asrt.NoError(v.client.Create(context.TODO(), &testMC))

	// Test create
	req := newAdmissionRequest(admissionv1beta1.Create, testVP)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project create validation to fail.")
	asrt.Containsf(res.Result.Reason, fmt.Sprintf("resource must be %q", constants.VerrazzanoMultiClusterNamespace), "unexpected failure string")

	// Test update
	req = newAdmissionRequest(admissionv1beta1.Update, testVP)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project update validation to fail.")
	asrt.Containsf(res.Result.Reason, fmt.Sprintf("resource must be %q", constants.VerrazzanoMultiClusterNamespace), "unexpected failure string")
}

// TestInvalidNamespaces tests the validation of VerrazzanoProject resource
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject contains an invalid namespace list
// THEN the validation should fail
func TestInvalidNamespaces(t *testing.T) {
	asrt := assert.New(t)
	v := newVerrazzanoProjectValidator()

	// Test data
	testVP := testProject
	testVP.Spec.Template.Namespaces = []NamespaceTemplate{}
	testMC := testManagedCluster
	asrt.NoError(v.client.Create(context.TODO(), &testMC))

	// Test create
	req := newAdmissionRequest(admissionv1beta1.Create, testVP)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project create validation to fail for invalid namespace list")
	asrt.Containsf(res.Result.Reason, fmt.Sprintf("One or more namespaces must be provided"), "unexpected failure string")

	// Test update
	req = newAdmissionRequest(admissionv1beta1.Update, testVP)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project create validation to fail for invalid namespace list")
	asrt.Containsf(res.Result.Reason, fmt.Sprintf("One or more namespaces must be provided"), "unexpected failure string")
}

// TestNetworkPolicyNamespace tests the validation of VerrazzanoProject NetworkPolicyTemplate
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject has a NetworkPolicyTemplate with a namespace that exists in the project
// THEN the validation should succeed
func TestNetworkPolicyNamespace(t *testing.T) {
	asrt := assert.New(t)
	v := newVerrazzanoProjectValidator()

	// Test data
	testVP := testNetworkPolicy
	testMC := testManagedCluster
	asrt.NoError(v.client.Create(context.TODO(), &testMC))

	// Test create
	req := newAdmissionRequest(admissionv1beta1.Create, testVP)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Error validating VerrazzanProject with NetworkPolicyTemplate")

	// Test update
	req = newAdmissionRequest(admissionv1beta1.Update, testVP)
	res = v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Error validating VerrazzanProject with NetworkPolicyTemplate")
}

// TestNetworkPolicyMissingNamespace tests the validation of VerrazzanoProject NetworkPolicyTemplate
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject has a NetworkPolicyTemplate with a namespace that does not exist in the project
// THEN the validation should fail
func TestNetworkPolicyMissingNamespace(t *testing.T) {
	asrt := assert.New(t)
	v := newVerrazzanoProjectValidator()

	// Test data
	testVP := testNetworkPolicy
	testVP.Spec.Template.Namespaces[0].Metadata.Name = "ns2"
	testMC := testManagedCluster
	asrt.NoError(v.client.Create(context.TODO(), &testMC))

	// Test create
	req := newAdmissionRequest(admissionv1beta1.Create, testVP)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project validation to fail due to missing network policy")
	asrt.Containsf(res.Result.Reason, "namespace ns1 used in NetworkPolicy net1 does not exist in project", "Error validating VerrazzanProject with NetworkPolicyTemplate")

	// Test update
	req = newAdmissionRequest(admissionv1beta1.Update, testVP)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project validation to fail due to missing network policy")
	asrt.Containsf(res.Result.Reason, "namespace ns1 used in NetworkPolicy net1 does not exist in project", "Error validating VerrazzanProject with NetworkPolicyTemplate")
}

// TestNamespaceUniquenessForProjects tests that the namespace of a VerrazzanoProject N does not conflict with a preexisting project
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject has a a namespace that conflicts with any pre-existing projects
// THEN the validation should fail
func TestNamespaceUniquenessForProjects(t *testing.T) {

	// When creating the fake client, prepopulate it with 2 Verrazzano projects
	// existingVP1 has namespaces project1 and project2
	// existingVP2 has namespaces project3 and project4
	// Adding any new Verrazzano projects with these namespaces will fail validation
	existingVP1 := &VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-project-1",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project1",
						},
					},
					{
						Metadata: metav1.ObjectMeta{
							Name: "project2",
						},
					},
				},
			},
		},
	}

	existingVP2 := &VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-project-2",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project3",
						},
					},
					{
						Metadata: metav1.ObjectMeta{
							Name: "project4",
						},
					},
				},
			},
		},
	}

	objs := []runtime.Object{existingVP1, existingVP2}
	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme(), objs...), nil
	}
	defer func() { getControllerRuntimeClient = getClient }()

	currentVP := VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project",
						},
					},
				},
			},
		},
	}
	// This test will succeed because Verrazzano project test-project has unique namespace project
	err := currentVP.validateNamespaceCanBeUsed()
	assert.Nil(t, err)

	currentVP = VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project1",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project2",
						},
					},
				},
			},
		},
	}

	// This test will fail because Verrazzano project test-project1 has conflicting namespace project2
	err = currentVP.validateNamespaceCanBeUsed()
	assert.NotNil(t, err)
	// This test will fail same as above but this time coming in through parent validator
	err = currentVP.validateVerrazzanoProject()
	assert.NotNil(t, err)

	currentVP = VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project2",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project",
						},
					},
					{
						Metadata: metav1.ObjectMeta{
							Name: "project4",
						},
					},
				},
			},
		},
	}
	// UPDATE FAIL This test will fail because Verrazzano project test-project2 has conflicting namespace project4
	err = currentVP.validateNamespaceCanBeUsed()
	assert.NotNil(t, err)

	currentVP = VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-project-1",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project",
						},
					},
					{
						Metadata: metav1.ObjectMeta{
							Name: "project4",
						},
					},
				},
			},
		},
	}
	// This test will fail because Verrazzano project name, existing-project-1, is using a namespace in existing-project-2
	err = currentVP.validateNamespaceCanBeUsed()
	assert.NotNil(t, err)

	currentVP = VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-project-1",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project",
						},
					},
					{
						Metadata: metav1.ObjectMeta{
							Name: "project2",
						},
					},
				},
			},
		},
	}
	// UPDATE PASS This test will pass because Verrazzano project name, existing-project-1, is not using a namespace associated with any existing projects
	err = currentVP.validateNamespaceCanBeUsed()
	assert.Nil(t, err)
}

// TestValidationFailureForProjectCreationWithoutTargetClusters tests preventing the creation
// of a VerrazzanoProject resources that is missing Placement information.
// GIVEN a call to validate a VerrazzanoProject resource
// WHEN the VerrazzanoProject resource is missing Placement information
// THEN the validation should fail.
func TestValidationFailureForProjectCreationWithoutTargetClusters(t *testing.T) {
	asrt := assert.New(t)
	v := newVerrazzanoProjectValidator()
	p := VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "test-target-namespace",
						},
					},
				},
			},
		},
	}
	req := newAdmissionRequest(admissionv1beta1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project validation to fail due to missing placement information.")
}

// TestValidationFailureForProjectCreationTargetingMissingManagedCluster tests preventing the creation
// of a VerrazzanoProject resources that references a non-existent managed cluster.
// GIVEN a call to validate a VerrazzanoProject resource
// WHEN the VerrazzanoProject resource references a VerrazzanoManagedCluster that does not exist
// THEN the validation should fail.
func TestValidationFailureForProjectCreationTargetingMissingManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newVerrazzanoProjectValidator()
	p := VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Placement: Placement{
				Clusters: []Cluster{{Name: "invalid-cluster-name"}},
			},
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "test-target-namespace",
						},
					},
				},
			},
		},
	}
	req := newAdmissionRequest(admissionv1beta1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project validation to fail due to missing placement information.")
}

// TestValidationSuccessForProjectCreationTargetingExistingManagedCluster tests allowing the creation
// of a VerrazzanoProject resources that references an existent managed cluster.
// GIVEN a call to validate a VerrazzanoProject resource
// WHEN the VerrazzanoProject resource references a VerrazzanoManagedCluster that does exist
// THEN the validation should pass.
func TestValidationSuccessForProjectCreationTargetingExistingManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newVerrazzanoProjectValidator()
	c := v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "valid-cluster-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec:       v1alpha1.VerrazzanoManagedClusterSpec{
			PrometheusSecret: "test-prometheus-secret",
			ManagedClusterManifestSecret: "test-cluster-manifest-secret",
			ServiceAccount: "test-service-account",
		},
	}
	p := VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Placement: Placement{
				Clusters: []Cluster{{Name: "valid-cluster-name"}},
			},
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "test-target-namespace",
						},
					},
				},
			},
		},
	}
	asrt.NoError(v.client.Create(context.TODO(), &c))
	req := newAdmissionRequest(admissionv1beta1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected project create validation to succeed.")
}

// TestValidationSuccessForProjectCreationWithoutTargetClustersOnManagedCluster tests allowing the creation
// of a VerrazzanoProject resources that is missing target cluster information when on managed cluster.
// GIVEN a call to validate a VerrazzanoProject resource
// WHEN the VerrazzanoProject resource is missing Placement information
// AND the validation is being done on a managed cluster
// THEN the validation should succeed.
func TestValidationSuccessForProjectCreationWithoutTargetClustersOnManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newVerrazzanoProjectValidator()
	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "verrazzano-cluster-registration",
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	p := VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Placement: Placement{
				Clusters: []Cluster{{Name: "invalid-cluster-name"}},
			},
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "test-target-namespace",
						},
					},
				},
			},
		},
	}
	asrt.NoError(v.client.Create(context.TODO(), &s))
	req := newAdmissionRequest(admissionv1beta1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected project validation to succeed withmissing placement information on managed cluster.")
}

// newVerrazzanoProjectValidator creates a new VerrazzanoProjectValidator
func newVerrazzanoProjectValidator() VerrazzanoProjectValidator {
	scheme := newScheme()
	decoder, _ := admission.NewDecoder(scheme)
	cli := fake.NewFakeClientWithScheme(scheme)
	v := VerrazzanoProjectValidator{client: cli, decoder: decoder}
	return v
}

// newAdmissionRequest creates a new admissionRequest with the provided operation and object.
func newAdmissionRequest(op admissionv1beta1.Operation, obj interface{}) admission.Request{
	raw := runtime.RawExtension{}
	bytes, _ := json.Marshal(obj)
	raw.Raw = bytes
	req := admission.Request{
		admissionv1beta1.AdmissionRequest{
			Operation: op, Object: raw}}
	return req
}

// newScheme creates a new scheme that includes this package's object for use by client
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	AddToScheme(scheme)
	scheme.AddKnownTypes(schema.GroupVersion{
		Version: "v1",
	}, &corev1.Secret{})
	v1alpha1.AddToScheme(scheme)
	return scheme
}
