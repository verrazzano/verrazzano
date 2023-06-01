// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var region string
var vcnID string
var userID string
var tenancyID string
var fingerprint string
var privateKeyPath string
var nodePublicKeyPath string
var compartmentID string
var workerNodeSubnet string
var controlPlaneSubnet string
var loadBalancerSubnet string

func init() {
	flag.StringVar(&region, "region", "", "region represents the region where the CAPI cluster will be created")
	flag.StringVar(&userID, "userID", "", "userID represents the user ID")
	flag.StringVar(&tenancyID, "tenancyID", "", "tenancyID represents the tenancy ID")
	flag.StringVar(&compartmentID, "compartmentID", "", "compartmentID represents the compartment ID")
	flag.StringVar(&vcnID, "vcnID", "", "vcnID represents the VCN ID")
	flag.StringVar(&fingerprint, "fingerprint", "", "fingerprint represents the OCI Credential config fingerprint")
	flag.StringVar(&privateKeyPath, "privateKeyPath", "", "privateKeyPath represents the OCI Credential config private key file path")
	flag.StringVar(&nodePublicKeyPath, "nodePublicKeyPath", "", "privateKeyPath represents the node public key file path")
	flag.StringVar(&workerNodeSubnet, "workerNodeSubnet", "", "workerNodeSubnet represents the worker node subnet")
	flag.StringVar(&controlPlaneSubnet, "controlPlaneSubnet", "", "controlPlaneSubnet represents the control plane node subnet")
	flag.StringVar(&loadBalancerSubnet, "loadBalancerSubnet", "", "loadBalancerSubnet represents the load balancer subnet")
}

func TestOCNEClusterDriver(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "OCNE Cluster Driver Suite")
}
