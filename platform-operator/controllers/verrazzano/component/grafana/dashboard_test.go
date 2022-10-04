// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDashboardNames verifies dashboard filenames in the configmap follow the expected format
// GIVEN the list of dashboards
//
//	WHEN I call dashboardName
//	THEN the name is formatted correctly
func TestDashboardNames(t *testing.T) {
	for _, dashboard := range dashboardList {
		name := dashboardName(dashboard)
		assert.NotContainsf(t, name, "manifest/", "should not contain manifests directory")
		assert.NotContainsf(t, name, "/", "forward slashes should be removed")
	}
}
