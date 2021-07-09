// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	ginkgoExt "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 5 * time.Second
)

var _ = Describe("Verrazzano", func() {

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
			Eventually(func() (bool, error) {
				return pkg.DoesCRDExist(name)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		ginkgoExt.Entry("verrazzanos should exist in cluster", "verrazzanos.install.verrazzano.io"),
		ginkgoExt.Entry("verrazzanomanagedclusters should exist in cluster", "verrazzanomanagedclusters.clusters.verrazzano.io"),
	)

	ginkgoExt.DescribeTable("ClusterRole",
		func(name string) {
			Eventually(func() (bool, error) {
				return pkg.DoesClusterRoleExist(name)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		ginkgoExt.Entry("verrazzano-admin should exist", "verrazzano-admin"),
		ginkgoExt.Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
		ginkgoExt.Entry("verrazzano-project-admin should exist", "verrazzano-project-admin"),
		ginkgoExt.Entry("verrazzano-project-monitor should exist", "verrazzano-project-monitor"),
	)

	ginkgoExt.DescribeTable("ClusterRoleBinding",
		func(name string) {
			Eventually(func() (bool, error) {
				return pkg.DoesClusterRoleBindingExist(name)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		ginkgoExt.Entry("verrazzano-admin should exist", "verrazzano-admin"),
		ginkgoExt.Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
	)

	Describe("ClusterRole verrazzano-admin", func() {
		var rules []rbacv1.PolicyRule

		BeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-admin")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		It("has correct number of rules", func() {
			Expect(len(rules)).To(Equal(11),
				"there should be eleven rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
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

	Describe("ClusterRole verrazzano-monitor", func() {
		var rules []rbacv1.PolicyRule

		BeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-monitor")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		It("has correct number of rules", func() {
			Expect(len(rules)).To(Equal(5),
				"there should be five rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
			},
			ginkgoExt.Entry("vzSystemReadRule should exist", vzSystemReadRule),
			ginkgoExt.Entry("vzAppReadRule should exist", vzAppReadRule),
			ginkgoExt.Entry("vzWebLogicReadRule should exist", vzWebLogicReadRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
			ginkgoExt.Entry("vzIstioReadRule should exist", vzIstioReadRule),
		)
	})

	Describe("ClusterRole verrazzano-project-admin", func() {
		var rules []rbacv1.PolicyRule

		BeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-project-admin")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		It("has correct number of rules", func() {
			Expect(len(rules)).To(Equal(6),
				"there should be six rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
			},
			ginkgoExt.Entry("vzAppReadRule should exist", vzAppReadRule),
			ginkgoExt.Entry("vzAppWriteRule should exist", vzAppWriteRule),
			ginkgoExt.Entry("vzWebLogicReadRule should exist", vzWebLogicReadRule),
			ginkgoExt.Entry("vzWebLogicWriteRule should exist", vzWebLogicWriteRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
			ginkgoExt.Entry("vzCoherenceWriteRule should exist", vzCoherenceWriteRule),
		)
	})

	Describe("ClusterRole verrazzano-project-monitor", func() {
		var rules []rbacv1.PolicyRule

		BeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-project-monitor")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		It("has correct number of rules", func() {
			Expect(len(rules)).To(Equal(3),
				"there should be three rules")
		})

		ginkgoExt.DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
			},
			ginkgoExt.Entry("vzAppReadRule should exist", vzAppReadRule),
			ginkgoExt.Entry("vzWebLogicReadRule should exist", vzWebLogicReadRule),
			ginkgoExt.Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
		)
	})

	Describe("ClusterRoleBinding verrazzano-admin", func() {
		It("has correct subjects and refs", func() {
			var crb *rbacv1.ClusterRoleBinding
			Eventually(func() (*rbacv1.ClusterRoleBinding, error) {
				var err error
				crb, err = pkg.GetClusterRoleBinding("verrazzano-admin")
				return crb, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			Expect(crb.RoleRef.APIGroup == "rbac.authorization.k8s.io").To(BeTrue(),
				"the roleRef.apiGroup should be rbac.authorization.k8s.io")
			Expect(crb.RoleRef.Name == "verrazzano-admin").To(BeTrue(),
				"the roleRef.name should be verrazzano-admin")
			Expect(crb.RoleRef.Kind == "ClusterRole").To(BeTrue(),
				"the roleRef.kind shoudl be ClusterRole")

			Expect(len(crb.Subjects) == 1).To(BeTrue(),
				"there should be one subject")
			s := crb.Subjects[0]
			Expect(s.APIGroup == "rbac.authorization.k8s.io").To(BeTrue(),
				"the subject's apiGroup should be rbac.authorization.k8s.io")
			Expect(s.Kind == "Group").To(BeTrue(),
				"the subject's kind should be Group")
			Expect(s.Name == "verrazzano-admins").To(BeTrue(),
				"the subject's name should be verrazzano-admins")
		})
	})

	Describe("ClusterRoleBinding verrazzano-admin-k8s", func() {
		It("has correct subjects and refs", func() {
			var crb *rbacv1.ClusterRoleBinding
			Eventually(func() (*rbacv1.ClusterRoleBinding, error) {
				var err error
				crb, err = pkg.GetClusterRoleBinding("verrazzano-admin-k8s")
				return crb, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			Expect(crb.RoleRef.APIGroup == "rbac.authorization.k8s.io").To(BeTrue(),
				"the roleRef.apiGroup should be rbac.authorization.k8s.io")
			Expect(crb.RoleRef.Name == "admin").To(BeTrue(),
				"the roleRef.name should be admin")
			Expect(crb.RoleRef.Kind == "ClusterRole").To(BeTrue(),
				"the roleRef.kind shoudl be ClusterRole")

			Expect(len(crb.Subjects) == 1).To(BeTrue(),
				"there should be one subject")
			s := crb.Subjects[0]
			Expect(s.APIGroup == "rbac.authorization.k8s.io").To(BeTrue(),
				"the subject's apiGroup should be rbac.authorization.k8s.io")
			Expect(s.Kind == "Group").To(BeTrue(),
				"the subject's kind should be Group")
			Expect(s.Name == "verrazzano-admins").To(BeTrue(),
				"the subject's name should be verrazzano-admins")
		})
	})

	Describe("ClusterRoleBinding verrazzano-monitor", func() {
		It("has correct subjects and refs", func() {
			var crb *rbacv1.ClusterRoleBinding
			Eventually(func() (*rbacv1.ClusterRoleBinding, error) {
				var err error
				crb, err = pkg.GetClusterRoleBinding("verrazzano-monitor")
				return crb, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			Expect(crb.RoleRef.APIGroup == "rbac.authorization.k8s.io").To(BeTrue(),
				"the roleRef.apiGroup should be rbac.authorization.k8s.io")
			Expect(crb.RoleRef.Name == "verrazzano-monitor").To(BeTrue(),
				"the roleRef.name should be verrazzano-monitor")
			Expect(crb.RoleRef.Kind == "ClusterRole").To(BeTrue(),
				"the roleRef.kind shoudl be ClusterRole")

			Expect(len(crb.Subjects) == 1).To(BeTrue(),
				"there should be one subject")
			s := crb.Subjects[0]
			Expect(s.APIGroup == "rbac.authorization.k8s.io").To(BeTrue(),
				"the subject's apiGroup should be rbac.authorization.k8s.io")
			Expect(s.Kind == "Group").To(BeTrue(),
				"the subject's kind should be Group")
			Expect(s.Name == "verrazzano-monitors").To(BeTrue(),
				"the subject's name should be verrazzano-monitors")
		})
	})

	Describe("ClusterRoleBinding verrazzano-monitor-k8s", func() {
		It("has correct subjects and refs", func() {
			var crb *rbacv1.ClusterRoleBinding
			Eventually(func() (*rbacv1.ClusterRoleBinding, error) {
				var err error
				crb, err = pkg.GetClusterRoleBinding("verrazzano-monitor-k8s")
				return crb, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			Expect(crb.RoleRef.APIGroup == "rbac.authorization.k8s.io").To(BeTrue(),
				"the roleRef.apiGroup should be rbac.authorization.k8s.io")
			Expect(crb.RoleRef.Name == "view").To(BeTrue(),
				"the roleRef.name should be view")
			Expect(crb.RoleRef.Kind == "ClusterRole").To(BeTrue(),
				"the roleRef.kind shoudl be ClusterRole")

			Expect(len(crb.Subjects) == 1).To(BeTrue(),
				"there should be one subject")
			s := crb.Subjects[0]
			Expect(s.APIGroup == "rbac.authorization.k8s.io").To(BeTrue(),
				"the subject's apiGroup should be rbac.authorization.k8s.io")
			Expect(s.Kind == "Group").To(BeTrue(),
				"the subject's kind should be Group")
			Expect(s.Name == "verrazzano-monitors").To(BeTrue(),
				"the subject's name should be verrazzano-monitors")
		})
	})

})
