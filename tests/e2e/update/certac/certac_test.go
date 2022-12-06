// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certac

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	aocnst "github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	pocnst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"sigs.k8s.io/yaml"
)

var (
	t               = framework.NewTestFramework("update fluentd external opensearch")
	adminCluster    *multicluster.Cluster
	managedClusters []*multicluster.Cluster
	originalCM      *vzapi.CertManagerComponent
	originalFluentd *vzapi.FluentdComponent
	waitTimeout     = 10 * time.Minute
	pollingInterval = 5 * time.Second
)

type CertModifier struct {
	CertManager *vzapi.CertManagerComponent
}

func (u *CertModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.CertManager = u.CertManager
}

var beforeSuite = t.BeforeSuiteFunc(func() {
	adminCluster = multicluster.AdminCluster()
	managedClusters = multicluster.ManagedClusters()
	adminVZ := adminCluster.GetCR(true)
	originalCM = adminVZ.Spec.Components.CertManager
	if !isDefaultCM(originalCM) {
		Fail("TestAdminClusterCertManagerUpdate requires default CertManager in AdminCluster")
	}
	originalFluentd = adminVZ.Spec.Components.Fluentd
	verifyRegistration()
})

var _ = BeforeSuite(beforeSuite)

var afterSuite = t.AfterSuiteFunc(func() {
	// verifyManagedClustersFluentdConnection()
})

var _ = AfterSuite(afterSuite)

var _ = t.Describe("Update admin-cluster cert-manager", Label("f:platform-lcm.update"), func() {
	t.Describe("multicluster cert-manager verify", Label("f:platform-lcm.multicluster-verify"), func() {
		t.It("admin-cluster cert-manager custom CA", func() {
			start := time.Now()
			oldIngressCaCrt := updateAdminClusterCA()
			// reapply registration for managed cluster - for self-signed certs, this manual step
			// is expected of users so that the admin kubeconfig used by the managed cluster can be
			// reliably updated to use the new CA cert, or intermittent timing related failures may occur
			reapplyManagedClusterRegManifest()
			verifyCaSync(oldIngressCaCrt)
			// verify new logs are flowing after updating admin cert
			verifyManagedFluentd(start)
		})
	})
	t.Describe("multicluster cert-manager verify cleanup", Label("f:platform-lcm.multicluster-verify"), func() {
		t.It("admin-cluster cert-manager revert to default self-signed CA", func() {
			start := time.Now()
			oldIngressCaCrt := revertToDefaultCertManager()
			verifyCaSync(oldIngressCaCrt)
			verifyManagedFluentd(start)
		})
	})
})

func updateAdminClusterCA() string {
	oldIngressCaCrt := adminCluster.
		GetSecretDataAsString(constants.VerrazzanoSystemNamespace, pocnst.VerrazzanoIngressSecret, mcconstants.CaCrtKey)
	genCA := adminCluster.GenerateCA()
	newCM := &vzapi.CertManagerComponent{
		Certificate: vzapi.Certificate{CA: vzapi.CA{SecretName: genCA, ClusterResourceNamespace: constants.CertManagerNamespace}},
	}
	m := &CertModifier{CertManager: newCM}
	update.RetryUpdate(m, adminCluster.KubeConfigPath, false, pollingInterval, waitTimeout)
	return oldIngressCaCrt
}

func isDefaultCM(cm *vzapi.CertManagerComponent) bool {
	return cm == nil || reflect.DeepEqual(*cm, vzapi.CertManagerComponent{})
}

func isDefaultFluentd(fl *vzapi.FluentdComponent) bool {
	return fl == nil || reflect.DeepEqual(*fl, vzapi.FluentdComponent{})
}

func revertToDefaultCertManager() string {
	oldIngressCaCrt := adminCluster.
		GetSecretDataAsString(constants.VerrazzanoSystemNamespace, pocnst.VerrazzanoIngressSecret, mcconstants.CaCrtKey)
	m := &CertModifier{CertManager: originalCM}
	update.RetryUpdate(m, adminCluster.KubeConfigPath, false, pollingInterval, waitTimeout)
	return oldIngressCaCrt
}

func verifyCaSync(oldIngressCaCrt string) {
	newIngressCaCrt := verifyAdminClusterIngressCA(oldIngressCaCrt)
	verifyCaSyncToManagedClusters(newIngressCaCrt)
}

