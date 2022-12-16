// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"testing"
)

// no error is expected from GetDeploymentList of a test dump cluster
// not empty deployments are expected in FindProblematicDeployments
func TestGetDeploymentList(t *testing.T) {
	deploymentList, err := GetDeploymentList(log.GetDebugEnabledLogger(), files.FindFileInNamespace("../../../test/cluster/problem-pods-install/cluster-snapshot", "verrazzano-install", "deployments.json"))
	assert.NoError(t, err)
	assert.NotEmpty(t, FindProblematicDeployments(deploymentList))
}
