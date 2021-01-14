// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

// TestIstioPreUpgrade tests the Istio preUpgrade function
// GIVEN a chartDir
//  WHEN I call istioPreUpgrade
//  THEN the correct job hame is passed as a parameter
func TestIstioPreUpgrade(t *testing.T) {
	assert := assert.New(t)
	deleteJobFunc = istioFakeDeletJob

	chartDir, _ := filepath.Abs("../../../../thirdparty/charts/istio")
	err := IstioPreUpgrade(nil, "", "ns", chartDir)
	assert.NoError(err, "IstioPreUpgrade returned an error")
}

func istioFakeDeletJob(_ clipkg.Client, jobName string, namespace string) error {
	if jobName != "istio-security-post-install-1.4.6" {
		return errors.New(fmt.Sprintf("Incorrect job name %s", jobName))
	}
	if namespace != "ns" {
		return errors.New(fmt.Sprintf("Incorrect namespace %s", namespace))
	}
	return nil
}
