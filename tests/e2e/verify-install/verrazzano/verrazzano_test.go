// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano_test

import (
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 5 * time.Second
)

var metricsLogger, _ = metrics.NewMetricsLogger("verrazzano")

var _ = framework.VzDescribe("Verrazzano", func() {

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

	DescribeTable("CRD for",
		func(name string) {
			Eventually(func() (bool, error) {
				return pkg.DoesCRDExist(name)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		Entry("verrazzanos should exist in cluster", "verrazzanos.install.verrazzano.io"),
		Entry("verrazzanomanagedclusters should exist in cluster", "verrazzanomanagedclusters.clusters.verrazzano.io"),
	)

	DescribeTable("ClusterRole",
		func(name string) {
			Eventually(func() (bool, error) {
				return pkg.DoesClusterRoleExist(name)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		Entry("verrazzano-admin should exist", "verrazzano-admin"),
		Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
		Entry("verrazzano-project-admin should exist", "verrazzano-project-admin"),
		Entry("verrazzano-project-monitor should exist", "verrazzano-project-monitor"),
	)

	DescribeTable("ClusterRoleBinding",
		func(name string) {
			Eventually(func() (bool, error) {
				return pkg.DoesClusterRoleBindingExist(name)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		Entry("verrazzano-admin should exist", "verrazzano-admin"),
		Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
	)

	framework.VzDescribe("ClusterRole verrazzano-admin", func() {
		var rules []rbacv1.PolicyRule

		framework.VzBeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-admin")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		framework.ItM(metricsLogger, "has correct number of rules", func() {
			Expect(len(rules)).To(Equal(11),
				"there should be eleven rules")
		})

		DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
			},
			Entry("vzInstallReadRule should exist", vzInstallReadRule),
			Entry("vzInstallWriteRule should exist", vzInstallWriteRule),
			Entry("vzSystemReadRule should exist", vzSystemReadRule),
			Entry("vzSystemWriteRule should exist", vzSystemWriteRule),
			Entry("vzAppReadRule should exist", vzAppReadRule),
			Entry("vzAppWriteRule should exist", vzAppWriteRule),
			Entry("vzWebLogicReadRule should exist", vzWebLogicReadRule),
			Entry("vzWebLogicWriteRule should exist", vzWebLogicWriteRule),
			Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
			Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
			Entry("vzIstioReadRule should exist", vzIstioReadRule),
		)
	})

	framework.VzDescribe("ClusterRole verrazzano-monitor", func() {
		var rules []rbacv1.PolicyRule

		framework.VzBeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-monitor")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		framework.ItM(metricsLogger, "has correct number of rules", func() {
			Expect(len(rules)).To(Equal(5),
				"there should be five rules")
		})

		DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
			},
			Entry("vzSystemReadRule should exist", vzSystemReadRule),
			Entry("vzAppReadRule should exist", vzAppReadRule),
			Entry("vzWebLogicReadRule should exist", vzWebLogicReadRule),
			Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
			Entry("vzIstioReadRule should exist", vzIstioReadRule),
		)
	})

	framework.VzDescribe("ClusterRole verrazzano-project-admin", func() {
		var rules []rbacv1.PolicyRule

		framework.VzBeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-project-admin")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		framework.ItM(metricsLogger, "has correct number of rules", func() {
			Expect(len(rules)).To(Equal(6),
				"there should be six rules")
		})

		DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
			},
			Entry("vzAppReadRule should exist", vzAppReadRule),
			Entry("vzAppWriteRule should exist", vzAppWriteRule),
			Entry("vzWebLogicReadRule should exist", vzWebLogicReadRule),
			Entry("vzWebLogicWriteRule should exist", vzWebLogicWriteRule),
			Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
			Entry("vzCoherenceWriteRule should exist", vzCoherenceWriteRule),
		)
	})

	framework.VzDescribe("ClusterRole verrazzano-project-monitor", func() {
		var rules []rbacv1.PolicyRule

		framework.VzBeforeEach(func() {
			var cr *rbacv1.ClusterRole
			Eventually(func() (*rbacv1.ClusterRole, error) {
				var err error
				cr, err = pkg.GetClusterRole("verrazzano-project-monitor")
				return cr, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			rules = cr.Rules
		})

		framework.ItM(metricsLogger, "has correct number of rules", func() {
			Expect(len(rules)).To(Equal(3),
				"there should be three rules")
		})

		DescribeTable("PolicyRule",
			func(rule rbacv1.PolicyRule) {
				Expect(pkg.SliceContainsPolicyRule(rules, rule)).To(BeTrue())
			},
			Entry("vzAppReadRule should exist", vzAppReadRule),
			Entry("vzWebLogicReadRule should exist", vzWebLogicReadRule),
			Entry("vzCoherenceReadRule should exist", vzCoherenceReadRule),
		)
	})

	framework.VzDescribe("ClusterRoleBinding verrazzano-admin", func() {
		framework.ItM(metricsLogger, "has correct subjects and refs", func() {
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

	framework.VzDescribe("ClusterRoleBinding verrazzano-admin-k8s", func() {
		framework.ItM(metricsLogger, "has correct subjects and refs", func() {
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

	framework.VzDescribe("ClusterRoleBinding verrazzano-monitor", func() {
		framework.ItM(metricsLogger, "has correct subjects and refs", func() {
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

	framework.VzDescribe("ClusterRoleBinding verrazzano-monitor-k8s", func() {
		framework.ItM(metricsLogger, "has correct subjects and refs", func() {
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

var _ = framework.VzDescribe("Mark's second fake test", func() {
	framework.ItM(metricsLogger, "randomly fails", func() {
		rand.Seed(time.Now().UnixNano())
		/* #nosec */
		r := rand.Intn(4)
		// fail if the random number was a 1
		Expect(r).ToNot(Equal(1))
	})
})
