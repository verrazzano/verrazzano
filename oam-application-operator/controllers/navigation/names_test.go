// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package navigation

import (
	asserts "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

// TestGetDefinitionOfResource tests various use cases of GetDefinitionOfResource
func TestGetDefinitionOfResource(t *testing.T) {
	assert := asserts.New(t)

	var actual types.NamespacedName
	var expect types.NamespacedName

	// GIVEN an valid resource group version kind
	// WHEN the GVK is converted to a CRD name
	// THEN verify the name is correct.
	actual = GetDefinitionOfResource("core.oam.dev/v1alpha2", "ContainerizedWorkload")
	expect = types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}
	assert.Equal(expect, actual)

	// GIVEN an valid "core" resource group version kind
	// WHEN the GVK is converted to a CRD name
	// THEN verify the name is correct.
	actual = GetDefinitionOfResource("v1", "Pod")
	expect = types.NamespacedName{Namespace: "", Name: "pods"}
	assert.Equal(expect, actual)
}

// TestParseGroupAndVersionFromAPIVersion test various use cases of ParseGroupAndVersionFromAPIVersion
func TestParseGroupAndVersionFromAPIVersion(t *testing.T) {
	assert := asserts.New(t)

	var group string
	var version string

	group, version = ParseGroupAndVersionFromAPIVersion("core.oam.dev/v1alpha2")
	assert.Equal("core.oam.dev", group)
	assert.Equal("v1alpha2", version)

	group, version = ParseGroupAndVersionFromAPIVersion("v1")
	assert.Equal("", group)
	assert.Equal("v1", version)
}

func TestGetNamespacedNameFromObjectMeta(t *testing.T) {
	assert := asserts.New(t)
	var objMeta metav1.ObjectMeta
	var nname types.NamespacedName

	objMeta = metav1.ObjectMeta{}
	nname = GetNamespacedNameFromObjectMeta(objMeta)
	assert.Equal("", nname.Namespace)
	assert.Equal("", nname.Name)
}

func TestGetNamespacedNameFromUnstructured(t *testing.T) {
	assert := asserts.New(t)
	var uns unstructured.Unstructured
	var nname types.NamespacedName

	uns = unstructured.Unstructured{}
	nname = GetNamespacedNameFromUnstructured(&uns)
	assert.Equal("", nname.Namespace)
	assert.Equal("", nname.Name)
}

// TestParseNamespacedNameFromQualifiedName tests various use cases of ParseNamespacedNameFromQualifiedName
func TestParseNamespacedNameFromQualifiedName(t *testing.T) {
	assert := asserts.New(t)
	var qname string
	var nname *types.NamespacedName
	var err error

	// GIVEN an empty qualified name
	// WHEN a namespaced name is extracted
	// THEN expect an error and nil namespaced name
	qname = ""
	nname, err = ParseNamespacedNameFromQualifiedName(qname)
	assert.Error(err)
	assert.Nil(nname)

	// GIVEN an valid qualified name
	// WHEN a namespaced name is extracted
	// THEN expect no error and a correct namespaced name returned
	qname = "test-space/test-name"
	nname, err = ParseNamespacedNameFromQualifiedName(qname)
	assert.NoError(err)
	assert.Equal("test-space", nname.Namespace)
	assert.Equal("test-name", nname.Name)

	// GIVEN an valid name qualified with "default" namespace
	// WHEN a namespaced name is extracted
	// THEN expect no error and a correct namespaced name returned
	qname = "/test-name"
	nname, err = ParseNamespacedNameFromQualifiedName(qname)
	assert.NoError(err)
	assert.Equal("", nname.Namespace)
	assert.Equal("test-name", nname.Name)

	// GIVEN an valid unqualified name
	// WHEN a namespaced name is extracted
	// THEN expect an error and nil namespaced name
	qname = "test-name"
	nname, err = ParseNamespacedNameFromQualifiedName(qname)
	assert.Error(err)
	assert.Nil(nname)

	// GIVEN an invalid name
	// WHEN a namespaced name is extracted
	// THEN expect an error and nil namespaced name
	qname = "/"
	nname, err = ParseNamespacedNameFromQualifiedName(qname)
	assert.Error(err)
	assert.Nil(nname)
}
