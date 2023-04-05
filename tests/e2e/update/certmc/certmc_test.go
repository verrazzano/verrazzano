// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmc

import (
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	pocnst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
)

var (
	t               = framework.NewTestFramework("update fluentd external opensearch")
	adminCluster    *multicluster.Cluster
	managedClusters []*multicluster.Cluster
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
	verifyRegistration()
	verifyThanosStore()
})

var _ = BeforeSuite(beforeSuite)

var afterSuite = t.AfterSuiteFunc(func() {
	verifyThanosStore()
})

var _ = AfterSuite(afterSuite)

var _ = t.Describe("Update managed-cluster cert-manager", Label("f:platform-lcm.update"), func() {
	t.Describe("multicluster cert-manager verify", Label("f:platform-lcm.multicluster-verify"), func() {
		t.It("managed-cluster cert-manager custom CA", func() {
			updateCustomCA()
			verifyCaSync()
			verifyThanosStore()
		})
	})
	t.Describe("multicluster cert-manager verify", Label("f:platform-lcm.multicluster-verify"), func() {
		t.It("managed-cluster cert-manager default self-signed CA", func() {
			revertToDefaultCertManager()
			verifyCaSync()
		})
	})
})

var originalCMs = map[string]*vzapi.CertManagerComponent{}
var oldCaCrts = map[string]string{}

func updateCustomCA() {
	for _, managedCluster := range managedClusters {
		managedVZ := managedCluster.GetCR(true)
		oldCaCrt := managedCluster.
			GetSecretDataAsString(constants.VerrazzanoSystemNamespace, pocnst.VerrazzanoIngressSecret, mcconstants.CaCrtKey)
		oldCaCrts[managedCluster.Name] = oldCaCrt
		oldCM := managedVZ.Spec.Components.CertManager
		var newCM *vzapi.CertManagerComponent
		if isDefaultConfig(oldCM) {
			caname := managedCluster.GenerateCA()
			newCM = &vzapi.CertManagerComponent{
				Certificate: vzapi.Certificate{CA: vzapi.CA{SecretName: caname, ClusterResourceNamespace: constants.CertManagerNamespace}},
			}
		}
		originalCMs[managedCluster.Name] = oldCM
		m := &CertModifier{CertManager: newCM}
		update.RetryUpdate(m, managedCluster.KubeConfigPath, true, pollingInterval, waitTimeout)
	}
}

func isDefaultConfig(cm *vzapi.CertManagerComponent) bool {
	return cm == nil || reflect.DeepEqual(*cm, vzapi.CertManagerComponent{})
}

func revertToDefaultCertManager() {
	for _, managedCluster := range managedClusters {
		oldCaCrt := managedCluster.
			GetSecretDataAsString(constants.VerrazzanoSystemNamespace, pocnst.VerrazzanoIngressSecret, mcconstants.CaCrtKey)
		oldCaCrts[managedCluster.Name] = oldCaCrt
		oldCm := originalCMs[managedCluster.Name]
		m := &CertModifier{CertManager: oldCm}
		update.RetryUpdate(m, managedCluster.KubeConfigPath, true, pollingInterval, waitTimeout)
	}
}

func verifyCaSync() {
	for _, managedCluster := range managedClusters {
		oldCaCrt := oldCaCrts[managedCluster.Name]
		newCaCrt := ""
		admCaCrt := ""
		start := time.Now()
		gomega.Eventually(func() bool {
			newCaCrt = managedCluster.
				GetSecretDataAsString(constants.VerrazzanoSystemNamespace, pocnst.VerrazzanoIngressSecret, mcconstants.CaCrtKey)
			if oldCaCrt == newCaCrt {
				pkg.Log(pkg.Error, fmt.Sprintf("%v of %v is not updated", pocnst.VerrazzanoIngressSecret, managedCluster.Name))
			} else {
				pkg.Log(pkg.Error, fmt.Sprintf("%v of %v took %v updated", pocnst.VerrazzanoIngressSecret, managedCluster.Name, time.Since(start)))
			}
			return newCaCrt != oldCaCrt
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("Sync CA %v", managedCluster.Name))
		newCaCrt = managedCluster.
			GetSecretDataAsString(constants.VerrazzanoSystemNamespace, pocnst.VerrazzanoIngressSecret, mcconstants.CaCrtKey)
		casecName := fmt.Sprintf("ca-secret-%s", managedCluster.Name)
		start = time.Now()
		gomega.Eventually(func() bool {
			admCaCrt = adminCluster.GetSecretDataAsString(constants.VerrazzanoMultiClusterNamespace, casecName, "cacrt")
			if admCaCrt == newCaCrt {
				pkg.Log(pkg.Error, fmt.Sprintf("%v of %v took %v updated", casecName, managedCluster.Name, time.Since(start)))
			} else {
				pkg.Log(pkg.Error, fmt.Sprintf("%v of %v is not updated", casecName, managedCluster.Name))
			}
			return admCaCrt == newCaCrt
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("Sync CA %v", managedCluster.Name))
	}
}

func verifyThanosStore() {
	for _, managedCluster := range managedClusters {
		gomega.Eventually(func() (bool, error) {
			metricsTest, err := pkg.NewMetricsTest([]string{adminCluster.KubeConfigPath, managedCluster.KubeConfigPath}, adminCluster.KubeConfigPath, map[string]string{})
			if err != nil {
				t.Logs.Errorf("Failed to create metrics test object for cluster: %v", err)
				return false, err
			}

			queryStores, err := metricsTest.Source.GetTargets()
			if err != nil {
				t.Logs.Errorf("Failed to create get metrics target source: %v", err)
				return false, err
			}

			expectedName := fmt.Sprintf("%s:443", managedCluster.GetQueryIngress())
			for _, store := range queryStores {
				storeMap, ok := store.(map[string]interface{})
				if !ok {
					t.Logs.Infof("Thanos store empty, skipping entry")
					continue
				}
				name, ok := storeMap["name"]
				if !ok {
					t.Logs.Infof("Name not found for store, skipping entry")
					continue
				}
				nameString, nameOk := name.(string)
				if !nameOk {
					t.Logs.Infof("Name not valid format, skipping entry")
					continue
				}
				if ok {
					t.Logs.Infof("Found store in Thanos %s, want is equal to %s", nameString, expectedName)
				}
				if ok && nameString == expectedName {
					return true, nil
				}
			}
			return false, nil
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("store of %s is not ready", managedCluster.Name))
	}
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
