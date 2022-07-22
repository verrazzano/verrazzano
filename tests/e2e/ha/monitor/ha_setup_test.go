// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var (
	api            *pkg.APIEndpoint
	vzHTTPClient   *retryablehttp.Client
	vmiCredentials *pkg.UsernamePassword
	rancherURL     string
	kialiHost      string
)

var _ = t.BeforeSuite(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())
	api = pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
	vzHTTPClient = pkg.EventuallyVerrazzanoRetryableHTTPClient()
	vmiCredentials = pkg.EventuallyGetSystemVMICredentials()
	rancherURL = pkg.EventuallyGetRancherURL(t.Logs, api)
	kialiHost = pkg.EventuallyGetKialiHost(clientset)
})
