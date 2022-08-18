// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	"github.com/verrazzano/verrazzano/pkg/metricsutils"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
)

const (
	scrapeConfigsKey         = "scrape_configs"
	prometheusConfigBasePath = "/etc/prometheus/config/"
	managedCertsBasePath     = "/etc/prometheus/managed-cluster-ca-certs/"
	scrapeConfigTemplate     = constants.PrometheusJobNameKey + `: ##JOB_NAME##
scrape_interval: 20s
scrape_timeout: 15s
scheme: https
honor_labels: true
metrics_path: '/federate'
params:
  'match[]':
   - '{__name__=~"..*"}'
# If an existing verrazzano_cluster metric is present, make sure it is always replaced to
# the right managed cluster name for the cluster. Do this with a metric_relabel_config so it
# happens at the end i.e. _after_ scraping is completed, before ingesting into data source.
metric_relabel_configs:
  - action: replace
    source_labels:
    - verrazzano_cluster
    target_label: verrazzano_cluster
    replacement: '##CLUSTER_NAME##'
static_configs:
- targets:
  - ##HOST##
  labels: # add the labels if not already present on managed cluster (this will no op if present)
    verrazzano_cluster: '##CLUSTER_NAME##'
basic_auth:
  username: verrazzano-prom-internal
  password: ##PASSWORD##
`
)

// syncPrometheusScraper will create a scrape configuration for the cluster and update the prometheus config map.  There will also be an
// entry for the cluster's CA cert added to the prometheus config map to allow for lookup of the CA cert by the scraper's HTTP client.
func (r *VerrazzanoManagedClusterReconciler) syncPrometheusScraper(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	var secret corev1.Secret
	fmt.Println("--Debug--Inside syncPrometheusScraper")
	// read the configuration secret specified if it exists
	if len(vmc.Spec.CASecret) > 0 {
		secretNsn := types.NamespacedName{
			Namespace: vmc.Namespace,
			Name:      vmc.Spec.CASecret,
		}

		// validate secret if it exists
		if err := r.Get(context.TODO(), secretNsn, &secret); err != nil {
			return fmt.Errorf("failed to fetch the managed cluster CA secret %s/%s, %v", vmc.Namespace, vmc.Spec.CASecret, err)
		}
		fmt.Println("--Debug--Secret exists")
	}

	// The additional scrape configs and managed cluster TLS secrets are needed by the Prometheus Operator Prometheus
	// because the federated scrape config can't be represented in a PodMonitor, ServiceMonitor, etc.
	fmt.Println("--Debug--Before mutateManagedClusterCACertsSecret")
	err := r.mutateManagedClusterCACertsSecret(ctx, vmc, &secret)
	if err != nil {
		fmt.Println("--Debug--After mutateManagedClusterCACertsSecret inside error", err)
		return err
	}
	fmt.Println("--Debug--Before mutateAdditionalScrapeConfigs")
	err = r.mutateAdditionalScrapeConfigs(ctx, vmc, &secret)
	if err != nil {
		fmt.Println("--Debug--After mutateAdditionalScrapeConfigs inside err block", err)
		return err
	}

	return nil
}

