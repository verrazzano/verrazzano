// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"os"
)

var (
	OCIUserID                        string
	OCIFingerprint                   string
	OCITenancyID                     string
	OCIRegion                        string
	OCICompartmentID                 string
	KubernetesVersion                string
	OCIVcnID                         string
	OCISubnetID                      string
	OCISubnetCIDR                    string
	OciSSHKey                        string
	OCICredsKey                      string
	ClusterName                      string
	ClusterTemplateGeneratedFilePath string
)

type TemplateInput struct {
	APIVersion                 string
	APIRepository              string
	APITag                     string
	OCIVersion                 string
	OCIRepository              string
	OCITag                     string
	OCNEBootstrapVersion       string
	OCNEBootstrapRepository    string
	OCNEBootstrapTag           string
	OCNEControlPlaneVersion    string
	OCNEControlPlaneRepository string
	OCNEControlPlaneTag        string
}

func ensureCAPIVarsInitialized() {
	OCIRegion = os.Getenv("OCI_REGION")
	OCIFingerprint = os.Getenv("OCI_CREDENTIALS_FINGERPRINT")
	OCIUserID = os.Getenv("OCI_USER_ID")
	OCITenancyID = os.Getenv("OCI_TENANCY_ID")
	OCICompartmentID = os.Getenv("OCI_COMPARTMENT_ID")
	KubernetesVersion = os.Getenv("KUBERNETES_VERSION")
	OCIVcnID = os.Getenv("OCI_VCN_ID")
	OCISubnetID = os.Getenv("OCI_SUBNET_ID")
	OCISubnetCIDR = "10.0.0.32/27"
	ClusterName = os.Getenv("CLUSTER_NAME")
	OCICredsKey = os.Getenv("TF_VAR_api_private_key_path")
	OciSSHKey = os.Getenv("TF_VAR_ssh_public_key_path")
}
