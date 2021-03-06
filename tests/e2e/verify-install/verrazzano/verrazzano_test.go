// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano_test

import (
	"github.com/onsi/ginkgo"
	ginkgoExt "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	rbacv1 "k8s.io/api/rbac/v1"
)

var _ = ginkgo.Describe("Verrazzano", func() {

	//	vzInstallReadRule := rbacv1.PolicyRule{
	//		Verbs:     []string{"get", "list", "watch"},
	//		APIGroups: []string{"install.verrazzano.io"},
	//		Resources: []string{"*", "*/status"},
	//	}
	//	vzInstallWriteRule := rbacv1.PolicyRule{
	//		Verbs:     []string{"create", "update", "patch", "delete", "deletecollection"},
	//		APIGroups: []string{"install.verrazzano.io"},
	//		Resources: []string{"*"},
	//	}
	vzInstallAllRule := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete", "deletecollection"},
		APIGroups: []string{"install.verrazzano.io"},
		Resources: []string{"*"},
	}
	vzAppReadRule := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{"verrazzano.io", "oam.verrazzano.io", "core.oam.dev"},
		Resources: []string{"*", "*/status"},
	}
	vzAppWriteRule := rbacv1.PolicyRule{
		Verbs:     []string{"create", "update", "patch", "delete", "deletecollection"},
		APIGroups: []string{"verrazzano.io", "oam.verrazzano.io", "core.oam.dev"},
		Resources: []string{"*"},
	}
	vzWebLogicReadRule := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{"weblogic.oracle"},
		Resources: []string{"domains", "domains/status"},
	}
	vzWebLogicWriteRule := rbacv1.PolicyRule{
		Verbs:     []string{"create", "update", "patch", "delete", "deletecollection"},
		APIGroups: []string{"weblogic.oracle"},
		Resources: []string{"domains"},
	}
	vzCoherenceReadRule := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{"coherence.oracle.com"},
		Resources: []string{"coherences", "coherences/status"},
	}
	vzCoherenceWriteRule := rbacv1.PolicyRule{
		Verbs:     []string{"create", "update", "patch", "delete", "deletecollection"},
		APIGroups: []string{"coherence.oracle.com"},
		Resources: []string{"coherences"},
	}

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
		ginkgoExt.Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
		ginkgoExt.Entry("verrazzano-project-admin should exist", "verrazzano-project-admin"),
		ginkgoExt.Entry("verrazzano-project-monitor should exist", "verrazzano-project-monitor"),
	)

	ginkgoExt.DescribeTable("ClusterRoleBinding",
		func(name string) {
			gomega.Expect(pkg.DoesClusterRoleBindingExist(name)).To(gomega.BeTrue())
		},
		ginkgoExt.Entry("verrazzano-admin should exist", "verrazzano-admin"),
		ginkgoExt.Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
	)

	ginkgo.Describe("ClusterRole verrazzano-admin", func() {
		cr := pkg.GetClusterRole("verrazzano-admin")
		rules := cr.Rules

		ginkgo.It("has correct number of rules", func() {
			gomega.Expect(len(rules)).To(gomega.Equal(7),
				"there should be seven rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(ruleSlice []rbacv1.PolicyRule, rule rbacv1.PolicyRule) {
				gomega.Expect(pkg.SliceContainsPolicyRule(ruleSlice, rule)).To(gomega.BeTrue())
			},
			//ginkgoExt.Entry("vzInstallReadRule should exist", rules, vzInstallReadRule),
			//ginkgoExt.Entry("vzInstallWriteRule should exist", rules, vzInstallWriteRule),
			ginkgoExt.Entry("vzInstallAllRule should exist", rules, vzInstallAllRule),
			ginkgoExt.Entry("vzAppReadRule should exist", rules, vzAppReadRule),
			ginkgoExt.Entry("vzAppWriteRule should exist", rules, vzAppWriteRule),
			ginkgoExt.Entry("vzWebLogicReadRule should exist", rules, vzWebLogicReadRule),
			ginkgoExt.Entry("vzWebLogicWriteRule should exist", rules, vzWebLogicWriteRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", rules, vzCoherenceReadRule),
			ginkgoExt.Entry("vzCoherenceWriteRule should exist", rules, vzCoherenceWriteRule),
		)
	})

	ginkgo.Describe("ClusterRole verrazzano-monitor", func() {
		cr := pkg.GetClusterRole("verrazzano-monitor")
		rules := cr.Rules

		ginkgo.It("has correct rules", func() {
			gomega.Expect(len(rules)).To(gomega.Equal(3),
				"there should be three rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(ruleSlice []rbacv1.PolicyRule, rule rbacv1.PolicyRule) {
				gomega.Expect(pkg.SliceContainsPolicyRule(ruleSlice, rule)).To(gomega.BeTrue())
			},
			ginkgoExt.Entry("vzAppReadRule should exist", rules, vzAppReadRule),
			ginkgoExt.Entry("vzWebLogicReadRule should exist", rules, vzWebLogicReadRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", rules, vzCoherenceReadRule),
		)
	})

	ginkgo.Describe("ClusterRole verrazzano-project-admin", func() {
		cr := pkg.GetClusterRole("verrazzano-project-admin")
		rules := cr.Rules

		ginkgo.It("has correct number of rules", func() {
			gomega.Expect(len(rules)).To(gomega.Equal(6),
				"there should be six rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(ruleSlice []rbacv1.PolicyRule, rule rbacv1.PolicyRule) {
				gomega.Expect(pkg.SliceContainsPolicyRule(ruleSlice, rule)).To(gomega.BeTrue())
			},
			ginkgoExt.Entry("vzAppReadRule should exist", rules, vzAppReadRule),
			ginkgoExt.Entry("vzAppWriteRule should exist", rules, vzAppWriteRule),
			ginkgoExt.Entry("vzWebLogicReadRule should exist", rules, vzWebLogicReadRule),
			ginkgoExt.Entry("vzWebLogicWriteRule should exist", rules, vzWebLogicWriteRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", rules, vzCoherenceReadRule),
			ginkgoExt.Entry("vzCoherenceWriteRule should exist", rules, vzCoherenceWriteRule),
		)
	})

	ginkgo.Describe("ClusterRole verrazzano-project-monitor", func() {
		cr := pkg.GetClusterRole("verrazzano-project-monitor")
		rules := cr.Rules

		ginkgo.It("has correct rules", func() {
			gomega.Expect(len(rules)).To(gomega.Equal(3),
				"there should be three rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(ruleSlice []rbacv1.PolicyRule, rule rbacv1.PolicyRule) {
				gomega.Expect(pkg.SliceContainsPolicyRule(ruleSlice, rule)).To(gomega.BeTrue())
			},
			ginkgoExt.Entry("vzAppReadRule should exist", rules, vzAppReadRule),
			ginkgoExt.Entry("vzWebLogicReadRule should exist", rules, vzWebLogicReadRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", rules, vzCoherenceReadRule),
		)
	})

	ginkgo.Describe("ClusterRoleBinding verrazzano-admin", func() {
		ginkgo.It("has correct subjects and refs", func() {
			crb := pkg.GetClusterRoleBinding("verrazzano-admin")
			gomega.Expect(crb.RoleRef.APIGroup == "rbac.authorization.k8s.io").To(gomega.BeTrue(),
				"the roleRef.apiGroup should be rbac.authorization.k8s.io")
			gomega.Expect(crb.RoleRef.Name == "verrazzano-admin").To(gomega.BeTrue(),
				"the roleRef.name should be verrazzano-admin")
			gomega.Expect(crb.RoleRef.Kind == "ClusterRole").To(gomega.BeTrue(),
				"the roleRef.kind shoudl be ClusterRole")

			gomega.Expect(len(crb.Subjects) == 1).To(gomega.BeTrue(),
				"there should be one subject")
			s := crb.Subjects[0]
			gomega.Expect(s.APIGroup == "rbac.authorization.k8s.io").To(gomega.BeTrue(),
				"the subject's apiGroup should be rbac.authorization.k8s.io")
			gomega.Expect(s.Kind == "Group").To(gomega.BeTrue(),
				"the subject's kind should be Group")
			gomega.Expect(s.Name == "verrazzano-admins").To(gomega.BeTrue(),
				"the subject's name should be verrazzano-admins")
		})
	})

	ginkgo.Describe("ClusterRoleBinding verrazzano-monitor", func() {
		ginkgo.It("has correct subjects and refs", func() {
			crb := pkg.GetClusterRoleBinding("verrazzano-monitor")
			gomega.Expect(crb.RoleRef.APIGroup == "rbac.authorization.k8s.io").To(gomega.BeTrue(),
				"the roleRef.apiGroup should be rbac.authorization.k8s.io")
			gomega.Expect(crb.RoleRef.Name == "verrazzano-monitor").To(gomega.BeTrue(),
				"the roleRef.name should be verrazzano-monitor")
			gomega.Expect(crb.RoleRef.Kind == "ClusterRole").To(gomega.BeTrue(),
				"the roleRef.kind shoudl be ClusterRole")

			gomega.Expect(len(crb.Subjects) == 1).To(gomega.BeTrue(),
				"there should be one subject")
			s := crb.Subjects[0]
			gomega.Expect(s.APIGroup == "rbac.authorization.k8s.io").To(gomega.BeTrue(),
				"the subject's apiGroup should be rbac.authorization.k8s.io")
			gomega.Expect(s.Kind == "Group").To(gomega.BeTrue(),
				"the subject's kind should be Group")
			gomega.Expect(s.Name == "verrazzano-monitors").To(gomega.BeTrue(),
				"the subject's name should be verrazzano-monitors")
		})
	})

})
