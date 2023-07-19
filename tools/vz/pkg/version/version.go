// Copyright (c) 2023, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package version

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

// ValidateCompatibleKubernetesVersion gets the bomDoc from the VPO pod
// it then compares the supported versions in the bom with the server version to determine if it is valid
func ValidateCompatibleKubernetesVersion(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	kubeClient, err := vzHelper.GetKubeClient(cmd)
	if err != nil {
		return err
	}
	config, err := vzHelper.GetConfig(cmd)
	if err != nil {
		return err
	}

	bomDoc, err := bom.GetBOMDoc(kubeClient, config)
	if err != nil {
		return err
	}
	discoveryClient, err := vzHelper.GetDiscoveryClient(cmd)
	if err != nil {
		return err
	}
	kubernetesVersion, err := discoveryClient.ServerVersion()

	return k8sutil.ValidateKubernetesVersionSupported(kubernetesVersion.String(), bomDoc.SupportedKubernetesVersions)
}
