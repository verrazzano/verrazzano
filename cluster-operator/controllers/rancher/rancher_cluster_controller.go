// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
)

const (
	CreatedByLabel      = "app.kubernetes.io/created-by"
	CreatedByVerrazzano = "verrazzano"
	localClusterName    = "local"

	finalizerName = "verrazzano.io/rancher-cluster"
)

type RancherClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    *zap.SugaredLogger
}

var gvk = schema.GroupVersionKind{
	Group:   "management.cattle.io",
	Version: "v3",
	Kind:    "Cluster",
}

func CattleClusterClientObject() client.Object {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	return obj
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *RancherClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(CattleClusterClientObject()).
		Complete(r)
}

// Reconcile is the main controller reconcile function
func (r *RancherClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Debugf("Reconciling Rancher cluster: %v", req.NamespacedName)

	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(gvk)
	err := r.Get(context.TODO(), req.NamespacedName, cluster)
	if err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if errors.IsNotFound(err) {
		r.Log.Debugf("Rancher cluster %v not found, nothing to do", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// if the deletion timestamp is set, delete the corresponding VMC
	if !cluster.GetDeletionTimestamp().IsZero() {
		if vzstring.SliceContainsString(cluster.GetFinalizers(), finalizerName) {
			if err := r.deleteVMC(cluster); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, r.removeFinalizer(cluster)
	}

	// add a finalizer to the Rancher cluster if it doesn't already exist
	if err := r.ensureFinalizer(cluster); err != nil {
		return ctrl.Result{}, err
	}

	// ensure the VMC exists
	if err = r.ensureVMC(cluster); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// ensureFinalizer adds a finalizer to the Rancher cluster if the finalizer is not already present
func (r *RancherClusterReconciler) ensureFinalizer(cluster *unstructured.Unstructured) error {
	// do not add finalizer to the "local" cluster resource
	if localClusterName == cluster.GetName() {
		return nil
	}
	if finalizers, added := vzstring.SliceAddString(cluster.GetFinalizers(), finalizerName); added {
		cluster.SetFinalizers(finalizers)
		if err := r.Update(context.TODO(), cluster); err != nil {
			return err
		}
	}
	return nil
}

// removeFinalizer removes the finalizer from the Rancher cluster resource
func (r *RancherClusterReconciler) removeFinalizer(cluster *unstructured.Unstructured) error {
	finalizers := vzstring.RemoveStringFromSlice(cluster.GetFinalizers(), finalizerName)
	cluster.SetFinalizers(finalizers)

	if err := r.Update(context.TODO(), cluster); err != nil {
		return err
	}
	return nil
}

// getClusterDisplayName returns the displayName spec field from the Rancher cluster resource
func (r *RancherClusterReconciler) getClusterDisplayName(cluster *unstructured.Unstructured) (string, error) {
	displayName, ok, err := unstructured.NestedString(cluster.Object, "spec", "displayName")
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("Could not find spec displayName field in Cattle Cluster resource: %s", cluster.GetName())
	}
	return displayName, nil
}

// ensureVMC ensures that a VMC exists for the Rancher cluster. It will also set the Rancher cluster id in the VMC status
// if it is not already set.
func (r *RancherClusterReconciler) ensureVMC(cluster *unstructured.Unstructured) error {
	// ignore the "local" cluster
	if localClusterName == cluster.GetName() {
		return nil
	}

	displayName, err := r.getClusterDisplayName(cluster)
	if err != nil {
		return err
	}

	// attempt to create the VMC, if it already exists we just ignore the error
	vmc := newVMC(displayName)
	if err := r.Create(context.TODO(), vmc); err != nil {
		if !errors.IsAlreadyExists(err) {
			r.Log.Errorf("Unable to create VMC with name %s: %v", displayName, err)
			return err
		}
		r.Log.Debugf("VMC %s already exists", displayName)
	} else {
		r.Log.Infof("Created VMC for discovered Rancher cluster with name: %s", displayName)
	}

	// read back the VMC and if the cluster id isn't set in the status, set it
	if err := r.Get(context.TODO(), types.NamespacedName{Name: displayName, Namespace: vzconst.VerrazzanoMultiClusterNamespace}, vmc); err != nil {
		r.Log.Errorf("Unable to get VMC with name: %s", displayName)
		return err
	}
	if vmc.Status.RancherRegistration.ClusterID == "" {
		// Rancher cattle cluster resource name is also the Rancher cluster id
		r.Log.Debugf("Updating VMC %s status with cluster id: %s", displayName, cluster.GetName())
		vmc.Status.RancherRegistration.ClusterID = cluster.GetName()
		if err := r.Status().Update(context.TODO(), vmc); err != nil {
			r.Log.Errorf("Unable to update VMC %s status: %v", displayName, err)
			return err
		}
	}

	return nil
}

// deleteVMC deletes the VMC associated with a Rancher cluster
func (r *RancherClusterReconciler) deleteVMC(cluster *unstructured.Unstructured) error {
	// ignore the "local" cluster
	if localClusterName == cluster.GetName() {
		return nil
	}

	displayName, err := r.getClusterDisplayName(cluster)
	if err != nil {
		return err
	}

	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: displayName, Namespace: vzconst.VerrazzanoMultiClusterNamespace}, vmc); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Debugf("VMC %s does not exist, nothing to do", displayName)
			return nil
		}
		return err
	}

	// if the VMC has a cluster id in the status, delete the VMC
	if len(vmc.Status.RancherRegistration.ClusterID) > 0 {
		r.Log.Infof("Deleting VMC %s because it is no longer in Rancher", vmc.Name)
		if err := r.Delete(context.TODO(), vmc); err != nil {
			r.Log.Errorf("Unable to delete VMC %s: %v", vmc.Name, err)
			return err
		}
	}

	return nil
}

// newVMC returns a minimally populated VMC object
func newVMC(name string) *clustersv1alpha1.VerrazzanoManagedCluster {
	return &clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: vzconst.VerrazzanoMultiClusterNamespace,
			Labels: map[string]string{
				CreatedByLabel:                    CreatedByVerrazzano,
				vzconst.VerrazzanoManagedLabelKey: "true",
			},
		},
	}
}
