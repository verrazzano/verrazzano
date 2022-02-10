// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package e2e

import (
	"k8s.io/apimachinery/pkg/util/rand"
	"os"
	"testing"
	"time"

	//import tests
	_ "github.com/verrazzano/verrazzano/tests/e2e/examples/add"
	_ "github.com/verrazzano/verrazzano/tests/e2e/poc"
)

func TestMain(m *testing.M) {

	// define framework.TestContext.RepoRoot then uncomment below
	//testfiles.AddFileSource(testfiles.RootFileSource{Root: framework.TestContext.RepoRoot})
	rand.Seed(time.Now().UnixNano())
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
