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
	OciUserID      = "OCI_USER_ID"
	OciFingerprint = "OCI_CREDENTIALS_FINGERPRINT"
	OciRegion      = "CAPI_CLUSTER_REGION"
	OciCompartment = "OCI_COMPARTMENT_ID"
	OciTenancyID   = "OCI_TENANCY_ID"
	PubKey         = "CAPI_NODE_SSH_KEY_PATH"
	APIKey         = "CAPI_OCI_PRIVATE_KEY_PATH" //nolint:gosec //#gosec G101
	ClusterID      = "CLUSTER_ID"
	OkeVersion     = "OKE_VERSION"
	OcneVersion    = "OCNE_VERSION"
	OcneImageID    = "OCNE_IMAGE_ID"
	OkeImageID     = "OKE_IMAGE_ID"
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
	return qc.ClusterType == Ocneoci || qc.ClusterType == Oke
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
		OciTenancyID,
		OciUserID,
	}
	for _, key := range keys {
		if err := i.addEnvValue(key); err != nil {
			return err
		}
	}

	i.b64EncodeKV(OciRegion, B64Region)
	i.b64EncodeKV(OciFingerprint, B64Fingerprint)
	i.b64EncodeKV(OciTenancyID, B64Tenancy)
	i.b64EncodeKV(OciUserID, B64User)
	return nil
}

func (i input) newOCIClient() (OCIClient, error) {
	tenancy := i[OciTenancyID].(string)
	user := i[OciUserID].(string)
	key := i[APIKey].(string)
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
	case Oke:
		// add the latest Oke version
		version, err := oci.GetLatestOKEVersion(client.ContainerEngineClient, compartmentID)
		if err != nil {
			return err
		}
		i[OkeVersion] = version
		imageID, err := oci.GetOKENodeImageForVersion(client.ContainerEngineClient, compartmentID, version)
		if err != nil {
			return err
		}
		i[OkeImageID] = imageID
		if err := i.addEnvValue(OkeCniType); err != nil {
			return err
		}
	case Ocneoci:
		imageID, err := oci.GetOL8ImageID(client.ComputeClient, compartmentID)
		if err != nil {
			return err
		}
		i[OcneImageID] = imageID
	}

	return nil
}
