// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package addonverrazzano_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAddonverrazzano(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Addonverrazzano Suite")
}
