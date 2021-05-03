// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rbac_test

import (
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"strings"
	"time"
)

var (
	expectedPodsOperator = []string{"verrazzano-application-operator"}
	expectedPodsOam      = []string{"oam-kubernetes-runtime"}
	waitTimeout          = 5 * time.Minute
	pollingInterval      = 10 * time.Second
)

const (
	verrazzanoSystemNS = "verrazzano-system"
	rbacTestNamespace  = "rbactest"
	v80ProjectAdmin    = "ocid1.user.oc1..aaaaaaaallodotxfvg0g1antsyq3gonyyhblya66kiqjnp2kogonykvjwi19"
	v80ProjectMonitor  = "ocid1.user.oc1..aaaaaaaallodotxfvg0yank33sq3gonyghblya66kiqjnp2kogonykvjwi19"
	verrazzanoAPI      = "verrazzano-api"
	impersonateVerb    = "impersonate"
)

var _ = ginkgo.BeforeSuite(func() {
	pkg.Log(pkg.Info, "Create namespace")
	if _, err := pkg.CreateNamespace(rbacTestNamespace, map[string]string{"verrazzano-managed": "true", "istio-injection": "enabled"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}
})

var failed = false
var _ = ginkgo.AfterEach(func() {
	failed = failed || ginkgo.CurrentGinkgoTestDescription().Failed
})

var _ = ginkgo.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	pkg.Log(pkg.Info, "Delete namespace")
	if err := pkg.DeleteNamespace(rbacTestNamespace); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the namespace: %v", err))
	}
	gomega.Eventually(func() bool {
		ns, err := pkg.GetNamespace(rbacTestNamespace)
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
})

var _ = ginkgo.Describe("Test RBAC Permission", func() {
	ginkgo.Context("for user with role verrazzano-project-admin", func() {

		ginkgo.It("Fail getting Pods in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace rbactest?  No")
			allowed, reason := pkg.CanI(v80ProjectAdmin, rbacTestNamespace, "list", "pods")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user list pods: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Create RoleBinding Admin for verrazzano-project-admin", func() {
			pkg.Log(pkg.Info, "Create RoleBinding Admin for verrazzano-project-admin")
			err := pkg.CreateRoleBinding(v80ProjectAdmin, rbacTestNamespace, "admin-binding", "admin")
			gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Admin RoleBinding creation failed: reason = %s", err))
		})

		ginkgo.It("Verify RoleBinding Admin for verrazzano-project-admin", func() {
			verifyRoleBindingExists("admin-binding")
		})

		ginkgo.It("Succeed getting Pods in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace rbactest?  Yes")
			gomega.Eventually(func() bool {
				allowed, _ := pkg.CanI(v80ProjectAdmin, rbacTestNamespace, "list", "pods")
				return allowed
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "FAIL: Did Not Pass Authorization on user list pods")
		})

		ginkgo.It("Fail getting Pods in namespace verrazzano-system", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace verrazzano-system?  No")
			allowed, reason := pkg.CanI(v80ProjectAdmin, verrazzanoSystemNS, "list", "pods")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user list pods: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Fail create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  No")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user create ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Fail list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  No")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user list ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Fail create OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  No")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "components", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user create OAM Components: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Fail list OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  No")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "components", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user list OAM Components: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Create RoleBinding verrazzano-project-admin-binding for verrazzano-project-admin", func() {
			pkg.Log(pkg.Info, "Create RoleBinding verrazzano-project-admin-binding for verrazzano-project-admin")
			err := pkg.CreateRoleBinding(v80ProjectAdmin, rbacTestNamespace, "verrazzano-project-admin-binding", "verrazzano-project-admin")
			gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: VerrazzanoProjectAdmin RoleBinding creation failed: reason = %s", err))
		})

		ginkgo.It("Verify RoleBinding verrazzano-project-admin-binding for verrazzano-project-admin", func() {
			verifyRoleBindingExists("verrazzano-project-admin-binding")
		})

		ginkgo.It("Succeed create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  Yes")
			gomega.Eventually(func() bool {
				allowed, _ := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev")
				return allowed
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "FAIL: Did Not Pass Authorization on user create ApplicationConfiguration")
		})

		ginkgo.It("Succeed list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  Yes")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeTrue(), fmt.Sprintf("FAIL: Did Not Pass Authorization on user list ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Succeed create OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  Yes")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "components", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeTrue(), fmt.Sprintf("FAIL: Did Not Pass Authorization on user create OAM Components: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Succeed list OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  Yes")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "components", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeTrue(), fmt.Sprintf("FAIL: Did Not Pass Authorization on user list OAM Components: Allowed = %t, reason = %s", allowed, reason))
		})

	})
})

