// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster_sync_test

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestClusterSync(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Default Rancher Cluster Sync Suite")
}
