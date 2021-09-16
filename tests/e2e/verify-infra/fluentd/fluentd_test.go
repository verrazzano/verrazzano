// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	appv1 "k8s.io/api/apps/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const verrazzanoNamespace string = "verrazzano-system"
const fluentdDaemonsetName string = "fluentd"
const externalESURL = "https://external-es.default.172.18.0.232.nip.io"
const externalESSecret = "external-es-secret"
const vmiESURL = "http://vmi-system-es-ingest-oidc:8775"
const vmiESSecret = "verrazzano"

var (
	vzCR                   *v1alpha1.Verrazzano
	fluentdDS       		*appv1.DaemonSet
	ingressURLs            map[string]string
	// false unless vzCR.Spec.EnvironmentName == "admin"
	isAdmin                bool
	// false unless env var EXTERNAL_ELASTICSEARCH is set to true
	useExternalElasticsearch bool
	waitTimeout            = 10 * time.Minute
	pollingInterval        = 5 * time.Second
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

	Eventually(func() (map[string]string, error) {
		ingressURLs, err = vzIngressURLs()
		return ingressURLs, err
	}, waitTimeout, pollingInterval).ShouldNot(BeEmpty())

	Eventually(func() (*appv1.DaemonSet, error) {
		fluentdDS, err = fluentdDaemonset()
		return fluentdDS, err
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
				assertFluentdESURLAndSecret(externalESURL, externalESSecret)
			})
		} else {
			It("Fluentd should point to VMI ES", func() {
				assertFluentdESURLAndSecret(vmiESURL, vmiESSecret)
			})
		}
	} else {
		It("Fluentd should point to VMI ES", func() {
			assertFluentdESURLAndSecret(vmiESURL, vmiESSecret)
		})
	}
})

func assertFluentdESURLAndSecret(expectedURL, expectSecret string) {
	esURLFound := ""
	containers := fluentdDS.Spec.Template.Spec.Containers
	Expect(len(containers)).To(Equal(1))
	for _, env := range containers[0].Env {
		if env.Name == "ELASTICSEARCH_URL" {
			esURLFound = env.Value
		}
	}
	Expect(esURLFound).To(Equal(expectedURL))
	esSecretFound := ""
	for _, vol := range fluentdDS.Spec.Template.Spec.Volumes {
		if vol.Name == "secret-volume" {
			esSecretFound = vol.Secret.SecretName
		}
	}
	Expect(esSecretFound).To(Equal(expectSecret))
}

func fluentdDaemonset() (*appv1.DaemonSet, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	ds, err := clientset.AppsV1().DaemonSets(verrazzanoNamespace).Get(context.TODO(), fluentdDaemonsetName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return ds, nil
}

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

func vzIngressURLs() (map[string]string, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	ingressList, err := clientset.ExtensionsV1beta1().Ingresses(verrazzanoNamespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ingressURLs := make(map[string]string)

	for _, ingress := range ingressList.Items {
		var ingressRules = ingress.Spec.Rules
		if len(ingressRules) != 1 {
			return nil, fmt.Errorf("expected ingress %s in namespace %s to have 1 ingress rule, but had %v",
				ingress.Name, ingress.Namespace, ingressRules)
		}
		ingressURLs[ingress.Name] = fmt.Sprintf("https://%s/", ingressRules[0].Host)
	}
	return ingressURLs, nil
}
