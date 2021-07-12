// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rbac_test

import (
	"fmt"
	"time"

	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

var (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 10 * time.Second
)

const (
	verrazzanoSystemNS = "verrazzano-system"
	rbacTestNamespace  = "rbactest"
	v80ProjectAdmin    = "ocid1.user.oc1..aaaaaaaallodotxfvg0g1antsyq3gonyyhblya66kiqjnp2kogonykvjwi19"
	v80ProjectMonitor  = "ocid1.user.oc1..aaaaaaaallodotxfvg0yank33sq3gonyghblya66kiqjnp2kogonykvjwi19"
	verrazzanoAPI      = "verrazzano-api"
	impersonateVerb    = "impersonate"
)

var _ = BeforeSuite(func() {
	pkg.Log(pkg.Info, "Create namespace")
	Eventually(func() (*corev1.Namespace, error) {
		return pkg.CreateNamespace(rbacTestNamespace, map[string]string{"verrazzano-managed": "true", "istio-injection": "enabled"})
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())
})

var failed = false
var _ = AfterEach(func() {
	failed = failed || CurrentGinkgoTestDescription().Failed
})

var _ = AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(rbacTestNamespace)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(rbacTestNamespace)
		return err != nil && errors.IsNotFound(err)
	}, waitTimeout, pollingInterval).Should(BeTrue())
})

var _ = Describe("Test RBAC Permission", func() {
	Context("for user with role verrazzano-project-admin", func() {

		It("Fail getting Pods in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanI(v80ProjectAdmin, rbacTestNamespace, "list", "pods")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL Authorization on user list pods: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user list pods. Timeout Expired")
		})

		It("Create RoleBinding Admin for verrazzano-project-admin", func() {
			pkg.Log(pkg.Info, "Create RoleBinding Admin for verrazzano-project-admin")
			Eventually(func() error {
				return pkg.CreateRoleBinding(v80ProjectAdmin, rbacTestNamespace, "admin-binding", "admin")
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		It("Verify RoleBinding Admin for verrazzano-project-admin", func() {
			verifyRoleBindingExists("admin-binding")
		})

		It("Succeed getting Pods in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace rbactest?  Yes")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanI(v80ProjectAdmin, rbacTestNamespace, "list", "pods")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should SUCCEED on user list pods in rbactest namespace: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "FAIL: Did Not Pass Authorization on user list pods. Timeout Expired")
		})

		It("Fail getting Pods in namespace verrazzano-system", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace verrazzano-system?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanI(v80ProjectAdmin, verrazzanoSystemNS, "list", "pods")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user list pods in verrazzano-system namespace: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user list pods. Timeout Expired")
		})

		It("Fail create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user create ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user create ApplicationConfiguration. Timeout Expired")
		})

		It("Fail list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user list ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user list ApplicationConfiguration. Timeout Expired")
		})

		It("Fail create OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "components", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user create OAM Components: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user create OAM Components. Timeout Expired")
		})

		It("Fail list OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "components", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user list OAM Components: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user list OAM Components. Timeout Expired")
		})

		It("Create RoleBinding verrazzano-project-admin-binding for verrazzano-project-admin", func() {
			pkg.Log(pkg.Info, "Create RoleBinding verrazzano-project-admin-binding for verrazzano-project-admin")
			Eventually(func() error {
				return pkg.CreateRoleBinding(v80ProjectAdmin, rbacTestNamespace, "verrazzano-project-admin-binding", "verrazzano-project-admin")
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		It("Verify RoleBinding verrazzano-project-admin-binding for verrazzano-project-admin", func() {
			verifyRoleBindingExists("verrazzano-project-admin-binding")
		})

		It("Succeed create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  Yes")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should SUCCEED on user create ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "FAIL: Did Not Pass Authorization on user create ApplicationConfiguration. Timeout Expired")
		})

		It("Succeed list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  Yes")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should SUCCEED on user list ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "FAIL: Did Not Pass Authorization on user list ApplicationConfiguration. Timeout Expired")
		})

		It("Succeed create OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  Yes")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "components", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should SUCCEED on user create OAM Components: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "FAIL: Did Not Pass Authorization on user create OAM Components. Timeout Expired")
		})

		It("Succeed list OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  Yes")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "components", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should SUCCEED on user list OAM Components: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "FAIL: Did Not Pass Authorization on user list OAM Components. Timeout Expired")
		})

	})
})

