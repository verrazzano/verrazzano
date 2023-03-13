// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loop

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestRancherLoopInstallUninstall(test *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(test, "Test Rancher Install then Uninstall in a loop")
}
