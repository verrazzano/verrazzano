// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"context"
	"fmt"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	c "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	system                   = "system"
	clusterLabel             = "opster.io/opensearch-cluster"
	nodePoolLabel            = "opster.io/opensearch-nodepool"
	verrazzanoOSClusterLabel = "verrazzano.io/opensearch-cluster"
	clusterName              = "opensearch"
)

// handleLegacyOpenSearch performs all the tasks required to upgrade from VMO OS to new operator OS
// 1. Retain Older PVs if they exist
// 2. Delete VMO OS/OSD
// 3. Delete master node's PVCs. Since they are created by the STS they are not deleted by VMO
// 4. Create new PVCs for operator
// 5. Wait for new PVCs to bind to PVS
func handleLegacyOpenSearch(ctx spi.ComponentContext) error {

	// Retain PVs
	if err := setPVsToRetain(ctx); err != nil {
		return fmt.Errorf("failed to set PVs to retain")
	}

	// Remove legacy OS and OSD
	if err := common.CreateOrUpdateVMI(ctx, updateFuncForUninstall); err != nil {
		return fmt.Errorf("failed to disable legacy OS and OSD: %v", err)
	}

	if err := deleteMasterNodePVC(ctx); err != nil {
		return fmt.Errorf("failed to delete existing master node pvc: %v", err)
	}

	if !arePVsReleased(ctx) {
		return ctrlerrors.RetryableError{Source: ComponentName, Cause: fmt.Errorf("waiting for exisitng PVs to be released")}
	}

	if err := createNewPVCs(ctx); err != nil {
		return fmt.Errorf("falied creating new pvc: %v", err)
	}

	if !arePVCsBound(ctx) {
		return ctrlerrors.RetryableError{Source: ComponentName, Cause: fmt.Errorf("waiting for PVCs to bind to PVs")}
	}

	if err := resetReclaimPolicy(ctx); err != nil {
		return err
	}

	return nil
}

func arePVsReleased(ctx spi.ComponentContext) bool {
	pvList, err := getOSPersistentVolumes(ctx)
	if err != nil {
		return false
	}

	for _, pv := range pvList {
		if pv.Status.Phase == v1.VolumeBound {
			ctx.Log().Progressf("waiting for pv %s to be released", pv.Name)
			return false
		}
	}
	return true
}

func resetReclaimPolicy(ctx spi.ComponentContext) error {
	nodes := ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes
	for _, node := range nodes {
		if err := common.ResetVolumeReclaimPolicy(ctx, getStorageLabel(node.Name)); err != nil {
			return fmt.Errorf("failed to reset reclaim policy for pv of node %s: %v", node.Name, err)
		}
	}
	return nil
}

func arePVCsBound(ctx spi.ComponentContext) bool {
	pvcList := &v1.PersistentVolumeClaimList{}
	if err := ctx.Client().List(context.TODO(), pvcList, c.MatchingLabels{clusterLabel: clusterName}); err != nil {
		if errors.IsNotFound(err) {
			return true
		}
		return false
	}
	for _, pvc := range pvcList.Items {
		if pvc.Status.Phase != v1.ClaimBound {
			ctx.Log().Progressf("Waiting for pvc %s to bind to pv", pvc.Name)
			return false
		}
	}
	pvList := &v1.PersistentVolumeList{}
	if err := ctx.Client().List(context.TODO(), pvList, c.MatchingLabels{verrazzanoOSClusterLabel: clusterName}); err != nil {
		if errors.IsNotFound(err) {
			return true
		}
		return false
	}
	for _, pv := range pvList.Items {
		if pv.Status.Phase != v1.VolumeBound {
			ctx.Log().Progressf("Waiting for pv %s to bind to pvc", pv.Name)
		}
	}
	return true
}

func setPVsToRetain(ctx spi.ComponentContext) error {
	pvList, err := getOSPersistentVolumes(ctx)
	if err != nil {
		return err
	}
	if len(pvList) <= 0 {
		ctx.Log().Once("PVs already set to Retain or no old PVs found to retain")
		return nil
	}
	nodes := ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes
	for i := range pvList {
		pv := pvList[i]
		oldReclaimPolicy := pv.Spec.PersistentVolumeReclaimPolicy
		pv.Spec.PersistentVolumeReclaimPolicy = v1.PersistentVolumeReclaimRetain

		if pv.Labels == nil {
			pv.Labels = make(map[string]string)
		}
		pv.Labels[constants.StorageForLabel] = getStorageLabel(getNodeNameFromClaimName(pv.Spec.ClaimRef.Name, nodes))
		pv.Labels[verrazzanoOSClusterLabel] = clusterName

		// If old reclaim policy is not already set, set it
		_, ok := pv.Labels[constants.OldReclaimPolicyLabel]
		if !ok {
			pv.Labels[constants.OldReclaimPolicyLabel] = string(oldReclaimPolicy)
		}
		if err := ctx.Client().Update(context.TODO(), &pv); err != nil {
			return ctx.Log().ErrorfNewErr("Failed to retain PV %s: %v", pv.Name, err)
		}
	}
	return nil
}

func createNewPVCs(ctx spi.ComponentContext) error {
	nodes := ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes

	for _, node := range nodes {
		nodePool := getStorageLabel(node.Name)
		pvList, err := common.GetPersistentVolumes(ctx, nodePool)
		if err != nil {
			return err
		}
		if len(pvList.Items) > 0 {
			replicaCount := 0
			for i := range pvList.Items {
				pv := pvList.Items[i]

				pv.Spec.ClaimRef = nil
				if err := ctx.Client().Update(context.TODO(), &pv); err != nil {
					return err
				}

				newPVCName := fmt.Sprintf("%s-%d", nodePool, replicaCount)
				err := createPVCFromPV(ctx, pv, types.NamespacedName{Namespace: constants.VerrazzanoLoggingNamespace, Name: newPVCName}, nodePool)
				if err != nil {
					return err
				}
				replicaCount++
			}
		} else {
			ctx.Log().Oncef("No existing PVs for node pool %s", node.Name)
		}
	}

	return nil
}

