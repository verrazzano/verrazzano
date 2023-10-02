// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	internalcapi "github.com/verrazzano/verrazzano/cluster-operator/internal/capi"
	"github.com/verrazzano/verrazzano/pkg/constants"
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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	finalizerName                = "verrazzano.io/capi-cluster"
	clusterProvisionerLabel      = "cluster.verrazzano.io/provisioner"
	clusterStatusSuffix          = "-cluster-status"
	clusterIDKey                 = "clusterId"
	clusterRegistrationStatusKey = "clusterRegistration"
	registrationRetrieved        = "retrieved"
	registrationInitiated        = "initiated"
)

type CAPIClusterReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Log                 *zap.SugaredLogger
	RancherRegistrar    *RancherRegistration
	RancherIngressHost  string
	RancherEnabled      bool
	VerrazzanoRegistrar *VerrazzanoRegistration
}

func CAPIClusterClientObject() client.Object {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(internalcapi.GVKCAPICluster)
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
	cluster.SetGroupVersionKind(internalcapi.GVKCAPICluster)
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
			err = r.unregisterCluster(ctx, cluster)
			if err != nil {
				r.Log.Warnf("Unable to unregister cluster %s: %v.  Cluster deletion will proceed.", cluster.GetName(), err)
			}
		}

		if err := r.removeFinalizer(cluster); err != nil {
			return ctrl.Result{}, err
		}

		// delete the cluster id secret
		clusterRegistrationStatusSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.VerrazzanoCAPINamespace,
				Name:      cluster.GetName() + clusterStatusSuffix,
			},
		}
		err = r.Delete(ctx, clusterRegistrationStatusSecret)
		if err != nil {
			if !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
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
	if err = r.setVMCClusterRef(ctx, cluster, vmc); err != nil {
		return ctrl.Result{}, err
	}

	if r.RancherEnabled {
		// Is Rancher Deployment ready
		r.Log.Debugf("Attempting cluster regisration with Rancher")
		result, err := r.RancherRegistrar.doReconcile(ctx, cluster)
		if err != nil {
			return result, err
		}
	}

	// add a finalizer to the CAPI cluster if it doesn't already exist
	if err = r.ensureFinalizer(cluster); err != nil {
		return ctrl.Result{}, err
	}

	r.Log.Debugf("Attempting cluster regisration with Verrazzano")
	return verrazzanoReconcileFn(ctx, cluster, r)
}

func (r *CAPIClusterReconciler) setVMCClusterRef(ctx context.Context, cluster *unstructured.Unstructured, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	vmc.Status.ClusterRef = &clustersv1alpha1.ClusterReference{
		APIVersion: cluster.GetAPIVersion(),
		Kind:       cluster.GetKind(),
		Name:       cluster.GetName(),
		Namespace:  cluster.GetNamespace(),
	}
	return r.Client.Status().Update(ctx, vmc, &client.UpdateOptions{})
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

// unregisterCluster removes the cluster from Rancher and/or Verrazzano.
func (r *CAPIClusterReconciler) unregisterCluster(ctx context.Context, cluster *unstructured.Unstructured) error {
	var err error
	if r.RancherEnabled {
		err = clusterRancherUnregistrationFn(ctx, r.RancherRegistrar, cluster)
		if err != nil {
			return err
		}
	}
	if err = UnregisterVerrazzanoCluster(ctx, r.VerrazzanoRegistrar, cluster); err != nil {
		return err
	}

	// remove the VMC
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetName(),
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
	}
	err = r.Delete(ctx, vmc)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Infof("VMC for cluster %s not found - nothing to do", cluster.GetName())
			return nil
		}
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

// getClusterClient returns a controller runtime client configured for the workload cluster
func getClusterClient(restConfig *rest.Config) (client.Client, error) {
	scheme := runtime.NewScheme()
	_ = rbacv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	_ = netv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = clustersv1alpha1.AddToScheme(scheme)

	return client.New(restConfig, client.Options{Scheme: scheme})
}

