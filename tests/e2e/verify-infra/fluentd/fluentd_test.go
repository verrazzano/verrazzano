// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd_test

import (
	"os"
	"strconv"
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
	// false unless vzCR.Spec.EnvironmentName == "admin"
	isAdmin bool
	// false unless env var EXTERNAL_ELASTICSEARCH is set to true
	useExternalElasticsearch bool
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 5 * time.Second
)

var _ = BeforeSuite(func() {
	var err error

	useExternalElasticsearch = false
	b, err := strconv.ParseBool(os.Getenv("EXTERNAL_ELASTICSEARCH"))
	if err == nil {
		useExternalElasticsearch = b
	}

	Eventually(func() (*v1alpha1.Verrazzano, error) {
		vzCR, err = verrazzanoCR()
		return vzCR, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	isAdmin = false
	if vzCR.Spec.EnvironmentName == "admin" {
		isAdmin = true
	}

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

	if isAdmin {
		if useExternalElasticsearch {
			It("Fluentd should point to external ES", func() {
				pkg.AssertFluentdURLAndSecret(fluentdDaemonset, "https://external-es.default.172.18.0.232.nip.io", "external-es-secret")
			})
		} else {
			It("Fluentd should point to VMI ES", func() {
				pkg.AssertFluentdURLAndSecret(fluentdDaemonset, pkg.VmiESURL, pkg.VmiESSecret)
			})
		}
	} else {
		It("Fluentd should point to VMI ES", func() {
			pkg.AssertFluentdURLAndSecret(fluentdDaemonset, pkg.VmiESURL, pkg.VmiESSecret)
		})
	}
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
