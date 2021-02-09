// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano_test

import (
	"github.com/onsi/ginkgo"
	ginkgoExt "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/rbac/v1"
)

var _ = ginkgo.Describe("Verrazzano", func() {

	ginkgoExt.DescribeTable("CRD for",
		func(name string) {
			gomega.Expect(pkg.DoesCRDExist(name)).To(gomega.BeTrue())
		},
		ginkgoExt.Entry("verrazzanos should exist in cluster", "verrazzanos.install.verrazzano.io"),
		ginkgoExt.Entry("verrazzanomanagedclusters should exist in cluster", "verrazzanomanagedclusters.clusters.verrazzano.io"),
	)

	ginkgoExt.DescribeTable("ClusterRole",
		func(name string) {
			gomega.Expect(pkg.DoesClusterRoleExist(name)).To(gomega.BeTrue())
		},
		ginkgoExt.Entry("verrazzano-admin should exist", "verrazzano-admin"),
		ginkgoExt.Entry("verrazzano-app-admin should exist", "verrazzano-app-admin"),
		ginkgoExt.Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
	)

	ginkgoExt.DescribeTable("ClusterRoleBinding",
		func(name string) {
			gomega.Expect(pkg.DoesClusterRoleBindingExist(name)).To(gomega.BeTrue())
		},
		ginkgoExt.Entry("verrazzano-admin should exist", "verrazzano-admin"),
		ginkgoExt.Entry("verrazzano-app-admin should exist", "verrazzano-app-admin"),
		ginkgoExt.Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
	)

	ginkgoExt.DescribeTable("ClusterRoles have the correct Rules",
		func(clusterrole string, apigroup string, resource string, verb string, expected bool) {
			theClusterrole := pkg.GetClusterRole(clusterrole)
			gomega.Expect(clusterroleContains(theClusterrole, apigroup, resource, verb)).To(gomega.Equal(expected))
		},
		ginkgoExt.Entry("verrazzano-admin", "verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-admin", "verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-admin", "verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-admin", "verrazzano.io", "'*'", "put", true),
		ginkgoExt.Entry("verrazzano-admin", "verrazzano.io", "'*'", "post", true),
		ginkgoExt.Entry("verrazzano-admin", "oam.verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-admin", "oam.verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-admin", "oam.verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-admin", "oam.verrazzano.io", "'*'", "put", true),
		ginkgoExt.Entry("verrazzano-admin", "oam.verrazzano.io", "'*'", "post", true),
		ginkgoExt.Entry("verrazzano-admin", "install.verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-admin", "install.verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-admin", "install.verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-admin", "install.verrazzano.io", "'*'", "put", true),
		ginkgoExt.Entry("verrazzano-admin", "install.verrazzano.io", "'*'", "post", true),

		ginkgoExt.Entry("verrazzano-app-admin", "verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-app-admin", "verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-app-admin", "verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-app-admin", "verrazzano.io", "'*'", "put", true),
		ginkgoExt.Entry("verrazzano-app-admin", "verrazzano.io", "'*'", "post", true),
		ginkgoExt.Entry("verrazzano-app-admin", "oam.verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-app-admin", "oam.verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-app-admin", "oam.verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-app-admin", "oam.verrazzano.io", "'*'", "put", true),
		ginkgoExt.Entry("verrazzano-app-admin", "oam.verrazzano.io", "'*'", "post", true),
		ginkgoExt.Entry("verrazzano-app-admin", "install.verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-app-admin", "install.verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-app-admin", "install.verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-app-admin", "install.verrazzano.io", "'*'", "put", true),
		ginkgoExt.Entry("verrazzano-app-admin", "install.verrazzano.io", "'*'", "post", true),

		ginkgoExt.Entry("verrazzano-monitor", "verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-monitor", "verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-monitor", "verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-monitor", "verrazzano.io", "'*'", "put", false),
		ginkgoExt.Entry("verrazzano-monitor", "verrazzano.io", "'*'", "post", false),
		ginkgoExt.Entry("verrazzano-monitor", "oam.verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-monitor", "oam.verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-monitor", "oam.verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-monitor", "oam.verrazzano.io", "'*'", "put", false),
		ginkgoExt.Entry("verrazzano-monitor", "oam.verrazzano.io", "'*'", "post", false),
		ginkgoExt.Entry("verrazzano-monitor", "install.verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-monitor", "install.verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-monitor", "install.verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-monitor", "install.verrazzano.io", "'*'", "put", false),
		ginkgoExt.Entry("verrazzano-monitor", "install.verrazzano.io", "'*'", "post", false),

		ginkgoExt.Entry("verrazzano-app-monitor", "verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-app-monitor", "verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-app-monitor", "verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-app-monitor", "verrazzano.io", "'*'", "put", false),
		ginkgoExt.Entry("verrazzano-app-monitor", "verrazzano.io", "'*'", "post", false),
		ginkgoExt.Entry("verrazzano-app-monitor", "oam.verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-app-monitor", "oam.verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-app-monitor", "oam.verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-app-monitor", "oam.verrazzano.io", "'*'", "put", false),
		ginkgoExt.Entry("verrazzano-app-monitor", "oam.verrazzano.io", "'*'", "post", false),
		ginkgoExt.Entry("verrazzano-app-monitor", "install.verrazzano.io", "'*'", "get", true),
		ginkgoExt.Entry("verrazzano-app-monitor", "install.verrazzano.io", "'*'", "list", true),
		ginkgoExt.Entry("verrazzano-app-monitor", "install.verrazzano.io", "'*'", "watch", true),
		ginkgoExt.Entry("verrazzano-app-monitor", "install.verrazzano.io", "'*'", "put", false),
		ginkgoExt.Entry("verrazzano-app-monitor", "install.verrazzano.io", "'*'", "post", false),
	)

	// now add clusterrolebindings...
})

func clusterroleContains(clusterrole *v1.ClusterRole, apigroup string, resource string, verb string) bool {
	for _, role := range clusterrole.Rules {
		if pkg.SliceContainsString(role.APIGroups, apigroup) {
			if pkg.SliceContainsString(role.Resources, resource) {
				if pkg.SliceContainsString(role.Verbs, verb) {
					return true
				}
			}
		}
	}
	return false
}