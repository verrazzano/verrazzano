// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package image

import (
	"github.com/onsi/ginkgo/v2"
	"testing"
)

func TestImageIssues(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Test Suite for VZ Tools Analysis on Image Related Issues")
}
