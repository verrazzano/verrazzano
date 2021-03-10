// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"fmt"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	roleForManagedClusterName = "verrazzano-managed-cluster"
	configMapKind             = "ConfigMap"
	configMapVersion          = "v1"
	verrazzanoSystemNamespace = "verrazzano-system"
	finalizerName             = "managedcluster.verrazzano.io"
)

// VerrazzanoManagedClusterReconciler reconciles a VerrazzanoManagedCluster object.
// The reconciler will create a ServiceAcount, ClusterRoleBinding, and a Secret which
// contains the kubeconfig to be used by the Multi-Cluster Agent to access the admin cluster.
type VerrazzanoManagedClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	log    *zap.SugaredLogger
}

// bindingParams used to mutate the ClusterRoleBinding
type bindingParams struct {
	vmc                     *clustersv1alpha1.VerrazzanoManagedCluster
	roleBindingName         string
	roleName                string
	serviceAccountName      string
	serviceAccountNamespace string
}

// prometheusConfig contains the information required to create a scrape configuration
type prometheusConfig struct {
	AuthPasswd string `yaml:"authpasswd"`
	Host       string `yaml:"host"`
	CaCrt      string `yaml:"cacrt"`
}

// prometheusInfo wraps the prometheus configuration info
type prometheusInfo struct {
	Prometheus prometheusConfig `yaml:"prometheus"`
}

// +kubebuilder:rbac:groups=clusters.verrazzano.io,resources=verrazzanomanagedclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clusters.verrazzano.io,resources=verrazzanomanagedclusters/status,verbs=get;update;patch

