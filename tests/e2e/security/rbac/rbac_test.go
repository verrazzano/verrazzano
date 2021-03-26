// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rbac_test

import (
	"fmt"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	"github.com/onsi/ginkgo"
)

var (
	expectedPodsOperator = []string{"verrazzano-application-operator"}
	expectedPodsOam      = []string{"oam-kubernetes-runtime"}
	waitTimeout          = 10 * time.Minute
	pollingInterval      = 30 * time.Second
)

const (
	verrazzanoSystemNS = "verrazzano-system"
	rbacTestNamespace  = "rbactest"
	v80ProjectAdmin    = "ocid1.user.oc1..aaaaaaaallodotxfvg0g1antsyq3gonyyhblya66kiqjnp2kogonykvjwi19"
	v80ProjectMonitor  = "ocid1.user.oc1..aaaaaaaallodotxfvg0yank33sq3gonyghblya66kiqjnp2kogonykvjwi19"
	// The tests can run so fast against a Kind cluster that a pause is put in after rolebindings are created for test stability
	sleepNumMS = 500 * time.Millisecond
)

var _ = ginkgo.BeforeSuite(func() {
	pkg.Log(pkg.Info, "Create namespace")
	if _, err := pkg.CreateNamespace(rbacTestNamespace, map[string]string{"verrazzano-managed": "true", "istio-injection": "enabled"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}
})

var _ = ginkgo.AfterSuite(func() {
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
			if allowed, reason := pkg.CanI(v80ProjectAdmin, rbacTestNamespace, "list", "pods"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user list pods: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Create RoleBinding Admin for verrazzano-project-admin", func() {
			pkg.Log(pkg.Info, "Create RoleBinding Admin for verrazzano-project-admin")
			if err := pkg.CreateRoleBinding(v80ProjectAdmin, rbacTestNamespace, "admin-binding", "admin"); err != nil {
				ginkgo.Fail(fmt.Sprintf("FAIL: RoleBinding creation failed: reason = %s", err))
			}
		})

		time.Sleep(sleepNumMS)

		ginkgo.It("Succeed getting Pods in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace rbactest?  Yes")
			if allowed, reason := pkg.CanI(v80ProjectAdmin, rbacTestNamespace, "list", "pods"); allowed == false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Did Not Pass Authorization on user list pods: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Fail getting Pods in namespace verrazzano-system", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace verrazzano-system?  No")
			if allowed, reason := pkg.CanI(v80ProjectAdmin, verrazzanoSystemNS, "list", "pods"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user list pods: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Fail create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  No")
			if allowed, reason := pkg.CanIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user create ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Fail list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  No")
			if allowed, reason := pkg.CanIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user list ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Fail create OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  No")
			if allowed, reason := pkg.CanIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "components", "core.oam.dev"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user create OAM Components: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Fail list OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  No")
			if allowed, reason := pkg.CanIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "components", "core.oam.dev"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user list OAM Components: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Create RoleBinding verrazzano-project-admin-binding for verrazzano-project-admin", func() {
			pkg.Log(pkg.Info, "Create RoleBinding verrazzano-project-admin-binding for verrazzano-project-admin")
			if err := pkg.CreateRoleBinding(v80ProjectAdmin, rbacTestNamespace, "verrazzano-project-admin-binding", "verrazzano-project-admin"); err != nil {
				ginkgo.Fail(fmt.Sprintf("FAIL: RoleBinding creation failed: reason = %s", err))
			}
		})

		time.Sleep(sleepNumMS)

		ginkgo.It("Succeed create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  Yes")
			if allowed, reason := pkg.CanIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev"); allowed == false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Did Not Pass Authorization on user create ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Succeed list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  Yes")
			if allowed, reason := pkg.CanIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev"); allowed == false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Did Not Pass Authorization on user list ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Succeed create OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  Yes")
			if allowed, reason := pkg.CanIGroup(v80ProjectAdmin, rbacTestNamespace, "create", "components", "core.oam.dev"); allowed == false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Did Not Pass Authorization on user create OAM Components: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Succeed list OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  Yes")
			if allowed, reason := pkg.CanIGroup(v80ProjectAdmin, rbacTestNamespace, "list", "components", "core.oam.dev"); allowed == false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Did Not Pass Authorization on user list OAM Components: Allowed = %t, reason = %s", allowed, reason))
			}
		})

	})
})

