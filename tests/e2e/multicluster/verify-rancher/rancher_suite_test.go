// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher_test

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestKubernetes(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Default Rancher multi-cluster Suite")
}