var _ = Describe("Test RBAC Permission", func() {
	Context("for user with role verrazzano-project-monitor", func() {

		It("Fail getting Pods in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanI(v80ProjectMonitor, rbacTestNamespace, "list", "pods")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user list pods in rbactest namespace: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user list pods. Timeout Expired")
		})

		It("Create RoleBinding Admin for verrazzano-project-monitor", func() {
			pkg.Log(pkg.Info, "Create RoleBinding Admin for verrazzano-project-monitor")
			Eventually(func() error {
				return pkg.CreateRoleBinding(v80ProjectMonitor, rbacTestNamespace, "monitor-binding", "admin")
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		It("Verify RoleBinding monitor-binding for verrazzano-project-monitor", func() {
			verifyRoleBindingExists("monitor-binding")
		})

		It("Succeed getting Pods in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace rbactest?  Yes")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanI(v80ProjectMonitor, rbacTestNamespace, "list", "pods")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should SUCCEED on user list pods in rbactest namespace: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "FAIL: Did Not Pass Authorization on user list pods. Timeout Expired")
		})

		It("Fail getting Pods in namespace verrazzano-system", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace verrazzano-system?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanI(v80ProjectMonitor, verrazzanoSystemNS, "list", "pods")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user list pods in verrazzano-system namespace: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user list pods. Timeout Expired")
		})

		It("Fail create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user create ApplicationConfiguration. Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user create ApplicationConfiguration. Timeout Expired")
		})

		It("Fail list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user list ApplicationConfiguration. Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user list ApplicationConfiguration. Timeout Expired")
		})

		It("Fail create OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "components", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user create OAM Components. Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user create OAM Components. Timeout Expired")
		})

		It("Fail list OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "components", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user list OAM Components. Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user list OAM Components. Timeout Expired")
		})

		It("Create RoleBinding verrazzano-project-monitor-binding for cluster role verrazzano-project-monitor", func() {
			pkg.Log(pkg.Info, "Create RoleBinding verrazzano-project-monitor-binding for verrazzano-project-monitor")
			Eventually(func() error {
				return pkg.CreateRoleBinding(v80ProjectMonitor, rbacTestNamespace, "verrazzano-project-monitor-binding", "verrazzano-project-monitor")
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		It("Verify RoleBinding verrazzano-project-monitor-binding for verrazzano-project-monitor", func() {
			verifyRoleBindingExists("verrazzano-project-monitor-binding")
		})

		It("Fail create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user create ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user create ApplicationConfiguration. Timeout Expired")
		})

		It("Succeed list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  Yes")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should SUCCEED on user list ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "FAIL: Did Not Pass Authorization on user list ApplicationConfiguration.  Timeout Expired")
		})

		It("Fail create OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  No")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "components", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should FAIL on user create OAM Components: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeFalse(), "FAIL: Passed Authorization on user create OAM Components.  Timeout Expired")
		})

		It("Succeed list OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  Yes")
			Eventually(func() (bool, error) {
				allowed, reason, err := pkg.CanIForAPIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "components", "core.oam.dev")
				pkg.Log(pkg.Info, fmt.Sprintf("Status: Should SUCCEED on user list OAM Components: Allowed = %t, reason = %s", allowed, reason))
				return allowed, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "FAIL: Did Not Pass Authorization on user list OAM Components.  Timeout Expired")
		})

	})
})

