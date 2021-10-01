// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package sock_shop

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	longPollingInterval  = 20 * time.Second
	longWaitTimeout      = 10 * time.Minute
	pollingInterval      = 5 * time.Second
	waitTimeout          = 5 * time.Minute
	consistentlyDuration = 1 * time.Minute
	sourceDir            = "sock-shop"
	testNamespace        = "sock-shop"
	testProjectName      = "sock-shop"
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false

var _ = AfterEach(func() {
	// set failed to true if any of the tests has failed
	failed = failed || CurrentGinkgoTestDescription().Failed
})
