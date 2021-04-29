// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import "strings"

// Returns well-known wildcard DNS name is used
func GetWildcardDNS(s string) string {
	wildcards := []string{"xip.io", "nip.io", "sslip.io"}
	for _, w := range wildcards {
		if strings.Contains(s, w) {
			return w
		}
	}
	return ""
}

// Returns true if string has DNS wildcard name
func HasWildcardDNS(s string) bool {
	return GetWildcardDNS(s) != ""
}
