// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var region string
var vcnId string
var userId string
var tenancyId string
var fingerprint string
var privateKeyContents string
var nodePublicKeyContents string
var compartmentId string
var workerNodeSubnet string
var controlPlaneSubnet string
var loadBalancerSubnet string
var podCidr string

func init() {
	flag.StringVar(&region, "region", "", "region represents the region where the CAPI cluster will be created")
	flag.StringVar(&userId, "userId", "", "userId represents the user ID")
	flag.StringVar(&tenancyId, "tenancyId", "", "tenancyId represents the tenancy ID")
	flag.StringVar(&compartmentId, "compartmentId", "", "compartmentId represents the compartment ID")
	flag.StringVar(&vcnId, "vcnId", "", "vcnId represents the VCN ID")
	flag.StringVar(&fingerprint, "fingerprint", "", "fingerprint represents the OCI Credential config fingerprint")
	flag.StringVar(&privateKeyContents, "privateKeyContents", "", "privateKeyContents represents the OCI Credential config private key contents")
	flag.StringVar(&nodePublicKeyContents, "nodePublicKeyContents", "", "privateKeyContents represents the node public key contents")
	flag.StringVar(&workerNodeSubnet, "workerNodeSubnet", "", "workerNodeSubnet represents the worker node subnet")
	flag.StringVar(&controlPlaneSubnet, "controlPlaneSubnet", "", "controlPlaneSubnet represents the control plane node subnet")
	flag.StringVar(&loadBalancerSubnet, "loadBalancerSubnet", "", "loadBalancerSubnet represents the load balancer subnet")
	flag.StringVar(&podCidr, "podCidr", "", "podCidr represents the pod CIDR")
}

func TestOCNEClusterDriver(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "OCNE Cluster Driver Suite")
}