var _ = ginkgo.Describe("Test RBAC Permission", func() {
	ginkgo.Context("for user with role verrazzano-project-monitor", func() {

		ginkgo.It("Fail getting Pods in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace rbactest?  No")
			allowed, reason := pkg.CanI(v80ProjectMonitor, rbacTestNamespace, "list", "pods")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user list pods: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Create RoleBinding Admin for verrazzano-project-monitor", func() {
			pkg.Log(pkg.Info, "Create RoleBinding Admin for verrazzano-project-monitor")
			err := pkg.CreateRoleBinding(v80ProjectMonitor, rbacTestNamespace, "monitor-binding", "admin")
			gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Admin RoleBinding creation failed: reason = %s", err))
		})

		ginkgo.It("Verify RoleBinding monitor-binding for verrazzano-project-monitor", func() {
			verifyRoleBindingExists("monitor-binding")
		})

		ginkgo.It("Succeed getting Pods in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace rbactest?  Yes")
			gomega.Eventually(func() bool {
				allowed, _ := pkg.CanI(v80ProjectMonitor, rbacTestNamespace, "list", "pods")
				return allowed
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "FAIL: Did Not Pass Authorization on user list pods")
		})

		ginkgo.It("Fail getting Pods in namespace verrazzano-system", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace verrazzano-system?  No")
			allowed, reason := pkg.CanI(v80ProjectMonitor, verrazzanoSystemNS, "list", "pods")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user list pods: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Fail create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  No")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user create ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Fail list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  No")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user list ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Fail create OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  No")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "components", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user create OAM Components: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Fail list OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  No")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "components", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user list OAM Components: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Create RoleBinding verrazzano-project-monitor-binding for cluster role verrazzano-project-monitor", func() {
			pkg.Log(pkg.Info, "Create RoleBinding verrazzano-project-monitor-binding for verrazzano-project-monitor")
			err := pkg.CreateRoleBinding(v80ProjectMonitor, rbacTestNamespace, "verrazzano-project-monitor-binding", "verrazzano-project-monitor")
			gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: VerrazzanoProjectAdmin RoleBinding creation failed: reason = %s", err))
		})

		ginkgo.It("Verify RoleBinding verrazzano-project-monitor-binding for verrazzano-project-monitor", func() {
			verifyRoleBindingExists("verrazzano-project-monitor-binding")
		})

		ginkgo.It("Fail create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  No")
			gomega.Eventually(func() bool {
				allowed, _ := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev")
				return allowed
			}, waitTimeout, pollingInterval).Should(gomega.BeFalse(), "FAIL: Passed Authorization on user create ApplicationConfiguration")
		})

		ginkgo.It("Succeed list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  Yes")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeTrue(), fmt.Sprintf("FAIL: Did Not Pass Authorization on user list ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Fail create OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  No")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "components", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on user create OAM Components: Allowed = %t, reason = %s", allowed, reason))
		})

		ginkgo.It("Succeed list OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  Yes")
			allowed, reason := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "components", "core.oam.dev")
			gomega.Expect(allowed).To(gomega.BeTrue(), fmt.Sprintf("FAIL: Did Not Pass Authorization on user list OAM Components: Allowed = %t, reason = %s", allowed, reason))
		})

	})
})

