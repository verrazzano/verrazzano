// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"testing"

	"github.com/verrazzano/verrazzano/application-operator/controllers/appconfig"

	ctrl "sigs.k8s.io/controller-runtime"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
)

var log = ctrl.Log.WithName("test")

// TestConvertAPIVersionToGroupAndVersion tests multiple use cases for parsing APIVersion
func TestConvertAPIVersionToGroupAndVersion(t *testing.T) {
	assert := asserts.New(t)
	var g, v string

	// GIVEN a normal group/version string
	// WHEN it is parsed into group and version parts
	// THEN ensure the parts are correct.
	g, v = ConvertAPIVersionToGroupAndVersion("group/version")
	assert.Equal("group", g)
	assert.Equal("version", v)

	// GIVEN a normal group/version string with no group.
	// WHEN it is parsed into group and version parts
	// THEN ensure the group is the empty string and the version is correct.
	// This is the case for older standard kubernetes core resources.
	g, v = ConvertAPIVersionToGroupAndVersion("/version")
	assert.Equal("", g)
	assert.Equal("version", v)

	// GIVEN a normal group/version string with no group.
	// WHEN it is parsed into group and version parts
	// THEN ensure the group is the empty string and the version is correct.
	// This is the case for older standard kubernetes core resources.
	g, v = ConvertAPIVersionToGroupAndVersion("version")
	assert.Equal("", g)
	assert.Equal("version", v)
}

// TestIsWorkloadMarkedForUpgrade tests IsWorkloadMarkedForUpgrade to ensure that it returns the correct response
// based on a map of annotations and a current version.
func TestIsWorkloadMarkedForUpgrade(t *testing.T) {
	assert := asserts.New(t)
	annotations := map[string]string{"foo": "bar", constants.AnnotationUpgradeVersion: "12345"}

	// GIVEN a current upgrade version that matches the corresponding annotation
	// WHEN IsWorkloadMarkedForUpgrade is called
	// THEN false is returned
	assert.False(IsWorkloadMarkedForUpgrade(annotations, "12345"))

	// GIVEN a current upgrade version that doesn't match the corresponding annotation
	// WHEN IsWorkloadMarkedForUpgrade is called
	// THEN true is returned
	assert.True(IsWorkloadMarkedForUpgrade(annotations, "99999"))
}

// TestIsWorkloadMarkedForRestart tests IsWorkloadMarkedForRestart to ensure that it returns the correct response
// based on a map of annotations and a current version.
func TestIsWorkloadMarkedForRestart(t *testing.T) {
	assert := asserts.New(t)
	annotations := map[string]string{"foo": "bar", appconfig.RestartVersionAnnotation: "abc"}

	// GIVEN a current restart version that matches the corresponding annotation
	// WHEN IsWorkloadMarkedForRestart is called
	// THEN false is returned
	assert.False(IsWorkloadMarkedForRestart(annotations, "abc", log))

	// GIVEN a current restart version that doesn't match the corresponding annotation
	// WHEN IsWorkloadMarkedForRestart is called
	// THEN true is returned
	assert.True(IsWorkloadMarkedForRestart(annotations, "xyz", log))
}
