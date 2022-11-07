// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil_test

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	objects  = "./testdata/objects"
	testdata = "./testdata"
)

func TestApplyD(t *testing.T) {
	var tests = []struct {
		name    string
		dir     string
		count   int
		isError bool
	}{
		{
			"should apply YAML files",
			objects,
			3,
			false,
		},
		{
			"should fail to apply non-existent directories",
			"blahblah",
			0,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
			y := k8sutil.NewYAMLApplier(c, "")
			err := y.ApplyD(tt.dir)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.count, len(y.Objects()))
			}
		})
	}
}

func TestApplyF(t *testing.T) {
	var tests = []struct {
		name    string
		file    string
		count   int
		isError bool
	}{
		{
			"should apply file",
			objects + "/service.yaml",
			1,
			false,
		},
		{
			"should apply file with two objects",
			testdata + "/two_objects.yaml",
			2,
			false,
		},
		{
			"should fail to apply files that are not YAML",
			"blahblah",
			0,
			true,
		},
		{
			"should fail when file is not YAML",
			objects + "/not-yaml.txt",
			0,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
			y := k8sutil.NewYAMLApplier(c, "test")
			err := y.ApplyF(tt.file)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.count, len(y.Objects()))
		})
	}
}

// TestApplyFNonSpec
// GIVEN a object that contains top level fields outside of spec
//
//	WHEN I call apply with changes non-spec fields
//	THEN the resulting object contains the updates
func TestApplyFNonSpec(t *testing.T) {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.VerrazzanoPlatformOperator,
			Namespace: constants.VerrazzanoInstall,
		},
		Secrets: []corev1.ObjectReference{
			{
				Name: "verrazzano-platform-operator-token",
			},
		},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(sa).Build()
	y := k8sutil.NewYAMLApplier(c, "")
	err := y.ApplyF(testdata + "/sa_add_imagepullsecrets.yaml")
	assert.NoError(t, err)

	// Verify the resulting SA
	saUpdated := &corev1.ServiceAccount{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperator, Namespace: constants.VerrazzanoInstall}, saUpdated)
	assert.NoError(t, err)

	assert.NotEmpty(t, saUpdated.ImagePullSecrets)
	assert.Equal(t, 1, len(saUpdated.ImagePullSecrets))
	assert.Equal(t, "verrazzano-container-registry", saUpdated.ImagePullSecrets[0].Name)

	assert.NotEmpty(t, saUpdated.Secrets)
	assert.Equal(t, 1, len(saUpdated.Secrets))
	assert.Equal(t, "verrazzano-platform-operator-token", saUpdated.Secrets[0].Name)
}

// TestApplyFMerge
// GIVEN a object that contains spec field
//
//	WHEN I call apply with additions to the spec field
//	THEN the resulting object contains the merged updates
func TestApplyFMerge(t *testing.T) {
	deadlineSeconds := int32(5)
	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.VerrazzanoPlatformOperator,
			Namespace: constants.VerrazzanoInstall,
		},
		Spec: appv1.DeploymentSpec{
			MinReadySeconds:         5,
			ProgressDeadlineSeconds: &deadlineSeconds,
		},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment).Build()
	y := k8sutil.NewYAMLApplier(c, "")
	err := y.ApplyF(testdata + "/deployment_merge.yaml")
	assert.NoError(t, err)

	// Verify the resulting Deployment
	depUpdated := &appv1.Deployment{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperator, Namespace: constants.VerrazzanoInstall}, depUpdated)
	assert.NoError(t, err)

	assert.Equal(t, int32(5), depUpdated.Spec.MinReadySeconds)
	assert.Equal(t, int32(5), *depUpdated.Spec.Replicas)
	assert.Equal(t, int32(10), *depUpdated.Spec.ProgressDeadlineSeconds)
}

