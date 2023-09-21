// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package quickcreate

import (
	"github.com/oracle/oci-go-sdk/v53/common"
	"github.com/oracle/oci-go-sdk/v53/containerengine"
	"github.com/oracle/oci-go-sdk/v53/core"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/oci"
)

const (
	OCI_USER_ID     = "OCI_USER_ID"
	OCI_FINGERPRINT = "OCI_CREDENTIALS_FINGERPRINT"
	OCI_REGION      = "OCI_REGION"
	OCI_COMPARTMENT = "OCI_COMPARTMENT_ID"
	OCI_TENANCY_ID  = "OCI_TENANCY_ID"
	PUB_KEY         = "CAPI_NODE_SSH_KEY_PATH"
	API_KEY         = "CAPI_OCI_PRIVATE_KEY_PATH"
	CLUSTER_ID      = "CLUSTER_ID"
	OKE_VERSION     = "OKE_VERSION"
	OCNE_VERSION    = "OCNE_VERSION"
	OCNE_IMAGE_ID   = "OCNE_IMAGE_ID"
	OKE_IMAGE_ID    = "OKE_IMAGE_ID"
	OKE_CNI_TYPE    = "CNI_TYPE"

	B64_KEY         = "B64_KEY"
	B64_FINGERPRINT = "B64_FINGERPRINT"
	B64_REGION      = "B64_REGION"
	B64_TENANCY     = "B64_TENANCY"
	B64_USER        = "B64_USER"
)

type (
	OCIClient struct {
		containerengine.ContainerEngineClient
		core.ComputeClient
	}
)

func (qc *QCContext) isOCICluster() bool {
	return qc.ClusterType == OCNEOCI || qc.ClusterType == OKE
}

func (i input) prepareOCI(clusterType string) error {
	if err := i.addOCIEnv(); err != nil {
		return err
	}
	ociClient, err := i.newOCIClient()
	if err != nil {
		return err
	}
	if err := i.addOCIValues(ociClient, clusterType); err != nil {
		return err
	}
	return nil
}

func (i input) addOCIEnv() error {
	keys := []string{
		OCI_REGION,
		OCI_COMPARTMENT,
		OCI_FINGERPRINT,
		OCI_TENANCY_ID,
		OCI_USER_ID,
	}
	for _, key := range keys {
		if err := i.addEnvValue(key); err != nil {
			return err
		}
	}

	i.b64EncodeKV(OCI_REGION, B64_REGION)
	i.b64EncodeKV(OCI_FINGERPRINT, B64_FINGERPRINT)
	i.b64EncodeKV(OCI_TENANCY_ID, B64_TENANCY)
	i.b64EncodeKV(OCI_USER_ID, B64_USER)
	return nil
}

func (i input) newOCIClient() (OCIClient, error) {
	tenancy := i[OCI_TENANCY_ID].(string)
	user := i[OCI_USER_ID].(string)
	key := i[API_KEY].(string)
	fingerprint := i[OCI_FINGERPRINT].(string)
	region := i[OCI_REGION].(string)
	provider := common.NewRawConfigurationProvider(tenancy, user, region, fingerprint, key, nil)
	containerEngineClient, err := containerengine.NewContainerEngineClientWithConfigurationProvider(provider)
	if err != nil {
		return OCIClient{}, err
	}
	computeClient, err := core.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		return OCIClient{}, err
	}
	return OCIClient{
		ContainerEngineClient: containerEngineClient,
		ComputeClient:         computeClient,
	}, nil
}

func (i input) addOCIValues(client OCIClient, clusterType string) error {
	compartmentID := i[OCI_COMPARTMENT].(string)
	switch clusterType {
	case OKE:
		// add the latest OKE version
		version, err := oci.GetLatestOKEVersion(client.ContainerEngineClient, compartmentID)
		if err != nil {
			return err
		}
		i[OKE_VERSION] = version
		imageID, err := oci.GetOKENodeImageForVersion(client.ContainerEngineClient, compartmentID, version)
		if err != nil {
			return err
		}
		i[OKE_IMAGE_ID] = imageID
		if err := i.addEnvValue(OKE_CNI_TYPE); err != nil {
			return err
		}
	case OCNEOCI:
		imageID, err := oci.GetOL8ImageID(client.ComputeClient, compartmentID)
		if err != nil {
			return err
		}
		i[OCNE_IMAGE_ID] = imageID
	}

	return nil
}
