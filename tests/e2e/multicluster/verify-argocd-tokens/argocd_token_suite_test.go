// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd_token_test

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestArgoCDTokens(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Argo CD Token Sync Testing Suite")
}
