// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installuninstall

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"k8s.io/client-go/dynamic"
	"time"
)

const (
	waitTimeout     = 2 * time.Minute
	pollingInterval = 5 * time.Second
)

var (
	clientset dynamic.Interface
)

var t = framework.NewTestFramework("uninstall verify Rancher CRs")

// This test performs a Rancher loop test
var _ = t.Describe("Rancher install-uninstall loop tests", Label("f:platform-lcm.rancher"), func() {

	for i := 1; i < 5; i++ {
		t.It(fmt.Sprintf("Installing Rancher: loop %s/n", i+1), func() {
		})
		t.It(fmt.Sprintf("Waiting for Install to complete"), func() {
		})
	}

})