// Reconcile reconciles a VerrazzanoManagedCluster object
func (r *VerrazzanoManagedClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	log := zap.S().With("resource", fmt.Sprintf("%s:%s", req.Namespace, req.Name))
	r.log = log
	log.Info("Reconciler called")
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{}

	err := r.Get(ctx, req.NamespacedName, vmc)
	if err != nil {
		// If the resource is not found, that means all of the finalizers have been removed,
		// and the verrazzano resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		// Error getting the VerrazzanoManagedCluster resource
		log.Errorf("Failed to fetch resource: %v", err)
		return reconcile.Result{}, err
	}

	if !vmc.ObjectMeta.DeletionTimestamp.IsZero() {
		// Finalizer is present, so lets do the uninstall
		if containsString(vmc.ObjectMeta.Finalizers, finalizerName) {
			if err := r.reconcileManagedClusterDelete(ctx, vmc); err != nil {
				return reconcile.Result{}, err
			}

			// Remove the finalizer and update the verrazzano resource if the uninstall has finished.
			log.Infof("Removing finalizer %s", finalizerName)
			vmc.ObjectMeta.Finalizers = removeString(vmc.ObjectMeta.Finalizers, finalizerName)
			err := r.Update(ctx, vmc)
			if err != nil && !errors.IsConflict(err) {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add our finalizer if not already added
	if !containsString(vmc.ObjectMeta.Finalizers, finalizerName) {
		log.Infof("Adding finalizer %s", finalizerName)
		vmc.ObjectMeta.Finalizers = append(vmc.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(ctx, vmc); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Sync the service account
	err = r.syncServiceAccount(vmc)
	if err != nil {
		log.Infof("Failed to sync the ServiceAccount: %v", err)
		return ctrl.Result{}, err
	}

	err = r.syncManagedRoleBinding(vmc)
	if err != nil {
		log.Infof("Failed to sync the ServiceAccount: %v", err)
		return ctrl.Result{}, err
	}

	err = r.syncRegistrationSecret(vmc)
	if err != nil {
		log.Infof("Failed to sync the registration used by managed cluster: %v", err)
		return ctrl.Result{}, err
	}

	err = r.syncElasticsearchSecret(vmc)
	if err != nil {
		log.Infof("Failed to sync the Elasticsearch secret used by managed cluster: %v", err)
		return ctrl.Result{}, err
	}

	err = r.syncManifestSecret(vmc)
	if err != nil {
		log.Infof("Failed to sync the YAML manifest secret used by managed cluster: %v", err)
		return ctrl.Result{}, err
	}

	err = r.syncPrometheusScraper(ctx, vmc)
	if err != nil {
		log.Infof("Failed to setup the prometheus scraper for managed cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *VerrazzanoManagedClusterReconciler) syncServiceAccount(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// Create or update the service account
	_, err := r.createOrUpdateServiceAccount(context.TODO(), vmc)
	if err != nil {
		return err
	}

	// Does the VerrazzanoManagedCluster object contain the service account name?
	saName := generateManagedResourceName(vmc.Name)
	if vmc.Spec.ServiceAccount != saName {
		r.log.Infof("Updating ServiceAccount from %q to %q", vmc.Spec.ServiceAccount, saName)
		vmc.Spec.ServiceAccount = saName
		err = r.Update(context.TODO(), vmc)
		if err != nil {
			return err
		}
	}

	return nil
}

// Create or update the ServiceAccount for a VerrazzanoManagedCluster
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateServiceAccount(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) (controllerutil.OperationResult, error) {
	var serviceAccount corev1.ServiceAccount
	serviceAccount.Namespace = vmc.Namespace
	serviceAccount.Name = generateManagedResourceName(vmc.Name)

	return controllerutil.CreateOrUpdate(ctx, r.Client, &serviceAccount, func() error {
		r.mutateServiceAccount(vmc, &serviceAccount)
		// This SetControllerReference call will trigger garbage collection i.e. the serviceAccount
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &serviceAccount, r.Scheme)
	})
}

func (r *VerrazzanoManagedClusterReconciler) mutateServiceAccount(vmc *clustersv1alpha1.VerrazzanoManagedCluster, serviceAccount *corev1.ServiceAccount) {
	serviceAccount.Name = generateManagedResourceName(vmc.Name)
}

// syncManagedRoleBinding syncs the ClusterRoleBinding that binds the service account used by the managed cluster
// to the role containing the permission
func (r *VerrazzanoManagedClusterReconciler) syncManagedRoleBinding(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	bindingName := generateManagedResourceName(vmc.Name)
	var binding rbacv1.ClusterRoleBinding
	binding.Name = bindingName

	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &binding, func() error {
		mutateBinding(&binding, bindingParams{
			vmc:                     vmc,
			roleBindingName:         bindingName,
			roleName:                roleForManagedClusterName,
			serviceAccountName:      vmc.Spec.ServiceAccount,
			serviceAccountNamespace: vmc.Namespace,
		})
		return nil
	})
	return err
}

// mutateBinding mutates the ClusterRoleBinding to ensure it has the valid params
func mutateBinding(binding *rbacv1.ClusterRoleBinding, p bindingParams) {
	binding.ObjectMeta = metav1.ObjectMeta{
		Name:   p.roleBindingName,
		Labels: p.vmc.Labels,
		// Set owner reference here instead of calling controllerutil.SetControllerReference
		// which does not allow cluster-scoped resources.
		// This reference will result in the clusterrolebinding resource being deleted
		// when the verrazzano CR is deleted.
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: p.vmc.APIVersion,
				Kind:       p.vmc.Kind,
				Name:       p.vmc.Name,
				UID:        p.vmc.UID,
				Controller: func() *bool {
					flag := true
					return &flag
				}(),
			},
		},
	}
	binding.RoleRef = rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     p.roleName,
	}
	binding.Subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      p.serviceAccountName,
			Namespace: p.serviceAccountNamespace,
		},
	}
}

// Generate the common name used by all resources specific to a given managed cluster
func generateManagedResourceName(clusterName string) string {
	return fmt.Sprintf("verrazzano-cluster-%s", clusterName)
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoManagedClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.VerrazzanoManagedCluster{}).
		Complete(r)
}

func (r *VerrazzanoManagedClusterReconciler) reconcileManagedClusterDelete(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// get and mutate the prometheus config map
	promConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       configMapKind,
			APIVersion: configMapVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: verrazzanoSystemNamespace,
			Name:      prometheusConfigMapName,
		}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, promConfigMap, func() error {
		err := r.mutatePrometheusConfigMap(vmc, promConfigMap, nil)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// containsString checks for a string in a slice of strings
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// removeString removes a string from a slice of strings
func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
