// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

import "time"

const (
	WaitTimeout     = 2 * time.Minute
	PollingInterval = 5 * time.Second
)

var (
	NamespaceLabels = map[string]string{
		"istio-injection":    "enabled",
		"verrazzano-managed": "true",
	}
)
