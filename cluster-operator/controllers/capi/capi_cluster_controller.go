// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/vmc"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	finalizerName                = "verrazzano.io/capi-cluster"
	clusterProvisionerLabel      = "cluster.verrazzano.io/provisioner"
	clusterStatusSuffix          = "-cluster-status"
	clusterIDKey                 = "clusterId"
	clusterRegistrationStatusKey = "clusterRegistration"
	registrationRetrieved        = "started"
	registrationInitiated        = "completed"
)

type CAPIClusterReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	Log                *zap.SugaredLogger
	RancherIngressHost string
}

type ClusterRegistrationFnType func(ctx context.Context, r *CAPIClusterReconciler, cluster *unstructured.Unstructured) (ctrl.Result, error)

var clusterRegistrationFn ClusterRegistrationFnType = ensureRancherRegistration

func SetClusterRegistrationFunction(f ClusterRegistrationFnType) {
	clusterRegistrationFn = f
}

func SetDefaultClusterRegistrationFunction() {
	clusterRegistrationFn = ensureRancherRegistration
}

type ClusterUnregistrationFnType func(ctx context.Context, r *CAPIClusterReconciler, cluster *unstructured.Unstructured) error

var clusterUnregistrationFn ClusterUnregistrationFnType = UnregisterRancherCluster

func SetClusterUnregistrationFunction(f ClusterUnregistrationFnType) {
	clusterUnregistrationFn = f
}

