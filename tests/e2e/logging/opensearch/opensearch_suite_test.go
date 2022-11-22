// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

// TestOpenSearchLogging tests the logging to Verrazzano indices in OpenSearch.
func TestOpenSearchLogging(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "OpenSearch Logging Suite")
}
