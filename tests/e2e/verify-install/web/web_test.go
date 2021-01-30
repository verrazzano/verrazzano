// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package web_test

import (
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var serverUrl = "https://verrazzano." + pkg.EnvName + "." + pkg.DnsZone

var _ = ginkgo.Describe("Verrazzano Web UI",
	func() {
		pkg.Log(pkg.Info, "The Web UI's URL is "+serverUrl)

		ginkgo.It("can be accessed", func() {
			rc, content := pkg.GetWebPageWithCABundle(serverUrl, "")
			gomega.Expect(rc).To(gomega.Equal(200))
			gomega.Expect(content).To(gomega.Not(gomega.BeEmpty()))
			gomega.Expect(content).To(gomega.Not(gomega.ContainSubstring("404")))
		})

		ginkgo.It("has the correct SSL certificate",
			func() {
				certs, err := pkg.GetCertificates(serverUrl)
				gomega.Expect(err).To(gomega.BeNil())
				// There will normally be several certs, but we only need to check the
				// first one -- might want to refactor the checks out into a pkg.IsCertValid()
				// function so we can use it from other test suites too??
				pkg.Log(pkg.Debug, "Issuer Common Name: "+certs[0].Issuer.CommonName)
				pkg.Log(pkg.Debug, "Subject Common Name: "+certs[0].Subject.CommonName)
				pkg.Log(pkg.Debug, "Not Before: "+certs[0].NotBefore.String())
				pkg.Log(pkg.Debug, "Not After: "+certs[0].NotAfter.String())
				gomega.Expect(time.Now().After(certs[0].NotBefore)).To(gomega.BeTrue())
				gomega.Expect(time.Now().Before(certs[0].NotAfter)).To(gomega.BeTrue())
			})
	})