// getClusterID returns the cluster ID assigned by Rancher for the given cluster
func getClusterID(ctx context.Context, client client.Client, cluster *unstructured.Unstructured) string {
	clusterID := ""

	regStatusSecret, err := getClusterRegistrationStatusSecret(ctx, client, cluster)
	if err != nil {
		return clusterID
	}
	clusterID = string(regStatusSecret.Data[clusterIDKey])

	return clusterID
}

// getClusterRegistrationStatus returns the Rancher registration status for the cluster
func getClusterRegistrationStatus(ctx context.Context, c client.Client, cluster *unstructured.Unstructured) string {
	clusterStatus := registrationRetrieved

	regStatusSecret, err := getClusterRegistrationStatusSecret(ctx, c, cluster)
	if err != nil {
		return clusterStatus
	}
	clusterStatus = string(regStatusSecret.Data[clusterRegistrationStatusKey])

	return clusterStatus
}

// getClusterRegistrationStatusSecret returns the secret that stores cluster status information
func getClusterRegistrationStatusSecret(ctx context.Context, c client.Client, cluster *unstructured.Unstructured) (*v1.Secret, error) {
	clusterIDSecret := &v1.Secret{}
	secretName := types.NamespacedName{
		Namespace: constants.VerrazzanoCAPINamespace,
		Name:      cluster.GetName() + clusterStatusSuffix,
	}
	err := c.Get(ctx, secretName, clusterIDSecret)
	if err != nil {
		return nil, err
	}
	return clusterIDSecret, err
}

// persistClusterStatus stores the cluster status in the cluster status secret
func persistClusterStatus(ctx context.Context, client client.Client, cluster *unstructured.Unstructured, log *zap.SugaredLogger, clusterID string, status string) error {
	log.Debugf("Persisting cluster %s cluster id: %s", cluster.GetName(), clusterID)
	clusterRegistrationStatusSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetName() + clusterStatusSuffix,
			Namespace: constants.VerrazzanoCAPINamespace,
		},
	}
	_, err := ctrl.CreateOrUpdate(ctx, client, clusterRegistrationStatusSecret, func() error {
		// Build the secret data
		if clusterRegistrationStatusSecret.Data == nil {
			clusterRegistrationStatusSecret.Data = make(map[string][]byte)
		}
		clusterRegistrationStatusSecret.Data[clusterIDKey] = []byte(clusterID)
		clusterRegistrationStatusSecret.Data[clusterRegistrationStatusKey] = []byte(status)

		return nil
	})
	if err != nil {
		log.Errorf("Unable to persist status for cluster %s: %v", cluster.GetName(), err)
		return err
	}

	return nil
}

// getWorkloadClusterKubeconfig returns a kubeconfig for accessing the workload cluster
func getWorkloadClusterKubeconfig(client client.Client, cluster *unstructured.Unstructured, log *zap.SugaredLogger) ([]byte, error) {
	// get the cluster kubeconfig
	kubeconfigSecret := &v1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-kubeconfig", cluster.GetName()), Namespace: cluster.GetNamespace()}, kubeconfigSecret)
	if err != nil {
		log.Warn(err, "failed to obtain workload cluster kubeconfig resource. Re-queuing...")
		return nil, err
	}
	kubeconfig, ok := kubeconfigSecret.Data["value"]
	if !ok {
		log.Error(err, "failed to read kubeconfig from resource")
		return nil, fmt.Errorf("Unable to read kubeconfig from retrieved cluster resource")
	}

	return kubeconfig, nil
}

func getWorkloadClusterClient(client client.Client, log *zap.SugaredLogger, cluster *unstructured.Unstructured) (client.Client, error) {
	// identify whether the workload cluster is using "untrusted" certs
	kubeconfig, err := getWorkloadClusterKubeconfig(client, cluster, log)
	if err != nil {
		// requeue since we're waiting for cluster
		return nil, err
	}
	// create a workload cluster client
	// create workload cluster client
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		log.Warnf("Failed getting rest config from workload kubeconfig")
		return nil, err
	}
	workloadClient, err := getClusterClient(restConfig)
	if err != nil {
		return nil, err
	}
	return workloadClient, nil
}
