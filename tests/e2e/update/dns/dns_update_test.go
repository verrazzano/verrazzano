// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dns

import (
	"fmt"
	"strings"
	"time"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

type EnvironmentNameModifier struct {
	EnvironmentName string
}

type WildcardDnsModifier struct {
	Domain string
}

type Ingress struct {
	Name           string
	Namespace      string
	Hostname       string
	LoadBalancerIP string
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

var ingressList []Ingress
var t = framework.NewTestFramework("update dns")

var _ = t.Describe("Update environment name", func() {
	t.It("Verify the environment name", func() {
		cr := update.GetCR()
		getIngressList()
		validateIngressList(cr.Spec.EnvironmentName, "nip.io")
	})
})

func getIngressList() error {
	// Fetch the ingresses for the Verrazzano components
	ingresses, err := pkg.GetIngressList("")
	if err == nil {
		for _, ingress := range ingresses.Items {
			ingressList = append(ingressList, Ingress{ingress.Name, ingress.Namespace, ingress.Spec.Rules[0].Host, ingress.Status.LoadBalancer.Ingress[0].IP})
		}
	}
	return nil
}

func validateIngressList(environmentName string, domain string) error {
	// Validate that the ingresses contain the expected environment name and domain name
	for _, ingress := range ingressList {
		str := fmt.Sprintf("%s.%s.%s", environmentName, ingress.LoadBalancerIP, domain)
		if !strings.Contains(ingress.Hostname, str) {
			return fmt.Errorf("Ingress %s in namespace %s  with hostname %s must contain %s", ingress.Name, ingress.Namespace, ingress.Hostname, str)
		}
	}
	return nil
}
