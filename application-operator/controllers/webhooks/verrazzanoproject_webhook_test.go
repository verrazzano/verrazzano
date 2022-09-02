// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	v1alpha12 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// newVerrazzanoProjectValidator creates a new VerrazzanoProjectValidator
func newVerrazzanoProjectValidator() VerrazzanoProjectValidator {
	scheme := newScheme()
	decoder, _ := admission.NewDecoder(scheme)
	cli := fake.NewClientBuilder().WithScheme(scheme).Build()
	v := VerrazzanoProjectValidator{client: cli, decoder: decoder}
	return v
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

	req := newAdmissionRequest(admissionv1.Create, testVP)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected project validation to succeed.")

	req = newAdmissionRequest(admissionv1.Update, testVP)
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
	req := newAdmissionRequest(admissionv1.Create, testVP)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project create validation to fail.")
	asrt.Containsf(res.Result.Reason, fmt.Sprintf("resource must be %q", constants.VerrazzanoMultiClusterNamespace), "unexpected failure string")

	// Test update
	req = newAdmissionRequest(admissionv1.Update, testVP)
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
	testVP.Spec.Template.Namespaces = []v1alpha12.NamespaceTemplate{}
	testMC := testManagedCluster
	asrt.NoError(v.client.Create(context.TODO(), &testMC))

	// Test create
	req := newAdmissionRequest(admissionv1.Create, testVP)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project create validation to fail for invalid namespace list")
	asrt.Containsf(res.Result.Reason, "One or more namespaces must be provided", "unexpected failure string")

	// Test update
	req = newAdmissionRequest(admissionv1.Update, testVP)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project create validation to fail for invalid namespace list")
	asrt.Containsf(res.Result.Reason, "One or more namespaces must be provided", "unexpected failure string")
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
	req := newAdmissionRequest(admissionv1.Create, testVP)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Error validating VerrazzanProject with NetworkPolicyTemplate")

	// Test update
	req = newAdmissionRequest(admissionv1.Update, testVP)
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
	req := newAdmissionRequest(admissionv1.Create, testVP)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project validation to fail due to missing network policy")
	asrt.Containsf(res.Result.Reason, "namespace ns1 used in NetworkPolicy net1 does not exist in project", "Error validating VerrazzanProject with NetworkPolicyTemplate")

	// Test update
	req = newAdmissionRequest(admissionv1.Update, testVP)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project validation to fail due to missing network policy")
	asrt.Containsf(res.Result.Reason, "namespace ns1 used in NetworkPolicy net1 does not exist in project", "Error validating VerrazzanProject with NetworkPolicyTemplate")
}

// TestNamespaceUniquenessForProjects tests that the namespace of a VerrazzanoProject N does not conflict with a preexisting project
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject has a a namespace that conflicts with any pre-existing projects
// THEN the validation should fail
func TestNamespaceUniquenessForProjects(t *testing.T) {

	v := newVerrazzanoProjectValidator()

	// When creating the fake client, prepopulate it with 2 Verrazzano projects
	// existingVP1 has namespaces project1 and project2
	// existingVP2 has namespaces project3 and project4
	// Adding any new Verrazzano projects with these namespaces will fail validation
	existingVP1 := &v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-project-1",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
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
	assert.NoError(t, v.client.Create(context.TODO(), existingVP1))

	existingVP2 := &v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-project-2",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
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
	assert.NoError(t, v.client.Create(context.TODO(), existingVP2))

	currentVP := v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
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
	err := validateNamespaceCanBeUsed(v.client, &currentVP)
	assert.Nil(t, err)

	currentVP = v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project1",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
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
	err = validateNamespaceCanBeUsed(v.client, &currentVP)
	assert.NotNil(t, err)
	// This test will fail same as above but this time coming in through parent validator
	err = validateVerrazzanoProject(v.client, &currentVP)
	assert.NotNil(t, err)

	currentVP = v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project2",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
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
	err = validateNamespaceCanBeUsed(v.client, &currentVP)
	assert.NotNil(t, err)

	currentVP = v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-project-1",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
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
	err = validateNamespaceCanBeUsed(v.client, &currentVP)
	assert.NotNil(t, err)

	currentVP = v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-project-1",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
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
	err = validateNamespaceCanBeUsed(v.client, &currentVP)
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
	p := v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "test-target-namespace",
						},
					},
				},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "target cluster")

	req = newAdmissionRequest(admissionv1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "target cluster")
}

// TestValidationFailureForProjectCreationTargetingMissingManagedCluster tests preventing the creation
// of a VerrazzanoProject resources that references a non-existent managed cluster.
// GIVEN a call to validate a VerrazzanoProject resource
// WHEN the VerrazzanoProject resource references a VerrazzanoManagedCluster that does not exist
// THEN the validation should fail.
func TestValidationFailureForProjectCreationTargetingMissingManagedCluster(t *testing.T) {

	asrt := assert.New(t)
	v := newVerrazzanoProjectValidator()
	p := v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: "invalid-cluster-name"}},
			},
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "test-target-namespace",
						},
					},
				},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "invalid-cluster-name")

	req = newAdmissionRequest(admissionv1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected project validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "invalid-cluster-name")
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
			Name:      "valid-cluster-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha1.VerrazzanoManagedClusterSpec{
			CASecret:                     "test-secret",
			ManagedClusterManifestSecret: "test-cluster-manifest-secret",
			ServiceAccount:               "test-service-account",
		},
	}
	p := v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: "valid-cluster-name"}},
			},
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
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

	req := newAdmissionRequest(admissionv1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected project create validation to succeed.")

	req = newAdmissionRequest(admissionv1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected project update validation to succeed.")
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
			Name:      constants.MCRegistrationSecret,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	p := v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: "invalid-cluster-name"}},
			},
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
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

	req := newAdmissionRequest(admissionv1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected project validation to succeed with missing placement information on managed cluster.")

	req = newAdmissionRequest(admissionv1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected project validation to succeed with missing placement information on managed cluster.")
}

// TestValidationSuccessForProjectCreationTargetingLocalCluster tests allowing the creation
// of a VerrazzanoProject resources on the local admin cluster.
// GIVEN a call to validate a VerrazzanoProject resource
// WHEN the VerrazzanoProject resource targets the local cluster via Placement information
// AND the validation is being done on the admin cluster
// THEN the validation should succeed.
func TestValidationSuccessForProjectCreationTargetingLocalCluster(t *testing.T) {

	asrt := assert.New(t)
	v := newVerrazzanoProjectValidator()
	p := v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: constants.DefaultClusterName}},
			},
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "test-target-namespace",
						},
					},
				},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected project validation to succeed with placement targeting local cluster.")

	req = newAdmissionRequest(admissionv1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected project validation to succeed with placement targeting local cluster.")
}

// TestVzProjHandleFailed tests to make sure the failure metric is being exposed
// GIVEN a call to validate a VerrazzanoProject resource
// WHEN the VerrazzanoProject resource is failing
// THEN the validation should fail.
func TestVzProjHandleFailed(t *testing.T) {

	assert := assert.New(t)
	// Create a request and Handle
	v := newVerrazzanoProjectValidator()
	// Test data
	testVP := testProject
	req := newAdmissionRequest(admissionv1.Create, testVP)
	v.Handle(context.TODO(), req)
	reconcileerrorCounterObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.VzProjHandleError)
	assert.NoError(err)
	// Expect a call to fetch the error
	reconcileFailedCounterBefore := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	reconcileerrorCounterObject.Get().Inc()
	reconcileFailedCounterAfter := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	assert.Equal(reconcileFailedCounterBefore, reconcileFailedCounterAfter-1)
}