var _ = ginkgo.Describe("Test Verrazzano API Service Account", func() {
	ginkgo.Context("for serviceaccount verrazzano-api", func() {
		var apiProxy corev1.Pod
		sa := pkg.GetServiceAccount(verrazzanoSystemNS, verrazzanoAPI)
		ginkgo.It("Validate the secret of the Service Account of Verrazzano API", func() {
			// Get secret for the SA
			saSecret := sa.Secrets[0]
			clientset := pkg.GetKubernetesClientset()
			pods := pkg.ListPodsInCluster(verrazzanoSystemNS, clientset)
			secretMatched := false
			for i := range pods.Items {
				// Get the secret of the API proxy pod
				if strings.HasPrefix(pods.Items[i].Name, verrazzanoAPI) {
					apiProxy = pods.Items[i]
					for j := range apiProxy.Spec.Volumes {
						if apiProxy.Spec.Volumes[j].Secret != nil && apiProxy.Spec.Volumes[j].Secret.SecretName == saSecret.Name {
							secretMatched = true
							break
						}
					}
					break
				}
			}
			gomega.Expect(secretMatched).To(gomega.BeTrue(), fmt.Sprintf("FAIL: The secret name of ServiceAccount "+
				"%s differs from the secret obtained from the Verrazzano API pod.", verrazzanoAPI))
		})

		ginkgo.It("Validate the role binding of the Service Account of Verrazzano API", func() {
			bindings := pkg.ListClusterRoleBindings()
			saName := sa.Name
			bcount := 0
			var rbinding v1.ClusterRoleBinding
			for rb := range bindings.Items {
				for sa := range bindings.Items[rb].Subjects {
					// Get cluster role bindings for verrazzano-api
					if bindings.Items[rb].Subjects[sa].Name == saName {
						rbinding = bindings.Items[rb]
						bcount++
					}
				}
			}
			// There should be a single cluster role binding, which references service account of Verrazzano API
			gomega.Expect(bcount > 1).To(gomega.BeFalse())
			gomega.Expect(len(rbinding.Subjects) > 1).To(gomega.BeFalse(),
				fmt.Sprintf("FAIL: There are more than one Subjects for the cluster role binding %s", rbinding.Subjects))

			// There should be a single subject of kind service account in verrazzano-system namespace
			gomega.Expect(rbinding.Subjects[0].Kind == "ServiceAccount").To(gomega.BeTrue(),
				fmt.Sprintf("FAIL: The KIND for service account %s is different than ServiceAccount.", rbinding.Subjects[0].Kind))
			gomega.Expect(rbinding.Subjects[0].Namespace == verrazzanoSystemNS).To(gomega.BeTrue(),
				fmt.Sprintf("FAIL: The namespace for service account %s is different than %s.",
					rbinding.Subjects[0].Namespace, verrazzanoSystemNS))

			// There should be a single rule with resources - users and groups, and verbs impersonate
			crole := pkg.GetClusterRole(rbinding.RoleRef.Name)
			gomega.Expect(len(crole.Rules) == 1).To(gomega.BeTrue(),
				fmt.Sprintf("FAIL: The cluster role %s contains more than one rules.", crole))

			crule := crole.Rules[0]
			gomega.Expect(len(crule.Resources) == 2).To(gomega.BeTrue(),
				fmt.Sprintf("FAIL: There are more resources than the expected users and groups %s", crule.Resources))
			res := make(map[string]bool)
			res["users"] = true
			res["groups"] = true
			for r := range crule.Resources {
				delete(res, crule.Resources[r])
			}
			gomega.Expect(len(res) == 0).To(gomega.BeTrue(),
				fmt.Sprintf("FAIL: The rule contains resource(s) other than expected users and groups - %s", crule.Resources))

			verbs := crule.Verbs
			gomega.Expect(len(verbs) == 1).To(gomega.BeTrue(),
				fmt.Sprintf("FAIL: The cluster role %s contains more than one verbs.", crole))
			gomega.Expect(verbs[0] == impersonateVerb).To(gomega.BeTrue(),
				fmt.Sprintf("FAIL: The cluster role %s contains verb other than impersonate.", crole))
		})
	})
})

func verifyRoleBindingExists(name string) {
	gomega.Eventually(func() bool {
		rbExists := pkg.DoesRoleBindingExist(name, rbacTestNamespace)
		return rbExists
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeTrue())
}
