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
	pollingInterval = 10 * time.Second
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
	currentEnvironmentName string
	currentDNSDomain       string
	testEnvironmentName    string = "env"
	testDNSDomain          string = "nip.io"
)

var _ = t.Describe("Test updates to environment name and dns domain", func() {
	t.It("Verify the current environment name", func() {
		cr := update.GetCR()
		currentEnvironmentName = cr.Spec.EnvironmentName
		currentDNSDomain = cr.Spec.Components.DNS.Wildcard.Domain
		validateIngressList(currentEnvironmentName, currentDNSDomain)
		validateGatewayList(currentDNSDomain)
	})

	t.It("Verify the updated environment name", func() {
		m := EnvironmentNameModifier{testEnvironmentName}
		update.UpdateCR(m)
		validateIngressList(testEnvironmentName, currentDNSDomain)
		validateGatewayList(currentDNSDomain)
	})

	t.It("Verify the updated dns domain", func() {
		m := WildcardDnsModifier{testDNSDomain}
		update.UpdateCR(m)
		validateIngressList(testEnvironmentName, testDNSDomain)
		validateGatewayList(testDNSDomain)
	})
})

func validateIngressList(environmentName string, domain string) {
	Eventually(func() bool {
		// Fetch the ingresses for the Verrazzano components
		ingressList, err := pkg.GetIngressList("")
		if err == nil {
			// Verify that the ingresses contain the expected environment name and domain name
			for _, ingress := range ingressList.Items {
				hostname := ingress.Spec.Rules[0].Host
				if !strings.Contains(hostname, environmentName) {
					fmt.Println(fmt.Errorf("Ingress %s in namespace %s with hostname %s must contain %s", ingress.Name, ingress.Namespace, hostname, environmentName))
					return false
				}
				if !strings.Contains(hostname, domain) {
					fmt.Println(fmt.Errorf("Ingress %s in namespace %s with hostname %s must contain %s", ingress.Name, ingress.Namespace, hostname, domain))
					return false
				}
			}
			return true
		}
		return false
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the ingresses are valid")
}

func validateGatewayList(domain string) {
	Eventually(func() bool {
		// Fetch the istio load balancer IP
		// Fetch the gateways for the deployed applications
		gatewayList, err := pkg.GetGatewayList("")
		if err == nil {
			// Verify that the gateways contain the expected environment name and domain name
			for _, gateway := range gatewayList.Items {
				hostname := gateway.Spec.Servers[0].Hosts[0]
				if !strings.Contains(hostname, domain) {
					fmt.Println(fmt.Errorf("Gateway %s in namespace %s with hostname %s must contain %s", gateway.Name, gateway.Namespace, hostname, domain))
					return false
				}
			}
			return true
		}
		return false
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the application gateways are valid")
}
