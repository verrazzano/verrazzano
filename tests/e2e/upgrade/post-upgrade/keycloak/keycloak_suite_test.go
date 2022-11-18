// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"github.com/onsi/ginkgo/v2"

	"testing"
)

func TestKeycloakPostUpgrade(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Post Upgrade Keycloak Suite")
}
