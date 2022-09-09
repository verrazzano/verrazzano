// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	c "sigs.k8s.io/controller-runtime/pkg/client"
)

// RetainPersistentVolume locates the persistent volume associated with the provided pvc
// and sets the reclaim policy to "retain" so that it can be migrated to the new deployment/statefulset.
func RetainPersistentVolume(ctx spi.ComponentContext, pvc *v1.PersistentVolumeClaim, componentName string) error {
	ctx.Log().Infof("Updating persistent volume associated with pvc %s so that the volume can be migrated", pvc.Name)

	pvName := pvc.Spec.VolumeName
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}

	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), pv, func() error {
		oldReclaimPolicy := pv.Spec.PersistentVolumeReclaimPolicy
		pv.Spec.PersistentVolumeReclaimPolicy = v1.PersistentVolumeReclaimRetain

		// add labels to the pv - one that allows the new deployment to select the volume and another that captures
		// the old reclaim policy so we can set it back once the volume is bound
		if pv.Labels == nil {
			pv.Labels = make(map[string]string)
		}
		pv.Labels[vzconst.StorageForLabel] = componentName
		if _, ok := pv.Labels[vzconst.OldReclaimPolicyLabel]; !ok {
			pv.Labels[vzconst.OldReclaimPolicyLabel] = string(oldReclaimPolicy)
		}
		return nil
	})

	return err
}

// UpdateExistingVolumeClaims removes a persistent volume claim from the volume if the
// status is "released". This allows the new deployment to bind to the existing volume.
func UpdateExistingVolumeClaims(ctx spi.ComponentContext, pvcName types.NamespacedName, newClaimName string, componentName string) error {
	ctx.Log().Debugf("Removing old claim from persistent volume if a volume exists")

	pvList, err := GetPersistentVolumes(ctx, componentName)
	if err != nil {
		return err
	}
	pvs := pvList.Items

	// find a volume that has been released but still has a claim for old deployment
	for i := range pvs {
		pv := pvs[i] // avoids "Implicit memory aliasing in for loop" linter complaint
		ctx.Log().Debugf("Update PV %s.  Current status: %s", pv.Name, pv.Status.Phase)
		if pv.Status.Phase == v1.VolumeBound {
			return ctx.Log().ErrorfNewErr("PV %s is still bound", pv.Namespace)
		}

		if pv.Spec.ClaimRef != nil && pv.Spec.ClaimRef.Namespace == pvcName.Namespace && pv.Spec.ClaimRef.Name == pvcName.Name {
			ctx.Log().Infof("Removing old claim from persistent volume %s", pv.Name)
			pv.Spec.ClaimRef = nil
			if err := ctx.Client().Update(context.TODO(), &pv); err != nil {
				return ctx.Log().ErrorfNewErr("Failed removing claim from persistent volume %s: %v", pv.Name, err)
			}
			// create a new PVC pointing to the existing PV
			if err := createPVCFromPV(ctx, pv, types.NamespacedName{Namespace: pvcName.Namespace, Name: newClaimName}); err != nil {
				return ctx.Log().ErrorfNewErr("Failed to create new PVC from volume %s: %v", pv.Name, err)
			}
			break
		}
	}
	return nil
}

// DeleteExistingVolumeClaims removes a persistent volume claim in order to allow the pv to be reclaimed by a new PVC.
func DeleteExistingVolumeClaim(ctx spi.ComponentContext, pvcName types.NamespacedName) error {
	ctx.Log().Debugf("Removing PVC %v", pvcName)

	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName.Name,
			Namespace: pvcName.Namespace,
		},
	}
	ctx.Log().Debugf("Deleting pvc %v", pvcName)
	if err := ctx.Client().Delete(context.TODO(), pvc); err != nil {
		if errors.IsNotFound(err) {
			ctx.Log().Debugf("PVC %v is not found", pvcName)
			return nil
		}
		ctx.Log().Errorf("Unable to delete PVC %v", pvcName)
		return err
	}

	return nil
}

// ResetVolumeReclaimPolicy resets the reclaim policy on a volume to its original value (prior to upgrade).
func ResetVolumeReclaimPolicy(ctx spi.ComponentContext, componentName string) error {
	ctx.Log().Debugf("Resetting reclaim policy on volume if a volume exists")

	pvList, err := GetPersistentVolumes(ctx, componentName)
	if err != nil {
		return err
	}

	for i := range pvList.Items {
		pv := pvList.Items[i] // avoids "Implicit memory aliasing in for loop" linter complaint
		ctx.Log().Infof("ResetVolumeReclaimPolicy - PV %s status: %s", pv.Name, pv.Status.Phase)
		if pv.Status.Phase != v1.VolumeBound {
			continue
		}

		if pv.Labels == nil {
			continue
		}
		oldPolicy, ok := pv.Labels[vzconst.OldReclaimPolicyLabel]

		if ok {
			// found a bound volume that still has an old reclaim policy label, so reset the reclaim policy and remove the label
			ctx.Log().Debugf("Resetting reclaim policy on persistent volume %s to %s", pv.Name, oldPolicy)
			pv.Spec.PersistentVolumeReclaimPolicy = v1.PersistentVolumeReclaimPolicy(oldPolicy)
			delete(pv.Labels, vzconst.OldReclaimPolicyLabel)

			if err := ctx.Client().Update(context.TODO(), &pv); err != nil {
				return ctx.Log().ErrorfNewErr("Failed resetting reclaim policy on persistent volume %s: %v", pv.Name, err)
			}
		}
	}
	return nil
}

// GetPersistentVolumes returns a volume list containing a persistent volume created by an older chart
func GetPersistentVolumes(ctx spi.ComponentContext, componentName string) (*v1.PersistentVolumeList, error) {
	pvList := &v1.PersistentVolumeList{}
	if err := ctx.Client().List(context.TODO(), pvList, c.MatchingLabels{vzconst.StorageForLabel: componentName}); err != nil {
		if errors.IsNotFound(err) {
			return pvList, nil
		}
		return nil, ctx.Log().ErrorfNewErr("Failed listing persistent volumes: %v", err)
	}
	ctx.Log().Debugf("Found %d volumes associated with component %s", len(pvList.Items), componentName)
	return pvList, nil
}

//createPVCFromPV creates a PVC from a PV definition, and sets the PVC to reference the PV by name
func createPVCFromPV(ctx spi.ComponentContext, volume v1.PersistentVolume, newClaimName types.NamespacedName) error {
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
		pvc.Spec.VolumeName = volume.Name
		return nil
	})
	return err
}
