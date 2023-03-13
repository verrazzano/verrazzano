// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loop

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os/exec"

	"os"
)

// This test performs a Rancher loop test
var _ = Describe("Rancher install-uninstall loop", func() {

	It(fmt.Sprintf("Installing VPO"), func() {
		vpoURL := os.Getenv("VPO_YAML_URL")
		if len(vpoURL) == 0 {
			Fail("Missing VPO_YAML_URL env var for VPO operator.yaml")
		}
		kubectlArgs := []string{
			"apply",
			"-f",
			vpoURL,
		}
		_, err := exec.Command("kubectl", kubectlArgs...).CombinedOutput() //nolint:gosec //#nosec G204
		if err != nil {
			Fail(fmt.Sprintf("Error occurred running kubectl apply to install VPO: %v", err))
		}
		Expect(err).ShouldNot(HaveOccurred())
	})
	for i := 1; i < 5; i++ {
		It(fmt.Sprintf("Starting install: loop %v/n", i+1), func() {
		})
		It(fmt.Sprintf("Waiting for install to complete"), func() {
		})
		It(fmt.Sprintf("Starting uninstall"), func() {
		})
		It(fmt.Sprintf("Waiting for uninstall to complete"), func() {
		})
	}

})