var _ = Describe("Test Verrazzano API Service Account", func() {
	Context("for serviceaccount verrazzano-api", func() {
		var serviceAccount *corev1.ServiceAccount

		BeforeEach(func() {
			Eventually(func() (*corev1.ServiceAccount, error) {
				var err error
				serviceAccount, err = pkg.GetServiceAccount(verrazzanoSystemNS, verrazzanoAPI)
				return serviceAccount, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil(), fmt.Sprintf("Failed to get service account %s in namespace %s", verrazzanoAPI, verrazzanoSystemNS))
		})

		It("Validate the secret of the Service Account of Verrazzano API", func() {
			// Get secret for the SA
			var pods *corev1.PodList
			saSecret := serviceAccount.Secrets[0]
			Eventually(func() (*corev1.PodList, error) {
				var err error
				clientset := pkg.GetKubernetesClientset()
				pods, err = pkg.ListPodsInCluster(verrazzanoSystemNS, clientset)
				return pods, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			secretMatched := false
			for i := range pods.Items {
				// Get the secret of the API proxy pod
				if strings.HasPrefix(pods.Items[i].Name, verrazzanoAPI) {
					apiProxy := pods.Items[i]
					for j := range apiProxy.Spec.Volumes {
						if apiProxy.Spec.Volumes[j].Secret != nil && apiProxy.Spec.Volumes[j].Secret.SecretName == saSecret.Name {
							secretMatched = true
							break
						}
					}
					break
				}
			}
			Expect(secretMatched).To(BeTrue(), fmt.Sprintf("FAIL: The secret name of ServiceAccount "+
				"%s differs from the secret obtained from the Verrazzano API pod.", verrazzanoAPI))
		})

		It("Validate the role binding of the Service Account of Verrazzano API", func() {
			var bindings *v1.ClusterRoleBindingList
			Eventually(func() (*v1.ClusterRoleBindingList, error) {
				var err error
				bindings, err = pkg.ListClusterRoleBindings()
				return bindings, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			saName := serviceAccount.Name
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
			Expect(bcount > 1).To(BeFalse())
			Expect(len(rbinding.Subjects) > 1).To(BeFalse(),
				fmt.Sprintf("FAIL: There are more than one Subjects for the cluster role binding %s", rbinding.Subjects))

			// There should be a single subject of kind service account in verrazzano-system namespace
			Expect(rbinding.Subjects[0].Kind == "ServiceAccount").To(BeTrue(),
				fmt.Sprintf("FAIL: The KIND for service account %s is different than ServiceAccount.", rbinding.Subjects[0].Kind))
			Expect(rbinding.Subjects[0].Namespace == verrazzanoSystemNS).To(BeTrue(),
				fmt.Sprintf("FAIL: The namespace for service account %s is different than %s.",
					rbinding.Subjects[0].Namespace, verrazzanoSystemNS))

			// There should be a single rule with resources - users and groups, and verbs impersonate
			var crole *v1.ClusterRole
			Eventually(func() (*v1.ClusterRole, error) {
				var err error
				crole, err = pkg.GetClusterRole(rbinding.RoleRef.Name)
				return crole, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			Expect(len(crole.Rules) == 1).To(BeTrue(),
				fmt.Sprintf("FAIL: The cluster role %v contains more than one rules.", crole))

			crule := crole.Rules[0]
			Expect(len(crule.Resources) == 2).To(BeTrue(),
				fmt.Sprintf("FAIL: There are more resources than the expected users and groups %s", crule.Resources))
			res := make(map[string]bool)
			res["users"] = true
			res["groups"] = true
			for r := range crule.Resources {
				delete(res, crule.Resources[r])
			}
			Expect(len(res) == 0).To(BeTrue(),
				fmt.Sprintf("FAIL: The rule contains resource(s) other than expected users and groups - %s", crule.Resources))

			verbs := crule.Verbs
			Expect(len(verbs) == 1).To(BeTrue(),
				fmt.Sprintf("FAIL: The cluster role %s contains more than one verbs.", crole))
			Expect(verbs[0] == impersonateVerb).To(BeTrue(),
				fmt.Sprintf("FAIL: The cluster role %s contains verb other than impersonate.", crole))
		})

		It("Fail impersonating any other service account", func() {
			pkg.Log(pkg.Info, "Can verrazzano-api service account impersonate any other service account?  No")
			allowed, reason, err := pkg.CanIForAPIGroupForServiceAccountOrUser("verrazzano-api", "", "impersonate", "serviceaccounts", "core", true, verrazzanoSystemNS)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(allowed).To(BeFalse(), fmt.Sprintf("FAIL: Passed Authorization on impersonating service accounts: Allowed = %t, reason = %s", allowed, reason))
		})

	})
})

func verifyRoleBindingExists(name string) {
	Eventually(func() bool {
		return pkg.DoesRoleBindingExist(name, rbacTestNamespace)
	}, waitTimeout, pollingInterval).Should(BeTrue())
}
