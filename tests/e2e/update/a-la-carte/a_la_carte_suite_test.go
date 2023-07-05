// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package alacarte

import (
	"flag"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var updateType string
var clusterResourceNamespace string
var dnsType string
var certificateType string

func init() {
	flag.StringVar(&updateType, "updateType", "", "updateType is the type of update to perform")
	flag.StringVar(&clusterResourceNamespace, "clusterResourceNamespace", "my-cert-manager", "the cluster issuer namespace")
	flag.StringVar(&dnsType, "dnsType", "wildcard", "the DNS type to configure")
	flag.StringVar(&certificateType, "certificateType", "default", "the certificate issuer")
}
func TestALaCarte(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "A-La-Carte Suite")
}
