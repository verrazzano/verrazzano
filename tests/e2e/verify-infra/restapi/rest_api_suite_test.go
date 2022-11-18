// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

var t = framework.NewTestFramework("restapi_test")

func TestRestApi(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "REST API Suite")
}
