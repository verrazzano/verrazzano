// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/client-go/kubernetes"
)

// config for HA monitoring test suite
type config struct {
	api        *pkg.APIEndpoint
	httpClient *retryablehttp.Client
	clientset  *kubernetes.Clientset
	hosts      struct {
		rancher string
		kiali   string
	}
	users struct {
		verrazzano *pkg.UsernamePassword
	}
}

var web = config{}

var _ = t.BeforeSuite(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())
	web.api = pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
	web.httpClient = pkg.EventuallyVerrazzanoRetryableHTTPClient()
	web.clientset = k8sutil.GetKubernetesClientsetOrDie()
	web.users.verrazzano = pkg.EventuallyGetSystemVMICredentials()
	web.hosts.rancher = pkg.EventuallyGetURLForIngress(t.Logs, web.api, "cattle-system", "rancher", "https")
	web.hosts.kiali = pkg.EventuallyGetKialiHost(web.clientset)
})
