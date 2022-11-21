// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify

import (
	"github.com/onsi/ginkgo/v2"

	"testing"
)

func TestVerifyAppRestart(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Upgrade Verify App Restart Suite")
}
