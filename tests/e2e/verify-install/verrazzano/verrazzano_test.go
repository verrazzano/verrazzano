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

	vzInstallReadRule := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{"install.verrazzano.io"},
		Resources: []string{"*", "*/status"},
	}
	vzInstallWriteRule := rbacv1.PolicyRule{
		Verbs:     []string{"create", "update", "patch", "delete", "deletecollection"},
		APIGroups: []string{"install.verrazzano.io"},
		Resources: []string{"*"},
	}
	vzSystemReadRule := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{"clusters.verrazzano.io", "images.verrazzano.io"},
		Resources: []string{"*", "*/status"},
	}
	vzSystemWriteRule := rbacv1.PolicyRule{
		Verbs:     []string{"create", "update", "patch", "delete", "deletecollection"},
		APIGroups: []string{"clusters.verrazzano.io", "images.verrazzano.io"},
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
		Resources: []string{"coherence", "coherence/status"},
	}
	vzCoherenceWriteRule := rbacv1.PolicyRule{
		Verbs:     []string{"create", "update", "patch", "delete", "deletecollection"},
		APIGroups: []string{"coherence.oracle.com"},
		Resources: []string{"coherence"},
	}
	vzIstioReadRule := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{"config.istio.io", "networking.istio.io", "security.istio.io"},
		Resources: []string{"*", "*/status"},
	}

	ginkgoExt.DescribeTable("CRD for",
		func(name string) {
			exists, err := pkg.DoesCRDExist(name)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(exists).To(gomega.BeTrue())
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
			exists, err := pkg.DoesClusterRoleBindingExist(name)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Error getting cluster role binding")
			gomega.Expect(exists).To(gomega.BeTrue())
		},
		ginkgoExt.Entry("verrazzano-admin should exist", "verrazzano-admin"),
		ginkgoExt.Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
	)

	ginkgo.Describe("ClusterRole verrazzano-admin", func() {
		var rules []rbacv1.PolicyRule

		ginkgo.BeforeEach(func() {
			cr, err := pkg.GetClusterRole("verrazzano-admin")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Error getting cluster role")
			rules = cr.Rules
		})

		ginkgo.It("has correct number of rules", func() {
			gomega.Expect(len(rules)).To(gomega.Equal(11),
				"there should be eleven rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				gomega.Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(gomega.BeTrue())
			},
			ginkgoExt.Entry("vzInstallReadRule should exist", vzInstallReadRule),
			ginkgoExt.Entry("vzInstallWriteRule should exist", vzInstallWriteRule),
			ginkgoExt.Entry("vzSystemReadRule should exist", vzSystemReadRule),
			ginkgoExt.Entry("vzSystemWriteRule should exist", vzSystemWriteRule),
			ginkgoExt.Entry("vzAppReadRule should exist", vzAppReadRule),
			ginkgoExt.Entry("vzAppWriteRule should exist", vzAppWriteRule),
			ginkgoExt.Entry("vzWebLogicReadRule should exist", vzWebLogicReadRule),
			ginkgoExt.Entry("vzWebLogicWriteRule should exist", vzWebLogicWriteRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
			ginkgoExt.Entry("vzIstioReadRule should exist", vzIstioReadRule),
		)
	})

	ginkgo.Describe("ClusterRole verrazzano-monitor", func() {
		var rules []rbacv1.PolicyRule

		ginkgo.BeforeEach(func() {
			cr, err := pkg.GetClusterRole("verrazzano-monitor")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Error getting cluster role")
			rules = cr.Rules
		})

		ginkgo.It("has correct number of rules", func() {
			gomega.Expect(len(rules)).To(gomega.Equal(5),
				"there should be five rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				gomega.Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(gomega.BeTrue())
			},
			ginkgoExt.Entry("vzSystemReadRule should exist", vzSystemReadRule),
			ginkgoExt.Entry("vzAppReadRule should exist", vzAppReadRule),
			ginkgoExt.Entry("vzWebLogicReadRule should exist", vzWebLogicReadRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
			ginkgoExt.Entry("vzIstioReadRule should exist", vzIstioReadRule),
		)
	})

	ginkgo.Describe("ClusterRole verrazzano-project-admin", func() {
		var rules []rbacv1.PolicyRule

		ginkgo.BeforeEach(func() {
			cr, err := pkg.GetClusterRole("verrazzano-project-admin")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Error getting cluster role")
			rules = cr.Rules
		})

		ginkgo.It("has correct number of rules", func() {
			gomega.Expect(len(rules)).To(gomega.Equal(6),
				"there should be six rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				gomega.Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(gomega.BeTrue())
			},
			ginkgoExt.Entry("vzAppReadRule should exist", vzAppReadRule),
			ginkgoExt.Entry("vzAppWriteRule should exist", vzAppWriteRule),
			ginkgoExt.Entry("vzWebLogicReadRule should exist", vzWebLogicReadRule),
			ginkgoExt.Entry("vzWebLogicWriteRule should exist", vzWebLogicWriteRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
			ginkgoExt.Entry("vzCoherenceWriteRule should exist", vzCoherenceWriteRule),
		)
	})

	ginkgo.Describe("ClusterRole verrazzano-project-monitor", func() {
		var rules []rbacv1.PolicyRule

		ginkgo.BeforeEach(func() {
			cr, err := pkg.GetClusterRole("verrazzano-project-monitor")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Error getting cluster role")
			rules = cr.Rules
		})

		ginkgo.It("has correct number of rules", func() {
			gomega.Expect(len(rules)).To(gomega.Equal(3),
				"there should be three rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				gomega.Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(gomega.BeTrue())
			},
			ginkgoExt.Entry("vzAppReadRule should exist", vzAppReadRule),
			ginkgoExt.Entry("vzWebLogicReadRule should exist", vzWebLogicReadRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
		)
	})

	ginkgo.Describe("ClusterRoleBinding verrazzano-admin", func() {
		ginkgo.It("has correct subjects and refs", func() {
			crb, err := pkg.GetClusterRoleBinding("verrazzano-admin")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Error getting cluster role binding")
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

	ginkgo.Describe("ClusterRoleBinding verrazzano-admin-k8s", func() {
		ginkgo.It("has correct subjects and refs", func() {
			crb, err := pkg.GetClusterRoleBinding("verrazzano-admin-k8s")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Error getting cluster role binding")
			gomega.Expect(crb.RoleRef.APIGroup == "rbac.authorization.k8s.io").To(gomega.BeTrue(),
				"the roleRef.apiGroup should be rbac.authorization.k8s.io")
			gomega.Expect(crb.RoleRef.Name == "admin").To(gomega.BeTrue(),
				"the roleRef.name should be admin")
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
			crb, err := pkg.GetClusterRoleBinding("verrazzano-monitor")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Error getting cluster role binding")
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

	ginkgo.Describe("ClusterRoleBinding verrazzano-monitor-k8s", func() {
		ginkgo.It("has correct subjects and refs", func() {
			crb, err := pkg.GetClusterRoleBinding("verrazzano-monitor-k8s")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Error getting cluster role binding")
			gomega.Expect(crb.RoleRef.APIGroup == "rbac.authorization.k8s.io").To(gomega.BeTrue(),
				"the roleRef.apiGroup should be rbac.authorization.k8s.io")
			gomega.Expect(crb.RoleRef.Name == "view").To(gomega.BeTrue(),
				"the roleRef.name should be view")
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