// newScrapeConfig will return a prometheus scraper configuration based on the entries in the prometheus info structure provided
func (r *VerrazzanoManagedClusterReconciler) newScrapeConfig(cacrtSecret *v1.Secret, vmc *clustersv1alpha1.VerrazzanoManagedCluster) (*gabs.Container, error) {
	var newScrapeConfig *gabs.Container
	fmt.Println("--Debug--newScrapeConfig Before if cacrtSecret == nil || vmc.Status.PrometheusHost == '' ")
	if cacrtSecret == nil || vmc.Status.PrometheusHost == "" {
		fmt.Println("--Debug--newScrapeConfig Inside if cacrtSecret == nil || vmc.Status.PrometheusHost == '' ")
		return newScrapeConfig, nil
	}
	fmt.Println("--Debug--newScrapeConfig Before vzPromSecret getsecret ")
	vzPromSecret, err := r.getSecret(constants.VerrazzanoSystemNamespace, constants.VerrazzanoPromInternal, true)
	if err != nil {
		fmt.Println("--Debug--newScrapeConfig Inside vzPromSecret error ", err)
		return nil, err
	}
	fmt.Println("--Debug--newScrapeConfig After vzPromSecret getsecret")

	newScrapeConfigMappings := map[string]string{
		"##JOB_NAME##":     vmc.Name,
		"##HOST##":         vmc.Status.PrometheusHost,
		"##PASSWORD##":     string(vzPromSecret.Data[mcconstants.VerrazzanoPasswordKey]),
		"##CLUSTER_NAME##": vmc.Name}
	configTemplate := scrapeConfigTemplate
	for key, value := range newScrapeConfigMappings {
		configTemplate = strings.ReplaceAll(configTemplate, key, value)
	}
	fmt.Println("--Debug--newScrapeConfig Before newScrapeConfig ParseScrapeConfig")
	newScrapeConfig, err = metricsutils.ParseScrapeConfig(configTemplate)
	fmt.Println("--Debug--newScrapeConfig After newScrapeConfig ParseScrapeConfig")
	if err != nil {
		fmt.Println("--Debug--newScrapeConfig Inside error of newScrapeConfig ParseScrapeConfig")
		return nil, err
	}
	if len(cacrtSecret.Data["cacrt"]) > 0 {
		newScrapeConfig.Set(prometheusConfigBasePath+getCAKey(vmc), "tls_config", "ca_file")
		newScrapeConfig.Set(false, "tls_config", "insecure_skip_verify")
	}
	return newScrapeConfig, nil
}

// deleteClusterPrometheusConfiguration deletes the managed cluster configuration from the prometheus configuration and updates the prometheus config
// map
func (r *VerrazzanoManagedClusterReconciler) deleteClusterPrometheusConfiguration(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	err := r.mutateAdditionalScrapeConfigs(ctx, vmc, nil)
	if err != nil {
		return err
	}
	err = r.mutateManagedClusterCACertsSecret(ctx, vmc, nil)
	if err != nil {
		return err
	}

	return nil
}

// parsePrometheusConfig returns an editable representation of the prometheus configuration
func parsePrometheusConfig(promConfigStr string) (*gabs.Container, error) {
	jsonConfig, err := yaml.YAMLToJSON([]byte(promConfigStr))
	if err != nil {
		return nil, err
	}
	prometheusConfig, err := gabs.ParseJSON(jsonConfig)
	if err != nil {
		return nil, err
	}
	return prometheusConfig, err
}

// getCAKey returns the key by which the CA cert will be retrieved by the scaper HTTP client
func getCAKey(vmc *clustersv1alpha1.VerrazzanoManagedCluster) string {
	return "ca-" + vmc.Name
}

