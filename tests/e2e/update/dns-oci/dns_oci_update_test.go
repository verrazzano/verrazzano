// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dnsoci

import (
	"log"
	"os"
	"strings"
	"time"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
)

const (
	waitTimeout     = 15 * time.Minute
	pollingInterval = 10 * time.Second
)

type OCIPublicDNSModifier struct {
	DNSZoneName            string
	DNSZoneOCID            string
	DNSZoneCompartmentOCID string
	OCIConfigSecret        string
	DNSScope               string
}

func (u OCIPublicDNSModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.DNS = &vzapi.DNSComponent{}
	cr.Spec.Components.DNS.OCI = &vzapi.OCI{}
	cr.Spec.Components.DNS.OCI.DNSZoneName = u.DNSZoneName
	cr.Spec.Components.DNS.OCI.DNSZoneOCID = u.DNSZoneOCID
	cr.Spec.Components.DNS.OCI.DNSZoneCompartmentOCID = u.DNSZoneCompartmentOCID
	cr.Spec.Components.DNS.OCI.OCIConfigSecret = u.OCIConfigSecret
}

var (
	t                                 = framework.NewTestFramework("Update OCI DNS")
	testDNSZoneName            string = os.Getenv("OCI_DNS_ZONE_NAME")
	testDNSZoneOCID            string = os.Getenv("OCI_DNS_ZONE_OCID")
	testDNSZoneCompartmentOCID string = os.Getenv("OCI_DNS_COMPARTMENT_OCID")
	testOCIConfigSecret        string = os.Getenv("OCI_CONFIG_SECRET")
	testDNSScope               string = os.Getenv("OCI_DNS_SCOPE")

	currentEnvironmentName string
	currentDNSDomain       string
)

var _ = t.Describe("Test DNS updates", func() {
	t.It("Verify the current environment name and DNS domain", func() {
		cr := update.GetCR()
		currentEnvironmentName = cr.Spec.EnvironmentName
		currentDNSDomain = cr.Spec.Components.DNS.Wildcard.Domain
		validateIngressList(currentEnvironmentName, currentDNSDomain)
		validateVirtualServiceList(currentDNSDomain)
	})

	t.It("Update and verify DNS domain", func() {
		m := OCIPublicDNSModifier{testDNSZoneName, testDNSZoneOCID, testDNSZoneCompartmentOCID, testOCIConfigSecret, testDNSScope}
		err := update.UpdateCR(m)
		if err != nil {
			log.Fatalf("Error in updating DNS domain - %s", err)
		}
		validateIngressList(currentEnvironmentName, testDNSZoneName)
		validateVirtualServiceList(testDNSZoneName)
	})
})

func validateIngressList(environmentName string, domain string) {
	Eventually(func() bool {
		// Fetch the ingresses for the Verrazzano components
		ingressList, err := pkg.GetIngressList("")
		if err != nil {
			log.Fatalf("Error while fetching IngressList\n%s", err)
		}
		// Verify that the ingresses contain the expected environment name and domain name
		for _, ingress := range ingressList.Items {
			hostname := ingress.Spec.Rules[0].Host
			if !strings.Contains(hostname, environmentName) {
				log.Printf("Ingress %s in namespace %s with hostname %s must contain %s", ingress.Name, ingress.Namespace, hostname, environmentName)
				return false
			}
			if !strings.Contains(hostname, domain) {
				log.Printf("Ingress %s in namespace %s with hostname %s must contain %s", ingress.Name, ingress.Namespace, hostname, domain)
				return false
			}
		}
		return true
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the ingress hosts contain the expected environment and domain names")
}

func validateVirtualServiceList(domain string) {
	Eventually(func() bool {
		// Fetch the virtual services for the deployed applications
		virtualServiceList, err := pkg.GetVirtualServiceList("")
		if err != nil {
			log.Fatalf("Error while fetching GatewayList\n%s", err)
		}
		// Verify that the virtual services contain the expected environment name and domain nameƒ
		for _, virtualService := range virtualServiceList.Items {
			hostname := virtualService.Spec.Hosts[0]
			if !strings.Contains(hostname, domain) {
				log.Printf("Virtual Service %s in namespace %s with hostname %s must contain %s\n", virtualService.Name, virtualService.Namespace, hostname, domain)
				return false
			}
		}
		return true
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the application virtual service hosts contain the expected domain name")
}
