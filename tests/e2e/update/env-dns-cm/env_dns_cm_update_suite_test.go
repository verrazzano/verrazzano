// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package envdnscm

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEnvironmentNameDNSCertManagerUpdate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Update env-dns-cm Suite")
}
