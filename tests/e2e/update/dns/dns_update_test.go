// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dns

import (
	"fmt"
	"time"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var t = framework.NewTestFramework("update dns")

var _ = t.Describe("Update environment name", func() {
	t.It("Verify the environment name", func() {
		cr := update.GetCR()
		environmentName := GetEnvironmentName(cr)
		ValidateIngresses(environmentName)
	})
})

func getEnvironmentName(cr *vzapi.Verrazzano) string {
	if cr.Spec.EnvironmentName == "" {
		return "default"
	} else {
		return cr.Spec.EnvironmentName
	}
}

func validateIngresses() {
	// Fetch the ingresses
	list, _ := pkg.GetIngressList("")
	for _, ingress := range list.Items {
		fmt.Println(ingress.Name, ingress.Namespace)
	}
}
