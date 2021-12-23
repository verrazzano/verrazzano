// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo

import (
	"flag"
	"github.com/onsi/gomega"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var skipDeploy bool
var skipUndeploy bool

func init() {
	flag.BoolVar(&skipDeploy, "skipDeploy", false, "skipDeploy skips the call to install the application")
	flag.BoolVar(&skipUndeploy, "skipUndeploy", false, "skipUndeploy skips the call to install the application")
}

// TestToDoListExample tests the ToDoList example
func TestToDoListExample(t *testing.T) {
	gomega.RegisterFailHandler(FailHandler)
	ginkgo.RunSpecs(t, "ToDo List Example Test Suite")
}
