package capi

import (
	"os"
)

var (
	OCIUserId                        string
	OCIFingerprint                   string
	OCITenancyId                     string
	OCIRegion                        string
	OCICompartmentID                 string
	KubernetesVersion                string
	OCIVcnId                         string
	OCISubnetID                      string
	OCISubnetCIDR                    string
	OciSshKey                        string
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
	OCIUserId = os.Getenv("OCI_USER_ID")
	OCITenancyId = os.Getenv("OCI_TENANCY_ID")
	OCICompartmentID = os.Getenv("OCI_COMPARTMENT_ID")
	KubernetesVersion = os.Getenv("KUBERNETES_VERSION")
	OCIVcnId = os.Getenv("OCI_VCN_ID")
	OCISubnetID = os.Getenv("OCI_SUBNET_ID")
	OCISubnetCIDR = os.Getenv("OCI_SUBNET_CIDR")
	OCICredsKey = os.Getenv("OCI_CREDENTIALS_KEY")
	OciSshKey = os.Getenv("OCI_SSH_KEY")
	ClusterName = os.Getenv("CLUSTER_NAME")
}