// mutateAdditionalScrapeConfigs adds and removes scrape config for managed clusters to the additional scrape configurations secret. Prometheus Operator appends the raw scrape config
// in this secret to the scrape config it generates from PodMonitor and ServiceMonitor resources.
func (r *VerrazzanoManagedClusterReconciler) mutateAdditionalScrapeConfigs(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster, cacrtSecret *v1.Secret) error {
	// get the existing additional scrape config, if the secret doesn't exist we will create it
	fmt.Println("--Debug--mutateAdditionalScrapeConfigsBefore getting the secret")
	secret, err := r.getSecret(vpoconst.VerrazzanoMonitoringNamespace, constants.PromAdditionalScrapeConfigsSecretName, false)
	fmt.Println("--Debug--mutateAdditionalScrapeConfigs After getting the secret")
	if err != nil && !errors.IsNotFound(err) {
		fmt.Println("--Debug--mutateAdditionalScrapeConfigs Inside error block", err)
		return err
	}
	var jobsStr string
	if secret.Data != nil {
		fmt.Println("--Debug--mutateAdditionalScrapeConfigs inside secret.Data", err)
		jobsStr = string(secret.Data[constants.PromAdditionalScrapeConfigsSecretKey])
	}

	// create the scrape config for the new managed cluster
	fmt.Println("--Debug--mutateAdditionalScrapeConfigs Before newScrapeConfig")
	newScrapeConfig, err := r.newScrapeConfig(cacrtSecret, vmc)
	if err != nil {
		fmt.Println("--Debug--mutateAdditionalScrapeConfigs Inside error of  newScrapeConfig", err)
		return err
	}
	// TODO: Set this in the newScrapeConfig function when we remove the "old" Prometheus code
	newScrapeConfig.Set(managedCertsBasePath+getCAKey(vmc), "tls_config", "ca_file")

	editScrapeJobName := vmc.Name
	fmt.Println("--Debug--mutateAdditionalScrapeConfigs before ParseScrapeConfig")
	// parse the scrape config so we can manipulate it
	jobs, err := metricsutils.ParseScrapeConfig(jobsStr)
	if err != nil {
		fmt.Println("--Debug--mutateAdditionalScrapeConfigs inside error block of ParseScrapeConfig", err)
		return err
	}
	fmt.Println("--Debug--mutateAdditionalScrapeConfigs before metricsutils.EditScrapeJob")
	scrapeConfigs, err := metricsutils.EditScrapeJob(jobs, editScrapeJobName, newScrapeConfig)
	if err != nil {
		fmt.Println("--Debug--mutateAdditionalScrapeConfigs Inside error block of  metricsutils.EditScrapeJob", err)
		return err
	}
	fmt.Println("--Debug--mutateAdditionalScrapeConfigs before  yaml.JSONToYAML(scrapeConfigs.Bytes())")
	bytes, err := yaml.JSONToYAML(scrapeConfigs.Bytes())
	if err != nil {
		fmt.Println("--Debug--mutateAdditionalScrapeConfigs Inside error block  yaml.JSONToYAML(scrapeConfigs.Bytes())", err)
		return err
	}

	// update the secret with the updated scrape config
	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.PromAdditionalScrapeConfigsSecretName,
			Namespace: vpoconst.VerrazzanoMonitoringNamespace,
		},
		Data: map[string][]byte{},
	}
	if _, err := controllerruntime.CreateOrUpdate(ctx, r.Client, &secret, func() error {
		secret.Data[constants.PromAdditionalScrapeConfigsSecretKey] = bytes
		return nil
	}); err != nil {
		fmt.Println("--Debug--mutateAdditionalScrapeConfigs Inside error of  controllerruntime.CreateOrUpdate", err)
		return err
	}

	return nil
}

// mutateManagedClusterCACertsSecret adds and removes managed cluster CA certs to/from the managed cluster CA certs secret
func (r *VerrazzanoManagedClusterReconciler) mutateManagedClusterCACertsSecret(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster, cacrtSecret *v1.Secret) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vpoconst.PromManagedClusterCACertsSecretName,
			Namespace: vpoconst.VerrazzanoMonitoringNamespace,
		},
	}
	fmt.Println("--Debug--Inside mutateManagedClusterCACertsSecret")
	if _, err := controllerruntime.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if secret.Data == nil {
			fmt.Println("--Debug--secret.Data == nil mutateManagedClusterCACertsSecret")
			secret.Data = make(map[string][]byte)
		}
		if cacrtSecret != nil && cacrtSecret.Data != nil && len(cacrtSecret.Data["cacrt"]) > 0 {
			fmt.Println("--Debug--secret.Data == nil mutateManagedClusterCACertsSecret cacrtSecret != nil && cacrtSecret.Data != nil && --- > 0")
			secret.Data[getCAKey(vmc)] = cacrtSecret.Data["cacrt"]
		} else {
			fmt.Println("--Debug--secret.Data == nil mutateManagedClusterCACertsSecret before delete(secret.Data, getCAKey(vmc))")
			delete(secret.Data, getCAKey(vmc))
		}
		return nil
	}); err != nil {
		fmt.Println("--Debug--Inside mutateManagedClusterCACertsSecret err occurred")
		return err
	}

	return nil
}
