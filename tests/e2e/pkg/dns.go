// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"reflect"
	"strings"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const (
	NipDomain   = "nip.io"
	SslipDomain = "sslip.io"
)

// Returns well-known wildcard DNS name is used
func GetWildcardDNS(s string) string {
	wildcards := []string{NipDomain, SslipDomain}
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

func IsDefaultDNS(dns *vzapi.DNSComponent) bool {
	return dns == nil ||
		reflect.DeepEqual(*dns, vzapi.DNSComponent{}) ||
		reflect.DeepEqual(*dns, vzapi.DNSComponent{Wildcard: &vzapi.Wildcard{Domain: NipDomain}})
}
