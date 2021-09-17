// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd_test

import (
	"os"
	"time"

	appv1 "k8s.io/api/apps/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const verrazzanoNamespace string = "verrazzano-system"

var (
	vzCR             *v1alpha1.Verrazzano
	fluentdDaemonset *appv1.DaemonSet
	waitTimeout      = 10 * time.Minute
	pollingInterval  = 5 * time.Second
)

var _ = BeforeSuite(func() {
	var err error

	Eventually(func() (*v1alpha1.Verrazzano, error) {
		vzCR, err = verrazzanoCR()
		return vzCR, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	Eventually(func() (*appv1.DaemonSet, error) {
		fluentdDaemonset, err = pkg.GetFluentdDaemonset()
		return fluentdDaemonset, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())
})

var _ = Describe("Eluentd", func() {
	It("Fluentd pod should be running", func() {
		podsRunning := func() bool {
			return pkg.PodsRunning(verrazzanoNamespace, []string{"fluentd"})
		}
		Eventually(podsRunning, waitTimeout, pollingInterval).Should(BeTrue(), "pods did not all show up")
	})

	It("managed cluster Fluentd should point to the correct ES", func() {
		useExternalElasticsearch := false
		if os.Getenv("EXTERNAL_ELASTICSEARCH") == "true" {
			useExternalElasticsearch = true
		}
		isAdmin := false
		if vzCR != nil && vzCR.Spec.EnvironmentName == "admin" {
			isAdmin = true
		}
		if isAdmin && useExternalElasticsearch {
			Eventually(func() bool {
				return pkg.AssertFluentdURLAndSecret(fluentdDaemonset, "https://external-es.default.172.18.0.232.nip.io", "external-es-secret")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected external ES in fluentd Daemonset setting")
		} else {
			Eventually(func() bool {
				return pkg.AssertFluentdURLAndSecret(fluentdDaemonset, pkg.VmiESURL, pkg.VmiESSecret)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected VMI ES in fluentd Daemonset setting")
		}
	})
})

func verrazzanoCR() (*v1alpha1.Verrazzano, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return nil, err
	}
	cr, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return cr, nil
}
