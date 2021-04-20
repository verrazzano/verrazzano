// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import "strings"

// Returns well-known wildcard suffix or empty string
func getDNSWildcard(s string) string {
	wildcards := []string {"xip.io", "nip.io", "sslip.io"}
	// get address segment (ignore port)
	segs := strings.Split(s, ":")
	for _, w := range wildcards {
		if strings.HasSuffix(segs[0], w) {
			return w
		}
	}
	return ""
}

// Returns true if string has DNS wildcard name
func hasDNSWildcard(s string) bool {
	return getDNSWildcard(s) != ""
}


