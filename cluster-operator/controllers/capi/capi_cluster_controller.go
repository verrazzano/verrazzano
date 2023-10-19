// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/capi"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	finalizerName = "verrazzano.io/capi-cluster"
)

type CAPIClusterReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	Log                *zap.SugaredLogger
	RancherIngressHost string
	RancherEnabled     bool
}

func CAPIClusterClientObject() client.Object {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(capi.GVKCAPICluster)
	return obj
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *CAPIClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(CAPIClusterClientObject()).
		Complete(r)
}

// Reconcile is the main controller reconcile function
func (r *CAPIClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Infof("Reconciling CAPI cluster: %v", req.NamespacedName)

	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(capi.GVKCAPICluster)
	err := r.Get(context.TODO(), req.NamespacedName, cluster)
	if err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if errors.IsNotFound(err) {
		r.Log.Debugf("CAPI cluster %v not found, nothing to do", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// if the deletion timestamp is set, unregister the corresponding Rancher cluster
	if !cluster.GetDeletionTimestamp().IsZero() {
		if vzstring.SliceContainsString(cluster.GetFinalizers(), finalizerName) {
			vmcName := r.getVMCName(cluster)
			// ensure a base VMC
			vmc := &clustersv1alpha1.VerrazzanoManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vmcName,
					Namespace: constants.VerrazzanoMultiClusterNamespace,
				},
			}
			if err = r.Delete(ctx, vmc); err != nil {
				if !errors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
			}
		}

		if err := r.removeFinalizer(cluster); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// obtain and persist the API endpoint IP address for the admin cluster
	err = r.createAdminAccessConfigMap(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	vmcName := r.getVMCName(cluster)
	// ensure a base VMC
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmcName,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
	}
	if _, err = r.createOrUpdateWorkloadClusterVMC(ctx, cluster, vmc, func() error {
		vmc.Spec = clustersv1alpha1.VerrazzanoManagedClusterSpec{
			Description: fmt.Sprintf("%s VerrazzanoManagedCluster Resource", cluster.GetName()),
		}
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	// add a finalizer to the CAPI cluster if it doesn't already exist
	if err = r.ensureFinalizer(cluster); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createOrUpdateWorkloadClusterVMC creates or updates the VMC resource for the workload cluster
func (r *CAPIClusterReconciler) createOrUpdateWorkloadClusterVMC(ctx context.Context, cluster *unstructured.Unstructured, vmc *clustersv1alpha1.VerrazzanoManagedCluster, f controllerutil.MutateFn) (*clustersv1alpha1.VerrazzanoManagedCluster, error) {
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, vmc, f); err != nil {
		r.Log.Errorf("Failed to create or update the VMC for cluster %s: %v", cluster.GetName(), err)
		return nil, err
	}

	return vmc, nil
}

// createAdminAccessConfigMap creates the config map required for the creation of the admin accessing kubeconfig
func (r *CAPIClusterReconciler) createAdminAccessConfigMap(ctx context.Context) error {
	ep := &v1.Endpoints{}
	if err := r.Get(ctx, types.NamespacedName{Name: "kubernetes", Namespace: "default"}, ep); err != nil {
		return err
	}
	apiServerIP := ep.Subsets[0].Addresses[0].IP

	// create the admin server IP config map
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "verrazzano-admin-cluster",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data["server"] = fmt.Sprintf("https://%s:6443", apiServerIP)

		return nil
	}); err != nil {
		r.Log.Errorf("Failed to create the Verrazzano admin cluster config map: %v", err)
		return err
	}
	return nil
}

// ensureFinalizer adds a finalizer to the CAPI cluster if the finalizer is not already present
func (r *CAPIClusterReconciler) ensureFinalizer(cluster *unstructured.Unstructured) error {
	if finalizers, added := vzstring.SliceAddString(cluster.GetFinalizers(), finalizerName); added {
		cluster.SetFinalizers(finalizers)
		if err := r.Update(context.TODO(), cluster); err != nil {
			return err
		}
	}
	return nil
}

// removeFinalizer removes the finalizer from the Rancher cluster resource
func (r *CAPIClusterReconciler) removeFinalizer(cluster *unstructured.Unstructured) error {
	finalizers := vzstring.RemoveStringFromSlice(cluster.GetFinalizers(), finalizerName)
	cluster.SetFinalizers(finalizers)

	if err := r.Update(context.TODO(), cluster); err != nil {
		return err
	}
	return nil
}

func (r *CAPIClusterReconciler) getVMCName(cluster *unstructured.Unstructured) string {
	// check for existence of a Rancher cluster management resource
	rancherMgmtCluster := &unstructured.Unstructured{}
	rancherMgmtCluster.SetGroupVersionKind(common.GetRancherMgmtAPIGVKForKind("Cluster"))
	err := r.Get(context.TODO(), types.NamespacedName{Name: cluster.GetName(), Namespace: cluster.GetNamespace()}, rancherMgmtCluster)
	if err != nil {
		return cluster.GetName()
	}
	// return the display Name
	return rancherMgmtCluster.UnstructuredContent()["spec"].(map[string]interface{})["displayName"].(string)
}
