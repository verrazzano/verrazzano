// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dns

import (
	"fmt"
	"strings"
	"time"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 15 * time.Second
)

type EnvironmentNameModifier struct {
	EnvironmentName string
}

type WildcardDnsModifier struct {
	Domain string
}

func (u EnvironmentNameModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.EnvironmentName = u.EnvironmentName
}

func (u WildcardDnsModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.DNS == nil {
		cr.Spec.Components.DNS = &vzapi.DNSComponent{}
	}
	if cr.Spec.Components.DNS.Wildcard == nil {
		cr.Spec.Components.DNS.Wildcard = &vzapi.Wildcard{}
	}
	cr.Spec.Components.DNS.Wildcard.Domain = u.Domain
}

var (
	t                      = framework.NewTestFramework("update dns")
	defaultEnvironmentName string
	testEnvironmentName    string = "test-env"
)

var _ = t.Describe("Update environment name", func() {
	t.It("Verify the default environment name", func() {
		cr := update.GetCR()
		defaultEnvironmentName = cr.Spec.EnvironmentName
		validateIngressList(defaultEnvironmentName, "nip.io")
		validateGatewayList(defaultEnvironmentName, "nip.io")
	})

	t.It("Verify the updated environment name", func() {
		validateIngressList(testEnvironmentName, "nip.io")
		validateGatewayList(testEnvironmentName, "nip.io")
	})
})

func validateIngressList(environmentName string, domain string) {
	Eventually(func() bool {
		// Fetch the ingresses for the Verrazzano components
		ingressList, err := pkg.GetIngressList("")
		if err == nil {
			// Verify that the ingresses contain the expected environment name and domain name
			for _, ingress := range ingressList.Items {
				hostname, loadbalancerIP := ingress.Spec.Rules[0].Host, ingress.Status.LoadBalancer.Ingress[0].IP
				suffix := fmt.Sprintf("%s.%s.%s", environmentName, loadbalancerIP, domain)
				if !strings.Contains(hostname, suffix) {
					fmt.Println(fmt.Errorf("Ingress %s in namespace %s  with hostname %s must contain suffix %s", ingress.Name, ingress.Namespace, hostname, suffix))
					return false
				}
			}
			return true
		}
		return false
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the ingresses are valid")
}

func validateGatewayList(environmentName string, domain string) {
	Eventually(func() bool {
		// Fetch the istio load balancer IP
		var loadbalancerIP string = ""
		// Fetch the gateways for the deployed applications
		gatewayList, err := pkg.GetGatewayList("")
		if err == nil {
			// Verify that the gateways contain the expected environment name and domain name
			for _, gateway := range gatewayList.Items {
				hostname := gateway.Spec.Servers[0].Hosts[0]
				suffix := fmt.Sprintf("%s.%s.%s", environmentName, loadbalancerIP, domain)
				if !strings.Contains(hostname, suffix) {
					fmt.Println(fmt.Errorf("Ingress %s in namespace %s  with hostname %s must contain %s", gateway.Name, gateway.Namespace, hostname, suffix))
					return false
				}
			}
			return true
		}
		return false
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the application gateways are valid")
}
