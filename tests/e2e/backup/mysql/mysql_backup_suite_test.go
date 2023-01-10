// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestMySQLOperatorBackup(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "MySQL Operator Backup Suite Using OCI Credentials")
}
