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
	OciUserId      = "OCI_USER_ID"
	OciFingerprint = "OCI_CREDENTIALS_FINGERPRINT"
	OciRegion      = "OCI_REGION"
	OciCompartment = "OCI_COMPARTMENT_ID"
	OciTenancyId   = "OCI_TENANCY_ID"
	PubKey         = "CAPI_NODE_SSH_KEY_PATH"
	ApiKey         = "CAPI_OCI_PRIVATE_KEY_PATH"
	ClusterId      = "CLUSTER_ID"
	OkeVersion     = "OKE_VERSION"
	OcneVersion    = "OCNE_VERSION"
	OcneImageId    = "OCNE_IMAGE_ID"
	OkeImageId     = "OKE_IMAGE_ID"
	OkeCniType     = "CNI_TYPE"

	B64Key         = "B64_KEY"
	B64Fingerprint = "B64_FINGERPRINT"
	B64Region      = "B64_REGION"
	B64Tenancy     = "B64_TENANCY"
	B64User        = "B64_USER"
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
		OciRegion,
		OciCompartment,
		OciFingerprint,
		OciTenancyId,
		OciUserId,
	}
	for _, key := range keys {
		if err := i.addEnvValue(key); err != nil {
			return err
		}
	}

	i.b64EncodeKV(OciRegion, B64Region)
	i.b64EncodeKV(OciFingerprint, B64Fingerprint)
	i.b64EncodeKV(OciTenancyId, B64Tenancy)
	i.b64EncodeKV(OciUserId, B64User)
	return nil
}

func (i input) newOCIClient() (OCIClient, error) {
	tenancy := i[OciTenancyId].(string)
	user := i[OciUserId].(string)
	key := i[ApiKey].(string)
	fingerprint := i[OciFingerprint].(string)
	region := i[OciRegion].(string)
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
	compartmentID := i[OciCompartment].(string)
	switch clusterType {
	case OKE:
		// add the latest OKE version
		version, err := oci.GetLatestOKEVersion(client.ContainerEngineClient, compartmentID)
		if err != nil {
			return err
		}
		i[OkeVersion] = version
		imageID, err := oci.GetOKENodeImageForVersion(client.ContainerEngineClient, compartmentID, version)
		if err != nil {
			return err
		}
		i[OkeImageId] = imageID
		if err := i.addEnvValue(OkeCniType); err != nil {
			return err
		}
	case OCNEOCI:
		imageID, err := oci.GetOL8ImageID(client.ComputeClient, compartmentID)
		if err != nil {
			return err
		}
		i[OcneImageId] = imageID
	}

	return nil
}
