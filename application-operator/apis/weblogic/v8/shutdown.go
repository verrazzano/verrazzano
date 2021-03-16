// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// Shutdown describes parameters for performing a shutdown of a domain
type Shutdown struct {
	// IgnoreSessions for graceful shutdown only, indicates to ignore pending HTTP sessions during in-flight work handling.
	// Not required. Defaults to false.
	IgnoreSessions bool `json:"ignoreSessions,omitempty"`

	// ShutdownType tells the operator how to shutdown server instances. Not required.
	// Defaults to graceful shutdown.
	ShutdownType string `json:"shutdownType,omitempty"`

	// TimeoutSeconds for graceful shutdown only, number of seconds to wait before aborting in-flight work and shutting down
	// the server. Not required. Defaults to 30 seconds.
	TimeoutSeconds int64 `json:"timeoutSeconds,omitempty"`
}
