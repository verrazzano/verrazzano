// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano_test

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestVerrazzano(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Verrazzano Suite")
}
