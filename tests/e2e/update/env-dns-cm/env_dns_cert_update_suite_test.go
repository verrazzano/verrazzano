// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package envdnscm

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
)

func TestEnvironmentNameDNSCertManagerUpdate(test *testing.T) {
	t.RegisterFailHandler()
	RunSpecs(test, "Update env-dns-cm Suite")
}
