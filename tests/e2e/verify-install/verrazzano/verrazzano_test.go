// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	rbacv1 "k8s.io/api/rbac/v1"
	"time"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 5 * time.Second
)

var t = framework.NewTestFramework("verrazzano")

var _ = t.AfterEach(func() {})

var _ = t.Describe("In Verrazzano", Label("f:platform-lcm.install"), func() {

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

	t.DescribeTable("CRD for",
		func(name string) {
			Eventually(func() (bool, error) {
				return pkg.DoesCRDExist(name)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		t.Entry("verrazzanos should exist in cluster", "verrazzanos.install.verrazzano.io"),
		t.Entry("verrazzanomanagedclusters should exist in cluster", "verrazzanomanagedclusters.clusters.verrazzano.io"),
	)

	t.DescribeTable("ClusterRole",
		func(name string) {
			Eventually(func() (bool, error) {
				return pkg.DoesClusterRoleExist(name)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		t.Entry("verrazzano-admin should exist", "verrazzano-admin"),
		t.Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
		t.Entry("verrazzano-project-admin should exist", "verrazzano-project-admin"),
		t.Entry("verrazzano-project-monitor should exist", "verrazzano-project-monitor"),
	)

	t.DescribeTable("ClusterRoleBinding",
		func(name string) {
			Eventually(func() (bool, error) {
				return pkg.DoesClusterRoleBindingExist(name)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		t.Entry("verrazzano-admin should exist", "verrazzano-admin"),
		t.Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
	)

	t.Describe("ClusterRole verrazzano-admin", Label("f:security.rbac"), func() {
		var rules []rbacv1.PolicyRule

		t.BeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-admin")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		t.It("has correct number of rules", func() {
			Expect(len(rules)).To(Equal(11),
				"there should be eleven rules")
		})

		t.DescribeTable("should have PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
			},
			t.Entry("vzInstallReadRule", vzInstallReadRule),
			t.Entry("vzInstallWriteRule", vzInstallWriteRule),
			t.Entry("vzSystemReadRule", vzSystemReadRule),
			t.Entry("vzSystemWriteRule", vzSystemWriteRule),
			t.Entry("vzAppReadRule", vzAppReadRule),
			t.Entry("vzAppWriteRule", vzAppWriteRule),
			t.Entry("vzWebLogicReadRule", vzWebLogicReadRule),
			t.Entry("vzWebLogicWriteRule", vzWebLogicWriteRule),
			t.Entry("vzCoherenceReadRule", vzCoherenceReadRule),
			t.Entry("vzCoherenceReadRule", vzCoherenceReadRule),
			t.Entry("vzIstioReadRule", vzIstioReadRule),
		)
	})

	t.Describe("ClusterRole verrazzano-monitor", Label("f:security.rbac"), func() {
		var rules []rbacv1.PolicyRule

		t.BeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-monitor")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		t.It("has correct number of rules", func() {
			Expect(len(rules)).To(Equal(5),
				"there should be five rules")
		})

		t.DescribeTable("should have PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
			},
			t.Entry("vzSystemReadRule", vzSystemReadRule),
			t.Entry("vzAppReadRule", vzAppReadRule),
			t.Entry("vzWebLogicReadRule", vzWebLogicReadRule),
			t.Entry("vzCoherenceReadRule", vzCoherenceReadRule),
			t.Entry("vzIstioReadRule", vzIstioReadRule),
		)
	})

	t.Describe("ClusterRole verrazzano-project-admin", Label("f:security.rbac"), func() {
		var rules []rbacv1.PolicyRule

		t.BeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-project-admin")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		t.It("has correct number of rules", func() {
			Expect(len(rules)).To(Equal(6),
				"there should be six rules")
		})

		t.DescribeTable("should have PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
			},
			t.Entry("vzAppReadRule", vzAppReadRule),
			t.Entry("vzAppWriteRule", vzAppWriteRule),
			t.Entry("vzWebLogicReadRule", vzWebLogicReadRule),
			t.Entry("vzWebLogicWriteRule", vzWebLogicWriteRule),
			t.Entry("vzCoherenceReadRule", vzCoherenceReadRule),
			t.Entry("vzCoherenceWriteRule", vzCoherenceWriteRule),
		)
	})

	t.Describe("ClusterRole verrazzano-project-monitor", Label("f:security.rbac"), func() {
		var rules []rbacv1.PolicyRule

		t.BeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-project-monitor")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		t.It("has correct number of rules", func() {
			Expect(len(rules)).To(Equal(3),
				"there should be three rules")
		})

		t.DescribeTable("should have PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
			},
			t.Entry("vzAppReadRule", vzAppReadRule),
			t.Entry("vzWebLogicReadRule", vzWebLogicReadRule),
			t.Entry("vzCoherenceReadRule", vzCoherenceReadRule),
		)
	})

	t.Describe("ClusterRoleBinding verrazzano-admin", Label("f:security.rbac"), func() {
		t.It("has correct subjects and refs", func() {
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

	t.Describe("ClusterRoleBinding verrazzano-admin-k8s", Label("f:security.rbac"), func() {
		t.It("has correct subjects and refs", func() {
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

	t.Describe("ClusterRoleBinding verrazzano-monitor", Label("f:security.rbac"), func() {
		t.It("has correct subjects and refs", func() {
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

	t.Describe("ClusterRoleBinding verrazzano-monitor-k8s", Label("f:security.rbac"), func() {
		t.It("has correct subjects and refs", func() {
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
