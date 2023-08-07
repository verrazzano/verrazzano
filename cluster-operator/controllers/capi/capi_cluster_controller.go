// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/vmc"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vzstring "github.com/verrazzano/verrazzano/pkg/string"
)

const (
	finalizerName           = "verrazzano.io/capi-cluster"
	clusterProvisionerLabel = "cluster.verrazzano.io/provisioner"
	clusterIdLabel          = "cluster.verrazzano.io/rancher-cluster-id"
)

type CAPIClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    *zap.SugaredLogger
}

var gvk = schema.GroupVersionKind{
	Group:   "cluster.x-k8s.io",
	Version: "v1beta1",
	Kind:    "Cluster",
}

func CAPIClusterClientObject() client.Object {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
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
	r.Log.Debugf("Reconciling CAPI cluster: %v", req.NamespacedName)

	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(gvk)
	err := r.Get(context.TODO(), req.NamespacedName, cluster)
	if err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if errors.IsNotFound(err) {
		r.Log.Debugf("CAPI cluster %v not found, nothing to do", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// only process CAPI cluster instances not managed by Rancher/container driver
	_, ok := cluster.GetLabels()[clusterProvisionerLabel]
	if ok {
		r.Log.Debugf("CAPI cluster created by Rancher, nothing to do", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// if the deletion timestamp is set, unregister the corresponding Rancher cluster
	if !cluster.GetDeletionTimestamp().IsZero() {
		if vzstring.SliceContainsString(cluster.GetFinalizers(), finalizerName) {
			if err := r.UnregisterRancherCluster(cluster); err != nil {
				return ctrl.Result{}, err
			}
		}

		if err := r.removeFinalizer(cluster); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// add a finalizer to the Rancher cluster if it doesn't already exist
	if err := r.ensureFinalizer(cluster); err != nil {
		return ctrl.Result{}, err
	}

	// ensure the VMC exists
	if err = r.ensureRancherRegistration(cluster); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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

// getClusterDisplayName returns the displayName spec field from the Rancher cluster resource
func (r *CAPIClusterReconciler) getClusterDisplayName(cluster *unstructured.Unstructured) (string, error) {
	displayName, ok, err := unstructured.NestedString(cluster.Object, "spec", "displayName")
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("Could not find spec displayName field in Cattle Cluster resource: %s", cluster.GetName())
	}
	return displayName, nil
}

// ensureRancherRegistration ensures that the CAPI cluster is registered with Rancher.
func (r *CAPIClusterReconciler) ensureRancherRegistration(cluster *unstructured.Unstructured) error {
	rc, log, err := r.GetRancherAPIResources(cluster)
	if err != nil {
		return err
	}
	labels := cluster.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	clusterId, clusterIdExists := labels[clusterIdLabel]
	registryYaml, clusterId, err := vmc.RegisterManagedClusterWithRancher(rc, cluster.GetName(), clusterId, log)
	// apply registration yaml to managed cluster
	if err != nil {
		log.Error(err, "failed to obtain registration manifest from Rancher")
		return err
	}
	// get the cluster kubeconfig
	kubeconfigSecret := &v1.Secret{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-kubeconfig", cluster.GetName()), Namespace: "default"}, kubeconfigSecret)
	if err != nil {
		log.Error(err, "failed to obtain cluster kubeconfig resource")
		return err
	}
	kubeconfig, ok := kubeconfigSecret.Data["value"]
	if !ok {
		log.Error(err, "failed to read kubeconfig from resource")
		return fmt.Errorf("Unable to read kubeconfig from retrieved cluster resource")
	}
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return err
	}
	err = resource.CreateOrUpdateResourceFromBytesUsingConfig([]byte(registryYaml), restConfig)
	if err != nil {
		return err
	}

	// Update the CAPI cluster status with the Rancher cluster ID
	if !clusterIdExists {
		r.Log.Debugf("Updating cluster %s labels with cluster id: %s", cluster.GetName(), clusterId)
		labels[clusterIdLabel] = clusterId
		cluster.SetLabels(labels)
		if err := r.Update(context.TODO(), cluster); err != nil {
			r.Log.Errorf("Unable to update clsuter %s labels: %v", cluster.GetName(), err)
			return err
		}
	}

	return nil
}

func (r *CAPIClusterReconciler) GetRancherAPIResources(cluster *unstructured.Unstructured) (*rancherutil.RancherConfig, vzlog.VerrazzanoLogger, error) {
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           cluster.GetName(),
		Namespace:      cluster.GetNamespace(),
		ID:             string(cluster.GetUID()),
		Generation:     cluster.GetGeneration(),
		ControllerName: "capicluster",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for CAPI cluster controller", err)
		return nil, nil, err
	}

	// using direct rancher API to register cluster
	ingressHost := rancherutil.DefaultRancherIngressHostPrefix + nginxutil.IngressNGINXNamespace()
	rc, err := rancherutil.NewAdminRancherConfig(r.Client, ingressHost, log)
	if err != nil {
		log.Error(err, "failed to create Rancher API client")
		return nil, nil, err
	}
	return rc, log, nil
}

func (r *CAPIClusterReconciler) UnregisterRancherCluster(cluster *unstructured.Unstructured) error {
	labels := cluster.GetLabels()
	clusterId, ok := labels[clusterIdLabel]
	if !ok {
		// no cluster id label
		return fmt.Errorf("No cluster ID found for cluster %s", cluster.GetName())
	}
	rc, log, err := r.GetRancherAPIResources(cluster)
	if err != nil {
		return err
	}
	_, err = vmc.DeleteClusterFromRancher(rc, clusterId, log)
	if err != nil {
		log.Errorf("Unable to unregister cluster %s from Rancher: %v", cluster.GetName(), err)
		return err
	}

	return nil
}
