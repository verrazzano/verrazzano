// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package integ

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/application-operator/test/integ/util"
)

var _ = Describe("Testing Multi-Cluster Namespace CRD", func() {
	It("MultiClusterNamespace CRD can be applied", func() {
		_, stderr := util.Kubectl("apply -f ../../config/crd/bases/clusters.verrazzano.io_multiclusternamespaces.yaml")
		Expect(stderr).To(Equal(""))
	})
	It("MultiClusterNamespace can be created ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multicluster_namespace_sample.yaml")
		Expect(stderr).To(Equal(""))
	})
})
