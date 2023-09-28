// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/vmc"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RancherRegistration struct {
	client.Client
	Log                *zap.SugaredLogger
	RancherIngressHost string
}

type ClusterRancherRegistrationFnType func(ctx context.Context, r *RancherRegistration, cluster *unstructured.Unstructured) (ctrl.Result, error)

var clusterRancherRegistrationFn ClusterRancherRegistrationFnType = ensureRancherRegistration

func SetClusterRancherRegistrationFunction(f ClusterRancherRegistrationFnType) {
	clusterRancherRegistrationFn = f
}

func SetDefaultClusterRancherRegistrationFunction() {
	clusterRancherRegistrationFn = ensureRancherRegistration
}

type ClusterRancherUnregistrationFnType func(ctx context.Context, r *RancherRegistration, cluster *unstructured.Unstructured) error

var clusterRancherUnregistrationFn ClusterRancherUnregistrationFnType = UnregisterRancherCluster

func SetClusterRancherUnregistrationFunction(f ClusterRancherUnregistrationFnType) {
	clusterRancherUnregistrationFn = f
}

func SetDefaultClusterRancherUnregistrationFunction() {
	clusterRancherUnregistrationFn = UnregisterRancherCluster
}

func (r *RancherRegistration) doReconcile(ctx context.Context, cluster *unstructured.Unstructured) (ctrl.Result, error) {
	// only process CAPI cluster instances not managed by Rancher/container driver
	_, ok := cluster.GetLabels()[clusterProvisionerLabel]
	if ok {
		return ctrl.Result{}, nil
	}

	err := ready.DeploymentsAreAvailable(r.Client, []types.NamespacedName{{
		Namespace: common.CattleSystem,
		Name:      common.RancherName,
	}})
	if err != nil {
		return vzctrl.LongRequeue(), nil
	}

	// wait for kubeconfig and complete registration on workload cluster
	return clusterRancherRegistrationFn(ctx, r, cluster)
}

// GetRancherAPIResources returns the set of resources required for interacting with Rancher
func (r *RancherRegistration) GetRancherAPIResources(cluster *unstructured.Unstructured) (*rancherutil.RancherConfig, vzlog.VerrazzanoLogger, error) {
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           cluster.GetName(),
		Namespace:      cluster.GetNamespace(),
		ID:             string(cluster.GetUID()),
		Generation:     cluster.GetGeneration(),
		ControllerName: "capicluster",
	})
	if err != nil {
		r.Log.Errorf("Failed to create controller logger for CAPI cluster controller", err)
		return nil, nil, err
	}

	// using direct rancher API to register cluster
	rc, err := rancherutil.NewAdminRancherConfig(r.Client, r.RancherIngressHost, log)
	if err != nil {
		r.Log.Error(err, "failed to create Rancher API client")
		return nil, nil, err
	}
	return rc, log, nil
}

// UnregisterRancherCluster performs the operations required to de-register the cluster from Rancher
func UnregisterRancherCluster(ctx context.Context, r *RancherRegistration, cluster *unstructured.Unstructured) error {
	_, ok := cluster.GetLabels()[clusterProvisionerLabel]
	if ok {
		return nil
	}

	clusterID := getClusterID(ctx, r.Client, cluster)
	if len(clusterID) == 0 {
		// no cluster id found, nothing to do
		return nil
	}
	rc, log, err := r.GetRancherAPIResources(cluster)
	if err != nil {
		return err
	}
	_, err = vmc.DeleteClusterFromRancher(rc, clusterID, log)
	if err != nil {
		log.Errorf("Unable to unregister cluster %s from Rancher: %v", cluster.GetName(), err)
		return err
	}

	return nil
}

// ensureRancherRegistration ensures that the CAPI cluster is registered with Rancher.
func ensureRancherRegistration(ctx context.Context, r *RancherRegistration, cluster *unstructured.Unstructured) (ctrl.Result, error) {
	workloadClient, err := getWorkloadClusterClient(r.Client, r.Log, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	rc, log, err := r.GetRancherAPIResources(cluster)
	if err != nil {
		r.Log.Infof("Failed getting rancher api resources")
		return ctrl.Result{}, err
	}

	clusterID := getClusterID(ctx, r.Client, cluster)

	// register with Rancher
	registryYaml, clusterID, registryErr := vmc.RegisterManagedClusterWithRancher(rc, cluster.GetName(), clusterID, log)
	// handle registry failure error
	if registryErr != nil {
		r.Log.Error(err, "failed to obtain registration manifest from Rancher")
		return ctrl.Result{}, registryErr
	}
	// it appears that in some circumstances the registry yaml may be empty so need to re-queue to re-attempt retrieval
	if len(registryYaml) == 0 {
		return vzctrl.ShortRequeue(), nil
	}

	if registrationInitiated != getClusterRegistrationStatus(ctx, r.Client, cluster) {
		// apply registration yaml to managed cluster
		yamlApplier := k8sutil.NewYAMLApplier(workloadClient, "")
		err = yamlApplier.ApplyS(registryYaml)
		if err != nil {
			r.Log.Infof("Failed applying Rancher registration yaml in workload cluster")
			return ctrl.Result{}, err
		}
		err = persistClusterStatus(ctx, r.Client, cluster, r.Log, clusterID, registrationInitiated)
		if err != nil {
			r.Log.Infof("Failed to perist cluster status")
			return ctrl.Result{}, err
		}
	}

	// get and label the cattle-system namespace
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: common.CattleSystem}}
	if _, err := ctrl.CreateOrUpdate(context.TODO(), workloadClient, ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels[constants.LabelVerrazzanoNamespace] = common.CattleSystem
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
