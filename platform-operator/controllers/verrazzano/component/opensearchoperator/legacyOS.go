// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"strings"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const (
	system              = "system"
	clusterLabel        = "opster.io/opensearch-cluster"
	nodePoolLabel       = "opster.io/opensearch-nodepool"
	opensearchNodeLabel = "verrazzano.io/opensearch-nodepool"
	clusterName         = "opensearch"
)

// handleLegacyOpenSearch performs all the tasks required to upgrade from VMO OS to new operator OS
// 1. Retain Older PVs if they exist
// 2. Delete VMO OS/OSD
// 3. Delete master node's PVCs. Since they are created by the STS they are not cleaned up by VMO
// 4. Create new PVCs for operator
// 5. Wait for new PVCs to bind to PVs
func handleLegacyOpenSearch(ctx spi.ComponentContext) error {

	ctx.Log().Once("Performing migration steps required for legacy OpenSearch")

	nodes := ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes
	if err := setPVsToRetain(ctx, nodes); err != nil {
		return err
	}

	// Remove legacy OS and OSD
	if vmiExists(ctx) {
		if err := common.CreateOrUpdateVMI(ctx, updateFuncForUninstall); err != nil {
			return fmt.Errorf("failed to disable legacy OS and OSD: %v", err)
		}
	}

	if err := deleteMasterNodePVC(ctx); err != nil {
		return fmt.Errorf("failed to delete existing master node pvc: %v", err)
	}

	if !arePVsReleased(ctx, nodes) {
		return ctrlerrors.RetryableError{Source: ComponentName, Cause: fmt.Errorf("waiting for existing PVs to be released")}
	}

	if err := createNewPVCs(ctx, nodes); err != nil {
		return fmt.Errorf("failed creating new pvc: %v", err)
	}

	if !arePVCsAndPVsBound(ctx) {
		return ctrlerrors.RetryableError{Source: ComponentName, Cause: fmt.Errorf("waiting for PVCs to bind to PVs")}
	}

	return nil
}

// vmiExists returns true if the VMI kind exists
func vmiExists(ctx spi.ComponentContext) bool {
	vmiList := vmov1.VerrazzanoMonitoringInstanceList{}
	err := ctx.Client().List(context.TODO(), &vmiList)

	if ok := meta.IsNoMatchError(err); ok {
		ctx.Log().Debugf("VerrazzanoMonitoring kind does not exist, skipping disabling legacy OS and OSD")
		return false
	}
	return true
}

// updateFuncForUninstall updates the VMI to disable VMO based OS and OSD
func updateFuncForUninstall(ctx spi.ComponentContext, storage *common.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance, existingVMI *vmov1.VerrazzanoMonitoringInstance) error {
	vmi.Spec.Opensearch = vmov1.Opensearch{Enabled: false}
	vmi.Spec.OpensearchDashboards = vmov1.OpensearchDashboards{Enabled: false}
	return nil
}

// getNodeNameFromClaimName returns the corresponding node name for a pvc name
func getNodeNameFromClaimName(claimName string, nodes []vzapi.OpenSearchNode) string {
	claimName = strings.TrimPrefix(claimName, "elasticsearch-master-")
	claimName = strings.TrimPrefix(claimName, "vmi-system-")

	// After trimming the above prefix, the pvc name can be as
	// 1. es-data
	// 2. es-data-1
	// 3. es-data-tqxkq   (In-case the PVC was resized, random suffix is appended at the end)
	// 4. es-data-1-8m66v

	// First removing the single suffix
	// For case 2, 3
	lastIndex := strings.LastIndex(claimName, "-")
	if lastIndex != -1 {
		for _, node := range nodes {
			if claimName[:lastIndex] == node.Name {
				return node.Name
			}
		}
		// Remove 2 suffix
		// For case 4
		secondLastIndex := strings.LastIndex(claimName[:lastIndex], "-")
		if secondLastIndex != -1 {
			for _, node := range nodes {
				if claimName[:secondLastIndex] == node.Name {
					return node.Name
				}
			}
		}
	}

	// Compare without removing suffix
	// For case 1
	for _, node := range nodes {
		if claimName == node.Name {
			return node.Name
		}
	}
	return ""
}
