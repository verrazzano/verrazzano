// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mark

import (
	"crypto/rand"
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"math/big"
	"os"
	"time"
)
const filename = ".upgrade-done"

var t = framework.NewTestFramework("mark")

var _ = t.SynchronizedBeforeSuite(func() []byte {
	// cleanup upgrade-done file from old run (hack to fake the inter-process wait)
	os.Remove(filename)

	work("SynchronizedBeforeSuite.process1 (runs only once) - I will new a kind cluster")
	framework.EnsureCluster(framework.DEFAULT_K8S_VERSION)
	return []byte{}
},
func([]byte) {
	work("SynchronizedBeforeSuite.allProcesses (runs once per process/node)")
})

var _ = t.Describe("Upgrade", ginkgo.Ordered, func() {

	t.BeforeAll(func() {
		framework.EnsureVerrazzanoInstalled(framework.DEFAULT_PRE_UPGRADE_V8O_VERSION)
		work("BeforeAll - I install the starting version of Verrazzano")
	})

	t.Context("when Verrazzano is installed", ginkgo.Ordered, func(){
		t.BeforeAll(func() {
			work("I wait for Verrazzano to be installed")
		})

		t.It("the install verification should pass", func() {
			work("verify-install")
		})

		t.It("the infrastructure verification should pass", func() {
			work("verify-infra")
		})

		t.When("and then", func() {
			t.It("upgraded", func() {
				work("I upgrade Verrazzano")

				// create a file to fake the inter-process wait
				f, _ := os.Create(filename)
				f.Close()
			})
		})
	})
})

var _ = t.Describe("After Verrazzano is upgraded", func() {
	t.BeforeEach(func() {
		fmt.Printf("%s Waiting for upgrade to complete...\n", time.Now().Format("3:04:05PM"))
		// wait for fake upgrade to complete
		done := false
		for done != true {
			_, err := os.Open(filename)
			if err == nil {
				done = true
			}
		}

		work("I test/wait for the upgrade to be marked as complete")
	})

	t.It("the BOM should be valid", func() {
		work("validate-bom")
	})

	t.It("istioctl verify-install should pass", func() {
		work("I run istioctl verify-install")
	})
})

func work(s string) {
	start := time.Now()
	n, err := rand.Int(rand.Reader, big.NewInt(3))
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Duration(n.Int64()) * time.Second)
	end := time.Now()
	fmt.Printf("worked from %s to %s (%d seconds), did: %s\n", start.Format("3:04:05PM"), end.Format("3:04:05PM"), n.Int64(), s)
}