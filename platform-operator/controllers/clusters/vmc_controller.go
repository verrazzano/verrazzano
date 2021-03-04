// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"fmt"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const (
	roleForManagedClusterName = "verrazzano-managed-cluster"
	managedClusterYamlKey     = "managed-cluster.yaml"
	configMapKind             = "ConfigMap"
	configMapVersion          = "v1"
	verrazzanoSystemNamespace = "verrazzano-system"
	prometheusConfigMapName   = "vmi-system-prometheus-config"
	prometheusYamlKey         = "prometheus.yml"
	prometheusConfigBasePath  = "/etc/prometheus/config/"
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

	// If the VerrazzanoManagedCluster is being deleted then return success
	if vmc.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
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

	err = r.setupPrometheusScraper(ctx, vmc)
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

// setupPrometheusScraper will create a scrape configuration for the cluster and update the prometheus config map.  There will also be an
// entry for the cluster's CA cert added to the prometheus config map to allow for lookup of the CA cert by the scraper's HTTP client.
func (r *VerrazzanoManagedClusterReconciler) setupPrometheusScraper(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// read the configuration secret specified
	if vmc.Spec.PrometheusSecret != "" {
		var secret corev1.Secret
		secretNsn := types.NamespacedName{
			Namespace: vmc.Namespace,
			Name:      vmc.Spec.PrometheusSecret,
		}
		if err := r.Get(context.TODO(), secretNsn, &secret); err != nil {
			return fmt.Errorf("Failed to fetch the managed cluster prometheus secret %s/%s, %v", vmc.Namespace, vmc.Spec.PrometheusSecret, err)
		}
		// mutate the prometheus system configuration config map, adding the scraper config for the managed cluster

		// obtain the configuration data from the prometheus secret
		config, ok := secret.Data[managedClusterYamlKey]
		if !ok {
			return fmt.Errorf("Managed clsuter yaml configuration not found")
		}
		// marshal the data into the prometheus info struct
		prometheusConfig := prometheusInfo{}
		err := yaml.Unmarshal(config, &prometheusConfig)
		if err != nil {
			return fmt.Errorf("Unable to umarshal the configuration data")
		}

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
		controllerutil.CreateOrUpdate(ctx, r.Client, promConfigMap, func() error {
			err := r.mutatePrometheusConfigMap(vmc, promConfigMap, prometheusConfig)
			if err != nil {
				return err
			}
			return nil
		})
	}
	return nil
}

// mutatePrometheusConfigMap will add a scraper configuration and a CA cert entry to the prometheus config map
func (r *VerrazzanoManagedClusterReconciler) mutatePrometheusConfigMap(vmc *clustersv1alpha1.VerrazzanoManagedCluster, configMap *corev1.ConfigMap, info prometheusInfo) error {
	cfg := &promconfig.Config{}
	err := yaml.Unmarshal([]byte(configMap.Data[prometheusYamlKey]), cfg)
	if err != nil {
		r.log.Errorf("Failed to unmarshal prometheus configuration: %v", err)
		return err
	}
	scrapeConfig := r.getScrapeConfig(vmc, info)
	// see if an entry exists - if it does, update it
	found := false
	for i := range cfg.ScrapeConfigs {
		if cfg.ScrapeConfigs[i].JobName == vmc.ClusterName {
			found = true
			cfg.ScrapeConfigs[i] = &scrapeConfig
			break
		}
	}
	if !found {
		cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, &scrapeConfig)
	}
	newConfig, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	configMap.Data[prometheusYamlKey] = string(newConfig)
	// add the ca entry to the configmap
	configMap.Data[getCAKey(vmc)] = info.Prometheus.CaCrt

	return nil
}

// getCAKey returns the key by which the CA cert will be retrieved by the scaper HTTP client
func getCAKey(vmc *clustersv1alpha1.VerrazzanoManagedCluster) string {
	return "ca-" + vmc.ClusterName
}

// getScraperConfig creates and returns a configuration based on the data available in the cluster's prometheus secret
func (r *VerrazzanoManagedClusterReconciler) getScrapeConfig(vmc *clustersv1alpha1.VerrazzanoManagedCluster, info prometheusInfo) promconfig.ScrapeConfig {
	scrapeConfig := promconfig.ScrapeConfig{
		JobName:        vmc.ClusterName,
		ScrapeInterval: model.Duration(20 * time.Second),
		ScrapeTimeout:  model.Duration(15 * time.Second),
		Scheme:         "https",
		HTTPClientConfig: promconfig.HTTPClientConfig{
			BasicAuth: &promconfig.BasicAuth{
				Username: "verrazzano",
				Password: promconfig.Secret(info.Prometheus.AuthPasswd),
			},
			TLSConfig: promconfig.TLSConfig{
				CAFile: prometheusConfigBasePath + getCAKey(vmc),
			},
		},
		ServiceDiscoveryConfig: promconfig.ServiceDiscoveryConfig{
			StaticConfigs: []*promconfig.TargetGroup{
				{
					Targets: []model.LabelSet{
						{model.AddressLabel: model.LabelValue(info.Prometheus.Host)},
					},
				},
			},
		},
	}
	return scrapeConfig
}
