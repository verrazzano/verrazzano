package capi

import (
	"fmt"
	"os"
)

var (
	OCIUserId         string
	OCIFingerprint    string
	OCITenancyId      string
	OCIRegion         string
	OCICompartmentID  string
	KubernetesVersion string
	OCIVcnId          string
	OCISubnetID       string
	OCISubnetCIDR     string
	OciSshKey         string
	OCICredsKey       string
)

func getEnvValues(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("Value of key '%s'is empty", key)
	}
	return value, nil
}

func ensureCAPIVarsInitialized() error {

	var err error
	OCIRegion, err = getEnvValues("OCI_REGION")
	if err != nil {
		return err
	}

	OCIFingerprint, err = getEnvValues("OCI_CREDENTIALS_FINGERPRINT")
	if err != nil {
		return err
	}
	OCIUserId, err = getEnvValues("OCI_USER_ID")
	if err != nil {
		return err
	}
	OCITenancyId, err = getEnvValues("OCI_TENANCY_ID")
	if err != nil {
		return err
	}
	OCICompartmentID, err = getEnvValues("OCI_COMPARTMENT_ID")
	if err != nil {
		return err
	}

	KubernetesVersion, err = getEnvValues("KUBERNETES_VERSION")
	if err != nil {
		return err
	}

	OCIVcnId, err = getEnvValues("OCI_VCN_ID")
	if err != nil {
		return err
	}

	OCISubnetID, err = getEnvValues("OCI_SUBNET_ID")
	if err != nil {
		return err
	}

	OCISubnetCIDR, err = getEnvValues("OCI_SUBNET_CIDR")
	if err != nil {
		return err
	}

	OCICredsKey, err = getEnvValues("OCI_CREDENTIALS_KEY")
	if err != nil {
		return err
	}

	OciSshKey, err = getEnvValues("OCI_SSH_KEY")
	if err != nil {
		return err
	}

	return nil

}
