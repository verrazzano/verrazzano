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
apiVersion: v1
kind: Service
metadata:
  name: nginx
  namespace: dnstest
  annotations:
    external-dns.alpha.kubernetes.io/hostname: {{ .hostname }}
spec:
  type: NodePort
  ports:
  - port: 80
    name: http
    nodePort: 30080
  selector:
    app: nginx
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: dnstest
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - image: nginx
        name: nginx
        ports:
        - containerPort: 80
          name: http
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
	var hostname string
	ginkgo.It("Make YAML.", func() {
		var err error
		hostname, err = makeYaml(1)
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to make YAML file: %v", err))
		}
	})

	ginkgo.It("Deploy resources.", func() {
		deployTestResources()
	})

	ginkgo.It("Waiting for DNS to resolve", func() {
		gomega.Eventually(func() bool {
			return isHostnameResolvable(hostname)
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
	})

	ginkgo.It("Undeploy resources.", func() {
		undeployTestResources()
	})
})
