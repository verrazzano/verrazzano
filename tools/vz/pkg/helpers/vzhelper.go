// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"io"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VZHelper interface {
	GetOutputStream() io.Writer
	GetClient() (client.Client, error)
}
