// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package inplaceupgrade

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestInPlaceUpgrade(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "OKE in-place cluster upgrade")
}