var _ = ginkgo.Describe("Test RBAC Permission", func() {
	ginkgo.Context("for user with role verrazzano-project-monitor", func() {

		ginkgo.It("Fail getting Pods in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace rbactest?  No")
			if allowed, reason := pkg.CanI(v80ProjectMonitor, rbacTestNamespace, "list", "pods"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user list pods: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Create RoleBinding Admin for verrazzano-project-monitor", func() {
			pkg.Log(pkg.Info, "Create RoleBinding Admin for verrazzano-project-monitor")
			if err := pkg.CreateRoleBinding(v80ProjectMonitor, rbacTestNamespace, "monitor-binding", "admin"); err != nil {
				ginkgo.Fail(fmt.Sprintf("FAIL: RoleBinding creation failed: reason = %s", err))
			}
		})

		time.Sleep(sleepNumMS)

		ginkgo.It("Succeed getting Pods in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace rbactest?  Yes")
			if allowed, reason := pkg.CanI(v80ProjectMonitor, rbacTestNamespace, "list", "pods"); allowed == false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Did Not Pass Authorization on user list pods: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Fail getting Pods in namespace verrazzano-system", func() {
			pkg.Log(pkg.Info, "Can User List Pods in NameSpace verrazzano-system?  No")
			if allowed, reason := pkg.CanI(v80ProjectMonitor, verrazzanoSystemNS, "list", "pods"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user list pods: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Fail create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  No")
			if allowed, reason := pkg.CanIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user create ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Fail list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  No")
			if allowed, reason := pkg.CanIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user list ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Fail create OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  No")
			if allowed, reason := pkg.CanIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "components", "core.oam.dev"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user create OAM Components: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Fail list OAM Component in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  No")
			if allowed, reason := pkg.CanIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "components", "core.oam.dev"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user list OAM Components: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Create RoleBinding verrazzano-project-monitor-binding for cluster role verrazzano-project-monitor", func() {
			pkg.Log(pkg.Info, "Create RoleBinding verrazzano-project-monitor-binding for verrazzano-project-monitor")
			if err := pkg.CreateRoleBinding(v80ProjectMonitor, rbacTestNamespace, "verrazzano-project-monitor-binding", "verrazzano-project-monitor"); err != nil {
				ginkgo.Fail(fmt.Sprintf("FAIL: RoleBinding creation failed: reason = %s", err))
			}
		})

		time.Sleep(sleepNumMS)

		ginkgo.It("Fail create ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create ApplicationConfiguration in NameSpace rbactest?  No")
			if allowed, reason := pkg.CanIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "applicationconfigurations", "core.oam.dev"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user create ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Succeed list ApplicationConfiguration in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list ApplicationConfiguration in NameSpace rbactest?  Yes")
			if allowed, reason := pkg.CanIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "applicationconfigurations", "core.oam.dev"); allowed == false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Did Not Pass Authorization on user list ApplicationConfiguration: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Fail create OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User create OAM Components in NameSpace rbactest?  No")
			if allowed, reason := pkg.CanIGroup(v80ProjectMonitor, rbacTestNamespace, "create", "components", "core.oam.dev"); allowed != false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Passed Authorization on user create OAM Components: Allowed = %t, reason = %s", allowed, reason))
			}
		})

		ginkgo.It("Succeed list OAM Components in namespace rbactest", func() {
			pkg.Log(pkg.Info, "Can User list OAM Components in NameSpace rbactest?  Yes")
			if allowed, reason := pkg.CanIGroup(v80ProjectMonitor, rbacTestNamespace, "list", "components", "core.oam.dev"); allowed == false {
				ginkgo.Fail(fmt.Sprintf("FAIL: Did Not Pass Authorization on user list OAM Components: Allowed = %t, reason = %s", allowed, reason))
			}
		})

	})
})