func verifyAdminClusterIngressCA(oldIngressCaCrt string) string {
	start := time.Now()
	gomega.Eventually(func() bool {
		newIngressCaCrt := adminCluster.
			GetSecretDataAsString(constants.VerrazzanoSystemNamespace, pocnst.VerrazzanoIngressSecret, mcconstants.CaCrtKey)
		if newIngressCaCrt == oldIngressCaCrt {
			pkg.Log(pkg.Error, fmt.Sprintf("%v of %v is not updated", pocnst.VerrazzanoIngressSecret, adminCluster.Name))
		} else {
			pkg.Log(pkg.Error, fmt.Sprintf("%v of %v took %v updated", pocnst.VerrazzanoIngressSecret, adminCluster.Name, time.Since(start)))
		}
		return newIngressCaCrt != oldIngressCaCrt
	}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("Fail updating ingress %v CA", adminCluster.Name))
	return adminCluster.
		GetSecretDataAsString(constants.VerrazzanoSystemNamespace, pocnst.VerrazzanoIngressSecret, mcconstants.CaCrtKey)
}

func verifyCaSyncToManagedClusters(admCaCrt string) {
	for _, managedCluster := range managedClusters {
		verifyManagedClusterRegistration(managedCluster, admCaCrt, mcconstants.AdminCaBundleKey)
		verifyManagedClusterAdminKubeconfig(managedCluster, admCaCrt, mcconstants.KubeconfigKey)
		if isDefaultFluentd(originalFluentd) {
			verifyManagedClusterRegistration(managedCluster, admCaCrt, mcconstants.ESCaBundleKey)
		}
	}
}

func verifyManagedClusterAdminKubeconfig(managedCluster *multicluster.Cluster, admCaCrt, kubeconfigKey string) {
	start := time.Now()
	gomega.Eventually(func() bool {
		newKubeconfig := managedCluster.
			GetSecretDataAsString(constants.VerrazzanoSystemNamespace, aocnst.MCAgentSecret, kubeconfigKey)
		jsonKubeconfig, err := yaml.YAMLToJSON([]byte(newKubeconfig))
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Failed converting admin kubeconfig to JSON: %v", err))
			return false
		}
		parsedKubeconfig, err := gabs.ParseJSON(jsonKubeconfig)
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Failed parsing admin kubeconfig JSON: %v", err))
			return false
		}
		newCaCrt := parsedKubeconfig.Path("certificate-authority-data").Data().(string)
		if newCaCrt == admCaCrt {
			pkg.Log(pkg.Error, fmt.Sprintf("%v of %v took %v updated", aocnst.MCAgentSecret, managedCluster.Name, time.Since(start)))
		} else {
			pkg.Log(pkg.Error, fmt.Sprintf("%v of %v is not updated", aocnst.MCAgentSecret, managedCluster.Name))
		}
		return admCaCrt == newCaCrt
	}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("Sync CA %v", managedCluster.Name))
}

func verifyManagedClusterRegistration(managedCluster *multicluster.Cluster, admCaCrt, cakey string) {
	start := time.Now()
	gomega.Eventually(func() bool {
		newCaCrt := managedCluster.
			GetSecretDataAsString(constants.VerrazzanoSystemNamespace, aocnst.MCRegistrationSecret, cakey)
		if newCaCrt == admCaCrt {
			pkg.Log(pkg.Error, fmt.Sprintf("%v of %v took %v updated", aocnst.MCRegistrationSecret, managedCluster.Name, time.Since(start)))
		} else {
			pkg.Log(pkg.Error, fmt.Sprintf("%v of %v is not updated", aocnst.MCRegistrationSecret, managedCluster.Name))
		}
		return admCaCrt == newCaCrt
	}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("Sync CA %v", managedCluster.Name))
}

func verifyManagedFluentd(since time.Time) {
	for _, managedCluster := range managedClusters {
		gomega.Eventually(func() bool {
			logs := managedCluster.FluentdLogs(5, since)
			ok := checkFluentdLogs(logs)
			if !ok {
				pkg.Log(pkg.Error, fmt.Sprintf("%v Fluentd is not read: \n%v\n", managedCluster.Name, logs))
			}
			return ok
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("scrape target of %s is not ready", managedCluster.Name))
	}
}

func checkFluentdLogs(logs string) bool {
	return !strings.Contains(strings.ToUpper(logs), "ERROR") && !strings.Contains(logs, "Exception")
}

func verifyRegistration() {
	for _, managedCluster := range managedClusters {
		reg, _ := adminCluster.GetRegistration(managedCluster.Name)
		if reg == nil {
			adminCluster.Register(managedCluster)
			gomega.Eventually(func() bool {
				reg, err := adminCluster.GetRegistration(managedCluster.Name)
				return reg != nil && err == nil
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("%s is not registered", managedCluster.Name))
		}
	}
}

func reapplyManagedClusterRegManifest() {
	for _, managedCluster := range managedClusters {
		reg, err := adminCluster.GetManifest(managedCluster.Name)
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("could not get manifest for managed cluster %v error: %v", managedCluster.Name, err))
		}
		pkg.Log(pkg.Info, fmt.Sprintf("Reapplying registration manifest for managed cluster %s", managedCluster.Name))
		managedCluster.Apply(reg)
	}
}