// TestApplyFClusterRole
// GIVEN a ClusterRole object
//
//	WHEN I call apply with additions
//	THEN the resulting object contains the merged updates
func TestApplyFClusterRole(t *testing.T) {
	deadlineSeconds := int32(5)
	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.VerrazzanoPlatformOperator,
			Namespace: constants.VerrazzanoInstall,
		},
		Spec: appv1.DeploymentSpec{
			MinReadySeconds:         5,
			ProgressDeadlineSeconds: &deadlineSeconds,
		},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment).Build()
	y := k8sutil.NewYAMLApplier(c, "")
	err := y.ApplyF(testdata + "/clusterrole_create.yaml")
	assert.NoError(t, err)

	// Verify the ClusterRole that was created
	clusterRole := &rbacv1.ClusterRole{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: "verrazzano-managed-cluster"}, clusterRole)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(clusterRole.Rules))
	rule := clusterRole.Rules[0]
	assert.Equal(t, "", rule.APIGroups[0])
	assert.Equal(t, "secrets", rule.Resources[0])
	assert.Equal(t, 3, len(rule.Verbs))

	// Update the ClusterRole
	err = y.ApplyF(testdata + "/clusterrole_update.yaml")
	assert.NoError(t, err)

	// Verify the ClusterRole that was updated
	clusterRoleUpdated := &rbacv1.ClusterRole{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: "verrazzano-managed-cluster"}, clusterRoleUpdated)
	assert.NoError(t, err)
	rule = clusterRoleUpdated.Rules[0]
	assert.Equal(t, 4, len(rule.Verbs))

	// Verify all the expected verbs are there
	foundCount := 0
	for _, verb := range rule.Verbs {
		switch verb {
		case "get":
			foundCount++
		case "list":
			foundCount++
		case "watch":
			foundCount++
		case "update":
			foundCount++
		}
	}
	assert.Equal(t, 4, foundCount)
}

func TestApplyFT(t *testing.T) {
	var tests = []struct {
		name    string
		file    string
		args    map[string]interface{}
		count   int
		isError bool
	}{
		{
			"should apply a template file",
			testdata + "/templated_service.yaml",
			map[string]interface{}{"namespace": "default"},
			1,
			false,
		},
		{
			"should fail to apply when template is incomplete",
			testdata + "/templated_service.yaml",
			map[string]interface{}{},
			0,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
			y := k8sutil.NewYAMLApplier(c, "")
			err := y.ApplyFT(tt.file, tt.args)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.count, len(y.Objects()))
		})
	}
}

// TestApplyDT tests the ApplyDT function.
func TestApplyDT(t *testing.T) {
	var tests = []struct {
		name    string
		dir     string
		args    map[string]interface{}
		count   int
		isError bool
	}{
		// GIVEN a directory of template YAML files
		// WHEN the ApplyDT function is called with substitution key/value pairs
		// THEN the call succeeds and the resources are applied to the cluster
		{
			"should apply all template files in directory",
			testdata,
			map[string]interface{}{"namespace": "default"},
			7,
			false,
		},
		// GIVEN a directory of template YAML files
		// WHEN the ApplyDT function is called with no substitution key/value pairs
		// THEN the call fails
		{
			"should fail to apply when one or more templates are incomplete",
			testdata,
			map[string]interface{}{},
			4,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
			y := k8sutil.NewYAMLApplier(c, "")
			err := y.ApplyDT(tt.dir, tt.args)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.count, len(y.Objects()))
		})
	}
}

func TestDeleteF(t *testing.T) {
	var tests = []struct {
		name    string
		file    string
		isError bool
	}{
		{
			"should delete valid file",
			testdata + "/two_objects.yaml",
			false,
		},
		{
			"should fail to delete invalid file",
			"blahblah",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
			y := k8sutil.NewYAMLApplier(c, "")
			err := y.DeleteF(tt.file)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteFD(t *testing.T) {
	var tests = []struct {
		name    string
		file    string
		args    map[string]interface{}
		isError bool
	}{
		{
			"should apply a template file",
			testdata + "/templated_service.yaml",
			map[string]interface{}{"namespace": "default"},
			false,
		},
		{
			"should fail to apply when template is incomplete",
			testdata + "/templated_service.yaml",
			map[string]interface{}{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
			y := k8sutil.NewYAMLApplier(c, "")
			err := y.DeleteFT(tt.file, tt.args)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDeleteFTDefaultConfig tests deleteFT with rest client from the default config
// GIVEN a filepath and args
//
//	WHEN TestDeleteFTDefaultConfig is called
//	THEN it fails to get the default restclient
func TestDeleteFTDefaultConfig(t *testing.T) {
	var tests = []struct {
		name    string
		file    string
		args    map[string]interface{}
		isError bool
	}{
		{
			"should fail to delete a template file",
			testdata + "/templated_service.yaml",
			map[string]interface{}{"namespace": "default"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
			y := k8sutil.NewYAMLApplier(c, "")
			err := y.DeleteFTDefaultConfig(tt.file, tt.args)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteAll(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	y := k8sutil.NewYAMLApplier(c, "")
	err := y.ApplyD(objects)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(y.Objects()))
	err = y.DeleteAll()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(y.Objects()))
}
