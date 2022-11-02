// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/embedded"
	"os"
	"testing"
)

func TestPossibleScenarios(t *testing.T) {
	// Extract the manifests and write them to a temp directory
	man, err := embedded.ExtractManifests()
	if err != nil {
		fmt.Printf("Unable to extract manifests from psrctl binary %v", err)
		os.Exit(1)
	}
	defer os.RemoveAll(man.RootTmpDir)

	sList, err := ListAvailableScenarios("./testdata")
	assert.NoError(t, err)
	assert.Equal(t, "OpenSearch-S1", sList[0].Name)
	assert.Equal(t, "ops-s1", sList[0].ID)
	assert.Equal(t, "ops-s1 description", sList[0].Description)

	assert.Equal(t, "OpenSearch-S2", sList[1].Name)
	assert.Equal(t, "ops-s2", sList[1].ID)
	assert.Equal(t, "ops-s2 description", sList[1].Description)
}
