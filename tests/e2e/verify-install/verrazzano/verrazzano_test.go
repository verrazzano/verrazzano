// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 5 * time.Second
)

// Initialized in BeforeSuite
var isMinVersion110 bool
var isMinVersion120 bool
var isMinVersion140 bool
var isMinVersion150 bool
var keycloakEnabled bool
var mySQLOperatorEnabled bool

var t = framework.NewTestFramework("verrazzano")

var _ = t.AfterEach(func() {})

var _ = t.BeforeSuite(func() {
	var err error
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	isMinVersion110, err = pkg.IsVerrazzanoMinVersion("1.1.0", kubeconfigPath)
	if err != nil {
		Fail(err.Error())
	}
	isMinVersion120, err = pkg.IsVerrazzanoMinVersion("1.2.0", kubeconfigPath)
	if err != nil {
		Fail(err.Error())
	}
	isMinVersion140, err = pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
	if err != nil {
		Fail(err.Error())
	}
	isMinVersion150, err = pkg.IsVerrazzanoMinVersion("1.5.0", kubeconfigPath)
	if err != nil {
		Fail(err.Error())
	}
	keycloakEnabled = pkg.IsKeycloakEnabled(kubeconfigPath)
	mySQLOperatorEnabled = pkg.IsMySQLOperatorEnabled(kubeconfigPath)
})

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
				"the roleRef.kind should be ClusterRole")

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
				"the roleRef.kind should be ClusterRole")

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

	t.Describe("verrazzano-console", Label("f:platform-lcm.install"), func() {
		t.It("has expected deployment", func() {
			if isMinVersion110 && !pkg.IsManagedClusterProfile() {
				Eventually(func() (bool, error) {
					return pkg.DoesDeploymentExist(constants.VerrazzanoSystemNamespace, "verrazzano-console")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})

		t.It("has affinity configured as expected", func() {
			if isMinVersion140 && !pkg.IsManagedClusterProfile() {
				assertPodAntiAffinity(map[string]string{"app": "verrazzano-console"}, constants.VerrazzanoSystemNamespace)
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.4.0")
			}
		})
	})

	t.Describe("verrazzano-authproxy", Label("f:platform-lcm.install"), func() {
		t.It("has expected deployment", func() {
			if isMinVersion110 {
				Eventually(func() (bool, error) {
					return pkg.DoesDeploymentExist(constants.VerrazzanoSystemNamespace, "verrazzano-authproxy")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})

		t.It("has correct number of pods running", func() {
			if isMinVersion110 {
				validateCorrectNumberOfPodsRunning("verrazzano-authproxy", constants.VerrazzanoSystemNamespace)
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})

		t.It("has affinity configured as expected", func() {
			if isMinVersion120 {
				assertPodAntiAffinity(map[string]string{"app": "verrazzano-authproxy"}, constants.VerrazzanoSystemNamespace)
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.2.0")
			}
		})
	})

	t.Describe("cert-manager", Label("f:platform-lcm.install"), func() {
		t.It("has expected deployment", func() {
			if isMinVersion110 {
				Eventually(func() (bool, error) {
					return pkg.DoesDeploymentExist(constants.CertManagerNamespace, "cert-manager")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})

		t.It("has affinity configured as expected", func() {
			if isMinVersion140 {
				assertPodAntiAffinity(map[string]string{"app": "cert-manager"}, constants.CertManagerNamespace)
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.4.0")
			}
		})
	})

	t.Describe("ingess-nginx", Label("f:platform-lcm.install"), func() {
		t.It("has expected controller deployment", func() {
			if isMinVersion110 {
				Eventually(func() (bool, error) {
					return pkg.DoesDeploymentExist(constants.IngressNamespace, constants.IngressController)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})
		t.It("has expected defaultBackend deployment", func() {
			if isMinVersion110 {
				Eventually(func() (bool, error) {
					return pkg.DoesDeploymentExist(constants.IngressNamespace, constants.IngressDefaultBackend)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})

		t.It("has correct number of  pods running", func() {
			if isMinVersion110 {
				validateCorrectNumberOfIngressNGINXPodsRunning()
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})

		t.It("has affinity configured as expected", func() {
			if isMinVersion140 {
				assertPodAntiAffinity(map[string]string{"app": "ingress-nginx"}, constants.IngressNamespace)
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.4.0")
			}
		})
	})

	t.Describe("istio-ingressgateway", Label("f:platform-lcm.install"), func() {
		t.It("has expected deployment", func() {
			if isMinVersion110 {
				Eventually(func() (bool, error) {
					return pkg.DoesDeploymentExist(constants.IstioSystemNamespace, "istio-ingressgateway")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})

		t.It("has correct number of pods running", func() {
			if isMinVersion110 {
				validateCorrectNumberOfPodsRunning("istio-ingressgateway", constants.IstioSystemNamespace)
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})

		t.It("has affinity configured as expected", func() {
			if isMinVersion120 {
				validateIstioGatewayAffinity("istio-ingressgateway", constants.IstioSystemNamespace)
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.2.0")
			}
		})
	})

	t.Describe("istio-egressgateway", Label("f:platform-lcm.install"), func() {
		t.It("has expected deployment", func() {
			if isMinVersion110 {
				Eventually(func() (bool, error) {
					return pkg.DoesDeploymentExist(constants.IstioSystemNamespace, "istio-egressgateway")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})

		t.It("has correct number of pods running", func() {
			if isMinVersion110 {
				validateCorrectNumberOfPodsRunning("istio-egressgateway", constants.IstioSystemNamespace)
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})

		t.It("has affinity configured as expected", func() {
			if isMinVersion120 {
				validateIstioGatewayAffinity("istio-egressgateway", constants.IstioSystemNamespace)
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.2.0")
			}
		})
	})

	if keycloakEnabled {
		t.Describe("keycloak", Label("f:platform-lcm.install"), func() {
			t.It("has expected statefulset", func() {
				if isMinVersion110 {
					Eventually(func() (bool, error) {
						return pkg.DoesStatefulSetExist(constants.KeycloakNamespace, constants.Keycloak)
					}, waitTimeout, pollingInterval).Should(BeTrue())
				} else {
					t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
				}
			})

			t.It("has correct number of pods running", func() {
				if isMinVersion110 {
					err := validateCorrectNumberOfPodsRunningSts(constants.Keycloak, constants.KeycloakNamespace, "app.kubernetes.io/name", constants.Keycloak)
					Expect(err).To(BeNil())
				} else {
					t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.1.0")
				}
			})

			t.It("has affinity configured as expected", func() {
				if isMinVersion140 {
					assertPodAntiAffinity(map[string]string{"app.kubernetes.io/name": constants.Keycloak}, constants.KeycloakNamespace)
				} else {
					t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.4.0")
				}
			})

			// check associated mysql elements
			t.It("has expected mysql statefulset", func() {
				if isMinVersion150 {
					Eventually(func() (bool, error) {
						return pkg.DoesStatefulSetExist(constants.KeycloakNamespace, "mysql")
					}, waitTimeout, pollingInterval).Should(BeTrue())
				} else {
					t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.5.0")
				}
			})

			t.It("has correct number of mysql server pods running", func() {
				if isMinVersion150 {
					err := validateCorrectNumberOfPodsRunningSts("mysql", constants.KeycloakNamespace, "app.kubernetes.io/name", "mysql-innodbcluster-mysql-server")
					Expect(err).To(BeNil())
				} else {
					t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.5.0")
				}
			})

			t.It("has expected mysql router deployment", func() {
				if isMinVersion150 {
					Eventually(func() (bool, error) {
						return pkg.DoesDeploymentExist(constants.KeycloakNamespace, "mysql-router")
					}, waitTimeout, pollingInterval).Should(BeTrue())
				} else {
					t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.5.0")
				}
			})

			t.It("has correct number of mysql router pods running", func() {
				if isMinVersion150 {
					validateCorrectNumberOfPodsRunning("mysql-router", constants.KeycloakNamespace)
				} else {
					t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.5.0")
				}
			})

			t.It("has affinity configured for mysql statefulset as expected", func() {
				if isMinVersion150 {
					assertPodAntiAffinity(map[string]string{"app.kubernetes.io/name": "mysql-innodbcluster-mysql-server"}, constants.KeycloakNamespace)
				} else {
					t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.5.0")
				}
			})

			t.It("has affinity configured for mysql router deployment as expected", func() {
				if isMinVersion150 {
					assertPodAntiAffinity(map[string]string{"app.kubernetes.io/name": "mysql-router"}, constants.KeycloakNamespace)
				} else {
					t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.5.0")
				}
			})
		})
	} else {
		t.Logs.Info("Skipping Keycloak check because it is not enabled")
	}

	// The MySQLOperator is enabled if Keycloak is enabled, or if it is explicitly enabled
	if keycloakEnabled || mySQLOperatorEnabled {
		t.It("has expected deployment", func() {
			if isMinVersion150 {
				Eventually(func() (bool, error) {
					return pkg.DoesDeploymentExist(constants.MySQLOperatorNamespace, mysqloperator.ComponentName)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.5.0")
			}
		})

		t.It("has correct number of pods running", func() {
			if isMinVersion150 {
				validateCorrectNumberOfPodsRunning(mysqloperator.ComponentName, constants.MySQLOperatorNamespace)
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not V1.5.0")
			}
		})

	} else {
		t.Logs.Info("Skipping MySQLOperator check because it is not enabled")

	}

	t.Describe("service resources in verrazzano-system", Label("f:platform-lcm.install"), func() {
		t.It("have expected Istio port names", func() {
			validateVerrazzanoSystemServicePorts()
		})
	})

})

func validateIstioGatewayAffinity(gwName string, gwNamespace string) {
	var pods []corev1.Pod
	Eventually(func() error {
		var err error
		pods, err = pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"app": gwName}}, gwNamespace)
		return err
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// Check the affinity configuration. Verify only a pod anti-affinity definition exists.
	for _, pod := range pods {
		affinity := pod.Spec.Affinity
		Expect(affinity).ToNot(BeNil())
		Expect(affinity.PodAffinity).To(BeNil())
		Expect(affinity.NodeAffinity).ToNot(BeNil())
		Expect(affinity.PodAntiAffinity).ToNot(BeNil())
		Expect(len(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
	}
}

func validateCorrectNumberOfPodsRunning(deployName string, nameSpace string) {
	// Get the deployment
	var deployment *appsv1.Deployment
	Eventually(func() (*appsv1.Deployment, error) {
		var err error
		deployment, err = pkg.GetDeployment(nameSpace, deployName)
		return deployment, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	var expectedPods = deployment.Spec.Replicas
	var pods []corev1.Pod
	Eventually(func() bool {
		var err error
		pods, err = pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"app": deployName}}, nameSpace)
		if err != nil {
			return false
		}
		// Compare the number of running pods to the expected number
		var runningPods int32 = 0
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodRunning {
				runningPods++
			}
		}
		return runningPods == *expectedPods
	}, waitTimeout, pollingInterval).Should(BeTrue())
}

func validateCorrectNumberOfIngressNGINXPodsRunning() {
	// Get the controller deployment
	var controllerDeployment *appsv1.Deployment
	Eventually(func() (*appsv1.Deployment, error) {
		var err error
		controllerDeployment, err = pkg.GetDeployment(constants.IngressNamespace, "ingress-controller-ingress-nginx-controller")
		return controllerDeployment, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	var defaultBackendDeployment *appsv1.Deployment
	Eventually(func() (*appsv1.Deployment, error) {
		var err error
		defaultBackendDeployment, err = pkg.GetDeployment(constants.IngressNamespace, constants.IngressDefaultBackend)
		return defaultBackendDeployment, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	var pods []corev1.Pod
	Eventually(func() bool {
		var err error
		pods, err = pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "ingress-nginx"}}, constants.IngressNamespace)
		if err != nil {
			return false
		}
		// Compare the number of running pods to the expected number
		var runningPods int32 = 0
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodRunning {
				runningPods++
			}
		}
		return runningPods == *controllerDeployment.Spec.Replicas+*defaultBackendDeployment.Spec.Replicas
	}, waitTimeout, pollingInterval).Should(BeTrue())
}

// validateCorrectNumberOfPodsRunningSts - validate the expected number of pods is running for a statefulset
func validateCorrectNumberOfPodsRunningSts(stsName string, nameSpace string, label string, labelValue string) error {
	// Get the deployment
	var statefulset *appsv1.StatefulSet
	Eventually(func() (*appsv1.StatefulSet, error) {
		var err error
		statefulset, err = pkg.GetStatefulSet(nameSpace, stsName)
		return statefulset, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	var expectedPods = statefulset.Spec.Replicas
	var pods []corev1.Pod
	Eventually(func() bool {
		var err error
		pods, err = pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{label: stsName}}, nameSpace)
		if err != nil {
			return false
		}
		// Compare the number of running pods to the expected number
		var runningPods int32 = 0
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodRunning {
				runningPods++
			}
		}
		return runningPods == *expectedPods
	}, waitTimeout, pollingInterval).Should(BeTrue())
	return nil
}

// Validate the verrazzano-system service ports to make sure they follow Istio conventions for
// naming ports.  Ports should have the prefix of "http-" or "https-" or be equal to "http" or "https".
func validateVerrazzanoSystemServicePorts() {
	// Get the list of verrazzano-system services
	var services *corev1.ServiceList
	Eventually(func() (*corev1.ServiceList, error) {
		var err error
		services, err = pkg.ListServices(constants.VerrazzanoSystemNamespace)
		return services, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	exceptions := []string{
		"coherence-operator-webhook",
		"internal-weblogic-operator-svc",
		mysqloperator.ComponentName,
	}
	for _, service := range services.Items {
		for _, port := range service.Spec.Ports {
			checkName := true
			for _, exception := range exceptions {
				if service.Name == exception {
					checkName = false
					break
				}
			}
			if checkName {
				hasPrefix := false
				if strings.Compare(port.Name, "http") == 0 || strings.Compare(port.Name, "https") == 0 ||
					strings.HasPrefix(port.Name, "http-") || strings.HasPrefix(port.Name, "https-") {
					hasPrefix = true
				}
				Expect(hasPrefix).Should(BeTrue(), fmt.Sprintf("Service \"%s\" port name \"%s\" is not a valid port name", service.Name, port.Name))
			}
		}
	}
}

func assertPodAntiAffinity(matchLabels map[string]string, namespace string) {
	var pods []corev1.Pod
	Eventually(func() error {
		var err error
		pods, err = pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: matchLabels}, namespace)
		return err
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// Check the affinity configuration. Verify only a pod anti-affinity definition exists.
	for _, pod := range pods {
		affinity := pod.Spec.Affinity
		Expect(affinity).ToNot(BeNil())
		Expect(affinity.PodAffinity).To(BeNil())
		Expect(affinity.NodeAffinity).To(BeNil())
		Expect(affinity.PodAntiAffinity).ToNot(BeNil())
		Expect(len(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
	}
}
