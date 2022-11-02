// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/embedded"
	"os"
	"sigs.k8s.io/yaml"
	"testing"
)

func Test(t *testing.T) {
	// Extract the manifests and write them to a temp directory
	man, err := embedded.ExtractManifests()
	if err != nil {
		fmt.Printf("Unable to extract manifests from psrctl binary %v", err)
		os.Exit(1)
	}
	defer os.RemoveAll(man.RootTmpDir)

	dir := embedded.Manifests.ScenarioAbsDir

	data, err := os.ReadFile(dir + "/opensearch/s1/scenario.yaml")
	assert.NoError(t, err)
	var sc Scenario
	yaml.Unmarshal(data, &sc)
	assert.Equal(t, sc.Name, "OpenSearch-S1")
}
