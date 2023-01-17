// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var skipDeploy bool
var namespace string
var skipVerify bool

func init() {
	flag.BoolVar(&skipDeploy, "skipDeploy", false, "skipDeploy skips the call to install the application")
	flag.StringVar(&namespace, "namespace", argoCdHelidon, "namespace is the app namespace")
	flag.BoolVar(&skipVerify, "skipVerify", false, "skipVerify skips the post deployment app validations")
}

func TestArgoCDHelidonExample(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Argo CD Hello Helidon Suite")
}
