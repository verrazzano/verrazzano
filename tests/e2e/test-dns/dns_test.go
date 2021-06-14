// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dns_test

import (
	"fmt"
	"net"
	"os"
	"text/template"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const testNamespace string = "dnstest"

const resourceYaml string = `
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: dnstest
  annotations:
    external-dns.alpha.kubernetes.io/hostname: {{ .hostname }}
    external-dns.alpha.kubernetes.io/target: 192.168.0.1
spec:
  rules:
  - host: {{ .hostname }}
`

var waitTimeout = 30 * time.Minute
var pollingInterval = 30 * time.Second

var _ = ginkgo.BeforeSuite(func() {
	nsLabels := map[string]string{}
	if _, err := pkg.CreateNamespace(testNamespace, nsLabels); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}
})

var _ = ginkgo.AfterSuite(func() {
	if err := pkg.DeleteNamespace(testNamespace); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the namespace: %v", err))
	}
})

func makeYaml(index int) (string, error) {
	tmpl, err := template.New("test").Parse(resourceYaml)
	if err != nil {
		return "", err
	}
	f, err := os.Create("/tmp/resources.yaml")
	if err != nil {
		return "", err
	}
	hostname := fmt.Sprintf("x%d.zaabbcc.v8o.io", index)
	vals := map[string]string{
		"hostname": hostname,
	}
	err = tmpl.Execute(f, vals)
	f.Close()
	return hostname, err
}

func isHostnameResolvable(host string) bool {
	ips, err := net.LookupIP(host)
	if err == nil && len(ips) > 0 {
		pkg.Log(pkg.Info, fmt.Sprintf("Resolved %s to IPs: %v", host, ips))
	} else {
		pkg.Log(pkg.Info, fmt.Sprintf("Unable to resolve %s", host))
	}
	return err == nil && len(ips) > 0
}

func deployTestResources() {
	pkg.Log(pkg.Info, "Create component resource")
	if err := pkg.CreateOrUpdateResourceFromFile("/tmp/resources.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create DNS test resources: %v", err))
	}
}

func undeployTestResources() {
	pkg.Log(pkg.Info, "Delete resources")
	if err := pkg.DeleteResourceFromFile("/tmp/resources.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete resources: %v", err))
	}
}

var _ = ginkgo.Describe("Verify the things", func() {
	ginkgo.It("Do all the things", func() {
		for i := 1; i < 500; i++ {
			hostname, err := makeYaml(i)
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("Failed to make YAML file: %v", err))
			}

			deployTestResources()

			gomega.Eventually(func() bool {
				return isHostnameResolvable(hostname)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())

			undeployTestResources()
		}
	})
})
