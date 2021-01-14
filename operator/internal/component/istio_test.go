// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

// TestPreUpgrade tests the Istio preUpgrade function
// GIVEN a chartDir
//  WHEN I call istioPreUpgrade
//  THEN the correct job hame is passed as a parameter
func TestPreUpgrade(t *testing.T) {
	assert := assert.New(t)
	deleteJobFunc = fakeDeletJob

	chartDir, _ := filepath.Abs("./testdata")
	err := PreUpgrade(nil, "", "ns", chartDir)
	assert.NoError(err, "PreUpgrade returned an error")
}

func fakeDeletJob(_ clipkg.Client, jobName string, namespace string) error {
	if jobName != "istio-security-post-install-1.4.6" {
		return fmt.Errorf("Incorrect job name %s", jobName)
	}
	if namespace != "ns" {
		return fmt.Errorf("Incorrect namespace %s", namespace)
	}
	return nil
}