func SetDefaultClusterUnregistrationFunction() {
	clusterUnregistrationFn = UnregisterRancherCluster
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
	r.Log.Infof("Reconciling CAPI cluster: %v", req.NamespacedName)

	// Is Rancher available
	err := ready.DeploymentsAreAvailable(r.Client, []types.NamespacedName{{
		Namespace: common.CattleSystem,
		Name:      common.RancherName,
	}})
	if err != nil {
		return vzctrl.LongRequeue(), nil
	}

	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(gvk)
	err = r.Get(context.TODO(), req.NamespacedName, cluster)
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
		r.Log.Infof("CAPI cluster %v created by Rancher is registered via VMC processing", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// if the deletion timestamp is set, unregister the corresponding Rancher cluster
	if !cluster.GetDeletionTimestamp().IsZero() {
		if vzstring.SliceContainsString(cluster.GetFinalizers(), finalizerName) {
			if err := clusterUnregistrationFn(ctx, r, cluster); err != nil {
				return ctrl.Result{}, err
			}
		}

		if err := r.removeFinalizer(cluster); err != nil {
			return ctrl.Result{}, err
		}

		// delete the cluster id secret
		clusterRegistrationStatusSecret, err := r.getClusterRegistrationStatusSecret(ctx, cluster)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = r.Delete(ctx, clusterRegistrationStatusSecret)
		if err != nil {
			if !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// add a finalizer to the CAPI cluster if it doesn't already exist
	if err = r.ensureFinalizer(cluster); err != nil {
		return ctrl.Result{}, err
	}

	if registrationInitiated != r.getClusterRegistrationStatus(ctx, cluster) {
		// wait for kubeconfig and complete registration on workload cluster
		if result, err := clusterRegistrationFn(ctx, r, cluster); err != nil {
			return result, err
		}
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

// ensureRancherRegistration ensures that the CAPI cluster is registered with Rancher.
func ensureRancherRegistration(ctx context.Context, r *CAPIClusterReconciler, cluster *unstructured.Unstructured) (ctrl.Result, error) {
	kubeconfig, err := r.getWorkloadClusterKubeconfig(ctx, cluster)
	if err != nil {
		// requeue since we're waiting for cluster
		return vzctrl.ShortRequeue(), err
	}

	rc, log, err := r.GetRancherAPIResources(cluster)
	if err != nil {
		r.Log.Infof("Failed getting rancher api resources")
		return ctrl.Result{}, err
	}

	clusterID := r.getClusterID(ctx, cluster)
	registryYaml, clusterID, registryErr := vmc.RegisterManagedClusterWithRancher(rc, cluster.GetName(), clusterID, log)
	// persist the cluster ID, if present, even if the registry yaml was not returned
	err = r.persistClusterStatus(ctx, cluster, clusterID, registrationRetrieved)
	if err != nil {
		return ctrl.Result{}, err
	}
	// handle registry failure error
	if registryErr != nil {
		r.Log.Error(err, "failed to obtain registration manifest from Rancher")
		return ctrl.Result{}, registryErr
	}
	// it appears that in some circumstances the registry yaml may be empty so need to re-queue to re-attempt retrieval
	if len(registryYaml) == 0 {
		return vzctrl.ShortRequeue(), nil
	}
	// create workload cluster client
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		r.Log.Warnf("Failed getting rest config from workload kubeconfig")
		return ctrl.Result{}, err
	}
	workloadClient, err := r.getWorkloadClusterClient(restConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	// apply registration yaml to managed cluster
	yamlApplier := k8sutil.NewYAMLApplier(workloadClient, "")
	err = yamlApplier.ApplyS(registryYaml)
	if err != nil {
		r.Log.Infof("Failed applying Rancher registration yaml in workload cluster")
		return ctrl.Result{}, err
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
	
	err = r.persistClusterStatus(ctx, cluster, clusterID, registrationInitiated)
	if err != nil {
		r.Log.Infof("Failed to perist cluster status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// getWorkloadClusterClient returns a controller runtime client configured for the workload cluster
func (r *CAPIClusterReconciler) getWorkloadClusterClient(restConfig *rest.Config) (client.Client, error) {
	scheme := runtime.NewScheme()
	_ = rbacv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	_ = netv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	return client.New(restConfig, client.Options{Scheme: scheme})
}

// getClusterID returns the cluster ID assigned by Rancher for the given cluster
func (r *CAPIClusterReconciler) getClusterID(ctx context.Context, cluster *unstructured.Unstructured) string {
	clusterID := ""

	regStatusSecret, err := r.getClusterRegistrationStatusSecret(ctx, cluster)
	if err != nil {
		return clusterID
	}
	clusterID = string(regStatusSecret.Data[clusterIDKey])

	return clusterID
}

// getClusterRegistrationStatus returns the Rancher registration status for the cluster
func (r *CAPIClusterReconciler) getClusterRegistrationStatus(ctx context.Context, cluster *unstructured.Unstructured) string {
	clusterStatus := registrationRetrieved

	regStatusSecret, err := r.getClusterRegistrationStatusSecret(ctx, cluster)
	if err != nil {
		return clusterStatus
	}
	clusterStatus = string(regStatusSecret.Data[clusterRegistrationStatusKey])

	return clusterStatus
}

// getClusterRegistrationStatusSecret returns the secret that stores cluster status information
func (r *CAPIClusterReconciler) getClusterRegistrationStatusSecret(ctx context.Context, cluster *unstructured.Unstructured) (*v1.Secret, error) {
	clusterIDSecret := &v1.Secret{}
	secretName := types.NamespacedName{
		Namespace: constants.VerrazzanoCAPINamespace,
		Name:      cluster.GetName() + clusterStatusSuffix,
	}
	err := r.Get(ctx, secretName, clusterIDSecret)
	if err != nil {
		return nil, err
	}
	return clusterIDSecret, err
}

// persistClusterStatus stores the cluster status in the cluster status secret
func (r *CAPIClusterReconciler) persistClusterStatus(ctx context.Context, cluster *unstructured.Unstructured, clusterID string, status string) error {
	r.Log.Debugf("Persisting cluster %s cluster id: %s", cluster.GetName(), clusterID)
	clusterRegistrationStatusSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetName() + clusterStatusSuffix,
			Namespace: constants.VerrazzanoCAPINamespace,
		},
	}
	_, err := ctrl.CreateOrUpdate(ctx, r.Client, clusterRegistrationStatusSecret, func() error {
		// Build the secret data
		if clusterRegistrationStatusSecret.Data == nil {
			clusterRegistrationStatusSecret.Data = make(map[string][]byte)
		}
		clusterRegistrationStatusSecret.Data[clusterIDKey] = []byte(clusterID)
		clusterRegistrationStatusSecret.Data[clusterRegistrationStatusKey] = []byte(status)

		return nil
	})
	if err != nil {
		r.Log.Errorf("Unable to persist status for cluster %s: %v", cluster.GetName(), err)
		return err
	}

	return nil
}

// getWorkloadClusterKubeconfig returns a kubeconfig for accessing the workload cluster
func (r *CAPIClusterReconciler) getWorkloadClusterKubeconfig(ctx context.Context, cluster *unstructured.Unstructured) ([]byte, error) {
	// get the cluster kubeconfig
	kubeconfigSecret := &v1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-kubeconfig", cluster.GetName()), Namespace: cluster.GetNamespace()}, kubeconfigSecret)
	if err != nil {
		r.Log.Warn(err, "failed to obtain workload cluster kubeconfig resource. Re-queuing...")
		return nil, err
	}
	kubeconfig, ok := kubeconfigSecret.Data["value"]
	if !ok {
		r.Log.Error(err, "failed to read kubeconfig from resource")
		return nil, fmt.Errorf("Unable to read kubeconfig from retrieved cluster resource")
	}

	return kubeconfig, nil
}

// GetRancherAPIResources returns the set of resources required for interacting with Rancher
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
func UnregisterRancherCluster(ctx context.Context, r *CAPIClusterReconciler, cluster *unstructured.Unstructured) error {
	clusterID := r.getClusterID(ctx, cluster)
	if len(clusterID) == 0 {
		// no cluster id found
		return fmt.Errorf("No cluster ID found for cluster %s", cluster.GetName())
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