//func getNewClaimName(oldClaimName string) (string, string) {
//	// Existing claim names have pattern as
//	// 1. elasticsearch-master-vmi-system-es-master-2
//	// 2. vmi-system-es-data-2
//	// 3. vmi-system-es-data
//	// 4. vmi-system-es-data-tqxkq (randomly generated suffix)
//	// 5. vmi-system-es-data-2-tqxkq
//	// Need to extract the nodePool name i.e. es-data, es-master etc.
//	// Need to append 0 for names not ending with number
//
//	// First trim to remove prefix elasticsearch-master for master nodes
//	// Second trim to remove prefix vmi-system
//	oldClaimName = strings.TrimPrefix(oldClaimName, "elasticsearch-master-")
//	oldClaimName = strings.TrimPrefix(oldClaimName, "vmi-system-")
//
//	regex := regexp.MustCompile(`^([^0-9]*?)(?:-([0-9]+))?$`)
//	matches := regex.FindStringSubmatch(oldClaimName)
//
//	var nodePool, cardinality string
//	if len(matches) > 2 {
//		nodePool = matches[1]
//		cardinality = matches[2]
//	}
//
//	if cardinality == "" {
//		cardinality = "0"
//	}
//
//	newPVCName := fmt.Sprintf("data-%s-%s-%s", clusterName, nodePool, cardinality)
//
//	return nodePool, newPVCName
//}

// createPVCFromPV creates a PVC from a PV definition, and sets the PVC to reference the PV by name
func createPVCFromPV(ctx spi.ComponentContext, volume v1.PersistentVolume, newClaimName types.NamespacedName, nodePool string) error {
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newClaimName.Name,
			Namespace: newClaimName.Namespace,
		},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), pvc, func() error {
		accessModes := make([]v1.PersistentVolumeAccessMode, len(volume.Spec.AccessModes))
		copy(accessModes, volume.Spec.AccessModes)
		pvc.Spec.AccessModes = accessModes
		pvc.Spec.Resources = v1.ResourceRequirements{
			Requests: map[v1.ResourceName]resource.Quantity{
				v1.ResourceStorage: volume.Spec.Capacity.Storage().DeepCopy(),
			},
		}
		labels := make(map[string]string)
		labels[clusterLabel] = clusterName
		labels[nodePoolLabel] = nodePool

		pvc.Labels = labels
		pvc.Spec.VolumeName = volume.Name
		return nil
	})
	return err
}

func deleteMasterNodePVC(ctx spi.ComponentContext) error {
	pvcList := &v1.PersistentVolumeClaimList{}
	if err := ctx.Client().List(context.TODO(), pvcList); err != nil {
		return ctx.Log().ErrorfNewErr("Failed listing persistent volume claims: %v", err)
	}

	for i := range pvcList.Items {
		pvc := pvcList.Items[i]
		if strings.Contains(pvc.Name, "elasticsearch-master") {
			if len(pvc.OwnerReferences) > 0 && pvc.OwnerReferences[0].Name == system && pvc.OwnerReferences[0].Kind == "VerrazzanoMonitoringInstance" {
				err := ctx.Client().Delete(context.TODO(), &pvc)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func updateFuncForUninstall(ctx spi.ComponentContext, storage *common.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance, existingVMI *vmov1.VerrazzanoMonitoringInstance) error {
	vmi.Spec.Opensearch = vmov1.Opensearch{Enabled: false}
	vmi.Spec.OpensearchDashboards = vmov1.OpensearchDashboards{Enabled: false}
	return nil
}

func getOSPersistentVolumes(ctx spi.ComponentContext) ([]v1.PersistentVolume, error) {
	pvList := &v1.PersistentVolumeList{}
	if err := ctx.Client().List(context.TODO(), pvList); err != nil {
		return nil, ctx.Log().ErrorfNewErr("Failed listing persistent volumes: %v", err)
	}

	nodes := ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes
	var openSearchPVList []v1.PersistentVolume
	for _, node := range nodes {
		for i := range pvList.Items {
			pv := pvList.Items[i]
			if pv.Spec.ClaimRef != nil && pv.Spec.ClaimRef.Namespace == constants.VerrazzanoSystemNamespace &&
				getNodeNameFromClaimName(pv.Spec.ClaimRef.Name, nodes) == node.Name {
				openSearchPVList = append(openSearchPVList, pv)
			}
		}
	}
	return openSearchPVList, nil
}

// getNodeNameFromClaimName returns the corresponding node name for a pvc name
func getNodeNameFromClaimName(claimName string, nodes []vzapi.OpenSearchNode) string {
	claimName = strings.TrimPrefix(claimName, "elasticsearch-master-")
	claimName = strings.TrimPrefix(claimName, "vmi-system-")

	// After trimming the above prefix, the pvc name can be as
	// 1. es-data
	// 2. es-data-1
	// 3. es-data-tqxkq
	// 4. es-data-1-8m66v

	// Compare with node names be first removing the suffix
	// Case 2, 3
	lastIndex := strings.LastIndex(claimName, "-")
	if lastIndex != -1 {
		for _, node := range nodes {
			if claimName[:lastIndex] == node.Name {
				return node.Name
			}
		}
		// Case 4
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
	// Case 1
	for _, node := range nodes {
		if claimName == node.Name {
			return node.Name
		}
	}
	return ""
}

func getStorageLabel(nodeName string) string {
	return fmt.Sprintf("%s-%s", clusterName, nodeName)
}
