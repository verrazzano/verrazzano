// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"strings"

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
)

// getOSPersistentVolumes returns the list of older PersistentVolumes used by VMO OpenSearch
func getOSPersistentVolumes(ctx spi.ComponentContext, nodes []vzapi.OpenSearchNode) ([]v1.PersistentVolume, error) {
	pvList := &v1.PersistentVolumeList{}
	if err := ctx.Client().List(context.TODO(), pvList); err != nil {
		return nil, ctx.Log().ErrorfNewErr(common.PVCListingError, err)
	}

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

// setPVsToRetain sets the ReclaimPolicy for older PersistentVolumes to Retain
// Adds a few extra labels used to identify these PVs in later steps
func setPVsToRetain(ctx spi.ComponentContext, nodes []vzapi.OpenSearchNode) error {
	pvList, err := getOSPersistentVolumes(ctx, nodes)
	if err != nil {
		return err
	}
	if len(pvList) <= 0 {
		ctx.Log().Once("PVs already set to Retain or no old PVs found to retain")
		return nil
	}

	for i := range pvList {
		pv := pvList[i]
		oldReclaimPolicy := pv.Spec.PersistentVolumeReclaimPolicy
		pv.Spec.PersistentVolumeReclaimPolicy = v1.PersistentVolumeReclaimRetain

		if pv.Labels == nil {
			pv.Labels = make(map[string]string)
		}
		// If old reclaim policy is not already set, set it
		// So that it can be changed back to its original value later
		_, ok := pv.Labels[constants.OldReclaimPolicyLabel]
		if !ok {
			pv.Labels[constants.OldReclaimPolicyLabel] = string(oldReclaimPolicy)
		}

		// Used to identify all opensearch PVs in later steps
		pv.Labels[constants.StorageForLabel] = clusterName
		// Used to get a list of PVs for each specific nodePool
		pv.Labels[opensearchNodeLabel] = getNodeNameFromClaimName(pv.Spec.ClaimRef.Name, nodes)

		ctx.Log().Debugf("Setting %s to retain", pv.Name)
		if err := ctx.Client().Update(context.TODO(), &pv); err != nil {
			return ctx.Log().ErrorfNewErr("Failed to retain PV %s, will retry: %v", pv.Name, err)
		}
	}
	return nil
}

// createNewPVCs creates new PersistentVolumeClaims for older PersistentVolumes
// based on the opensearch-operator naming convention
func createNewPVCs(ctx spi.ComponentContext, nodes []vzapi.OpenSearchNode) error {
	for _, node := range nodes {
		nodePool := node.Name
		// Get older PVs for this node pool
		pvList, err := common.GetPVsBasedOnLabel(ctx, opensearchNodeLabel, nodePool)
		if err != nil {
			return err
		}
		// Get newly created PVCs for this node pool
		pvcList, err := common.GetPVCsBasedOnLabel(ctx, nodePoolLabel, nodePool)
		if err != nil {
			return err
		}

		// If there are old PVs and all new PVCs are yet to be created, create the remaining PVCs
		// If all new PVCs are already created, do not update as PVCs are immutable after creation
		if len(pvList) > 0 && len(pvcList) < len(pvList) {
			// replicaCount denotes the replica number for which the PVC will be created
			// Initially starts at 0, since the number of newly created PVC will be 0 initially
			replicaCount := len(pvcList)
			for i := range pvList {
				pv := pvList[i]

				// Check if for the given PV, new PVC was already created
				pvcAlreadyCreated := false
				for _, pvc := range pvcList {
					if pv.Name == pvc.Spec.VolumeName {
						pvcAlreadyCreated = true
					}
				}

				if pvcAlreadyCreated {
					continue
				}

				pv.Spec.ClaimRef = nil
				if err := ctx.Client().Update(context.TODO(), &pv); err != nil {
					return err
				}

				// As per opensearch-operator naming convention
				newPVCName := fmt.Sprintf("data-%s-%s-%d", clusterName, nodePool, replicaCount)

				err := createPVCFromPV(ctx, pv, types.NamespacedName{Namespace: constants.VerrazzanoLoggingNamespace, Name: newPVCName}, nodePool)
				if err != nil {
					return err
				}
				replicaCount++
			}
		} else {
			ctx.Log().Oncef("New PVCs already created or no existing PVs for node pool %s", node.Name)
		}
	}

	return nil
}

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

		// Add labels required by opensearch-operator
		labels := make(map[string]string)
		labels[clusterLabel] = clusterName
		labels[nodePoolLabel] = nodePool

		pvc.Labels = labels
		pvc.Spec.VolumeName = volume.Name
		return nil
	})
	return err
}

// deleteMasterNodePVC deletes the leftover PVCs for the master node
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

// arePVsReleased return true if all PersistentVolumes are in Released Phase
// so that new PersistentVolumeClaims can claim them
func arePVsReleased(ctx spi.ComponentContext, nodes []vzapi.OpenSearchNode) bool {
	pvList, err := getOSPersistentVolumes(ctx, nodes)
	if err != nil {
		ctx.Log().Errorf("Failed to list PVs: %v", err)
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

// resetReclaimPolicy resets the ReclaimPolicy to its original value
func resetReclaimPolicy(ctx spi.ComponentContext) error {
	if err := common.ResetVolumeReclaimPolicy(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to reset reclaim policy for PVs in post-install: %v", err)
	}

	return nil
}

// arePVCsAndPVsBound checks if all the newly created PVCs and older PVs are bound or not
func arePVCsAndPVsBound(ctx spi.ComponentContext) bool {
	pvcList := &v1.PersistentVolumeClaimList{}
	if err := ctx.Client().List(context.TODO(), pvcList, c.MatchingLabels{clusterLabel: clusterName}); err != nil {
		return errors.IsNotFound(err)
	}
	for _, pvc := range pvcList.Items {
		if pvc.Status.Phase != v1.ClaimBound {
			ctx.Log().Progressf("Waiting for pvc %s to bind to pv", pvc.Name)
			return false
		}
	}
	pvList, err := common.GetPersistentVolumes(ctx, clusterName)
	if err != nil {
		return errors.IsNotFound(err)
	}
	for _, pv := range pvList.Items {
		if pv.Status.Phase != v1.VolumeBound {
			ctx.Log().Progressf("Waiting for pv %s to bind to pvc", pv.Name)
		}
	}
	return true
}
