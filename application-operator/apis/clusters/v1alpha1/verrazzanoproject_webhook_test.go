// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"bytes"
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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"
	"testing"
	"text/template"
)

var testProject = VerrazzanoProject{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	},
	Spec: VerrazzanoProjectSpec{
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
	// Test data
	testVP := testProject

	// Test create
	err := testVP.ValidateCreate()
	assert.NoError(t, err, "Error validating VerrazzanoMultiCluster resource")

	// Test update
	err = testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.NoError(t, err, "Error validating VerrazzanoMultiCluster resource")
}

// TestInvalidNamespace tests the validation of VerrazzanoProject resource
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject contains an invalid namespace
// THEN the validation should fail
func TestInvalidNamespace(t *testing.T) {

	// Test data
	testVP := testProject
	testVP.Namespace = "invalid-namespace"

	// Test create
	err := testVP.ValidateCreate()
	assert.Error(t, err, "Expected failure for invalid namespace")
	assert.Containsf(t, err.Error(), fmt.Sprintf("resource must be %q", constants.VerrazzanoMultiClusterNamespace), "unexpected failure string")

	// Test update
	err = testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.Error(t, err, "Expected failure for invalid namespace")
	assert.Containsf(t, err.Error(), fmt.Sprintf("resource must be %q", constants.VerrazzanoMultiClusterNamespace), "unexpected failure string")
}

// TestInvalidNamespaces tests the validation of VerrazzanoProject resource
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject contains an invalid namespace list
// THEN the validation should fail
func TestInvalidNamespaces(t *testing.T) {

	// Test data
	testVP := testProject
	testVP.Spec.Template.Namespaces = []NamespaceTemplate{}

	// Test create
	err := testVP.ValidateCreate()
	assert.Error(t, err, "Expected failure for invalid namespace list")
	assert.Containsf(t, err.Error(), "One or more namespaces must be provided", "unexpected failure string")

	// Test update
	err = testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.Error(t, err, "Expected failure for invalid namespace list")
	assert.Containsf(t, err.Error(), "One or more namespaces must be provided", "unexpected failure string")
}

// TestNetworkPolicyNamespace tests the validation of VerrazzanoProject NetworkPolicyTemplate
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject has a NetworkPolicyTemplate with a namespace that exists in the project
// THEN the validation should succeed
func TestNetworkPolicyNamespace(t *testing.T) {
	// Test data
	testVP := testNetworkPolicy

	// Test create
	err := testVP.ValidateCreate()
	assert.NoError(t, err, "Error validating VerrazzanProject with NetworkPolicyTemplate")

	// Test update
	err = testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.NoError(t, err, "Error validating VerrazzanProject with NetworkPolicyTemplate")
}

// TestNetworkPolicyMissingNamespace tests the validation of VerrazzanoProject NetworkPolicyTemplate
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject has a NetworkPolicyTemplate with a namespace that does not exist in the project
// THEN the validation should fail
func TestNetworkPolicyMissingNamespace(t *testing.T) {
	// Test data
	testVP := testNetworkPolicy
	testVP.Spec.Template.Namespaces[0].Metadata.Name = "ns2"

	// Test create
	err := testVP.ValidateCreate()
	assert.EqualError(t, err, "namespace ns1 used in NetworkPolicy net1 does not exist in project", "Error validating VerrazzanProject with NetworkPolicyTemplate")

	// Test update
	err = testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.EqualError(t, err, "namespace ns1 used in NetworkPolicy net1 does not exist in project", "Error validating VerrazzanProject with NetworkPolicyTemplate")
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
			Name: "test-cluser-name",
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
	asrt.True(res.Allowed, "Expected project validation to succeed.")
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

// newTestProject creates a new VerrazzanoProject with the provided name and target namespace.
func newTestProject(projectName string, targetNamespace string) VerrazzanoProject {
	return VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectName,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: targetNamespace,
						},
					},
				},
			},
		},
	}
}

//	vmc := `
//apiVersion: clusters.verrazzano.io/v1alpha1
//kind: VerrazzanoManagedCluster
//metadata:
//  name: managed1
//  namespace: verrazzano-mc
//spec:
//  managedClusterManifestSecret: verrazzano-cluster-managed1-manifest
//  prometheusSecret: prometheus-managed1
//  serviceAccount: verrazzano-cluster-managed1`
//	asrt.NoError(createResourceFromTemplate(cli, vmc, map[string]string{}))

// executeTemplate reads a template from a file and replaces values in the template from param maps
// template - The filename of a template
// params - a vararg of param maps
func executeTemplate(s string, d interface{}) (string, error) {
	t, err := template.New(s).Parse(s)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = t.ExecuteTemplate(&buf, s, d)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// updateUnstructuredFromYAMLTemplate updates an unstructured from a populated YAML template file.
// uns - The unstructured to update
// template - The template file
// params - The param maps to merge into the template
func updateUnstructuredFromYAMLTemplate(uns *unstructured.Unstructured, template string, data interface{}) error {
	str, err := executeTemplate(template, data)
	if err != nil {
		return err
	}
	bytes, err := yaml.YAMLToJSON([]byte(str))
	if err != nil {
		return err
	}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(bytes, nil, uns)
	if err != nil {
		return err
	}
	return nil
}

// createResourceFromTemplate builds a resource by merging the data with the template file and then
// creates the resource using the client.
func createResourceFromTemplate(cli client.Client, template string, data interface{}) error {
	uns := unstructured.Unstructured{}
	if err := updateUnstructuredFromYAMLTemplate(&uns, template, data); err != nil {
		return err
	}
	if err := cli.Create(context.TODO(), &uns); err != nil {
		return err
	}
	return nil
}

