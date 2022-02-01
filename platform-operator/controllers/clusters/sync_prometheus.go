// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/verrazzano/verrazzano/pkg/constants"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

const (
	prometheusConfigMapName  = "vmi-system-prometheus-config"
	prometheusYamlKey        = "prometheus.yml"
	scrapeConfigsKey         = "scrape_configs"
	jobNameKey               = "job_name"
	prometheusConfigBasePath = "/etc/prometheus/config/"
	configMapKind            = "ConfigMap"
	configMapVersion         = "v1"
	scrapeConfigTemplate     = `job_name: ##JOB_NAME##
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
	}

	// Get the Prometheus configuration.  The ConfigMap may not exist if this delete is being called during an uninstall of Verrazzano.
	promConfigMap, err := r.getPrometheusConfig(ctx, vmc)
	if err != nil {
		r.log.Infof("Failed adding Prometheus configuration for managed cluster %s: %v", vmc.ClusterName, err)
		return nil
	}

	// Update Prometheus configuration to stop scraping for this VMC
	err = r.mutatePrometheusConfigMap(vmc, promConfigMap, &secret)
	if err != nil {
		return err
	}
	err = r.Client.Update(ctx, promConfigMap)
	if err != nil {
		return err
	}
	return nil
}

// getPrometheusConfig will get the ConfigMap containing the Prometheus configuration
func (r *VerrazzanoManagedClusterReconciler) getPrometheusConfig(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) (*corev1.ConfigMap, error) {
	var promConfigMap = &corev1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: prometheusConfigMapName, Namespace: constants.VerrazzanoSystemNamespace}, promConfigMap)
	if err != nil {
		return nil, err
	}
	return promConfigMap, nil
}

// mutatePrometheusConfigMap will add a scraper configuration and a CA cert entry to the prometheus config map
func (r *VerrazzanoManagedClusterReconciler) mutatePrometheusConfigMap(vmc *clustersv1alpha1.VerrazzanoManagedCluster, configMap *corev1.ConfigMap, cacrtSecret *v1.Secret) error {
	prometheusConfig, err := parsePrometheusConfig(configMap.Data[prometheusYamlKey])
	if err != nil {
		return err
	}

	oldScrapeConfigs := prometheusConfig.Path(scrapeConfigsKey).Children()
	prometheusConfig.Array(scrapeConfigsKey) // zero out the array of scrape configs

	// create the new scrape config
	newScrapeConfig, err := r.newScrapeConfig(cacrtSecret, vmc)
	if err != nil {
		return err
	}
	if newScrapeConfig == nil {
		//deletion
		delete(configMap.Data, getCAKey(vmc))
	}
	existingReplaced := false
	for _, oldScrapeConfig := range oldScrapeConfigs {
		oldScrapeJob := oldScrapeConfig.Search(jobNameKey).Data()
		if vmc.Name == oldScrapeJob {
			if vmc.DeletionTimestamp == nil || vmc.DeletionTimestamp.IsZero() {
				// need to replace existing entry for this vmc
				prometheusConfig.ArrayAppendP(newScrapeConfig.Data(), scrapeConfigsKey)
				cacrt := cacrtSecret.Data["cacrt"]
				if cacrt != nil {
					cacrtValue := string(cacrt)
					if len(cacrtValue) > 0 {
						// cert configured for scraper - needs to be added to config map
						configMap.Data[getCAKey(vmc)] = cacrtValue
					}
				}
				existingReplaced = true
			}
		} else {
			prometheusConfig.ArrayAppendP(oldScrapeConfig.Data(), scrapeConfigsKey)
		}
	}
	if !existingReplaced && newScrapeConfig != nil {
		prometheusConfig.ArrayAppendP(newScrapeConfig.Data(), scrapeConfigsKey)
		configMap.Data[getCAKey(vmc)] = string(cacrtSecret.Data["cacrt"])
	}

	bytes, err := yaml.JSONToYAML(prometheusConfig.Bytes())
	if err != nil {
		return err
	}
	configMap.Data[prometheusYamlKey] = string(bytes)

	return nil
}

// newScrapeConfig will return a prometheus scraper configuration based on the entries in the prometheus info structure provided
func (r *VerrazzanoManagedClusterReconciler) newScrapeConfig(cacrtSecret *v1.Secret, vmc *clustersv1alpha1.VerrazzanoManagedCluster) (*gabs.Container, error) {
	var newScrapeConfig *gabs.Container
	if cacrtSecret == nil || vmc.Status.PrometheusHost == "" {
		return newScrapeConfig, nil
	}

	vzPromSecret, err := r.getSecret(constants.VerrazzanoSystemNamespace, constants.VerrazzanoPromInternal, true)
	if err != nil {
		return nil, err
	}

	newScrapeConfigMappings := map[string]string{
		"##JOB_NAME##":     vmc.Name,
		"##HOST##":         vmc.Status.PrometheusHost,
		"##PASSWORD##":     string(vzPromSecret.Data[VerrazzanoPasswordKey]),
		"##CLUSTER_NAME##": vmc.Name}
	configTemplate := scrapeConfigTemplate
	for key, value := range newScrapeConfigMappings {
		configTemplate = strings.ReplaceAll(configTemplate, key, value)
	}

	newScrapeConfig, err = parseScrapeConfig(configTemplate)
	if err != nil {
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
	// Get the Prometheus configuration.  The ConfigMap may not exist if this delete is being called during an uninstall of Verrazzano.
	promConfigMap, err := r.getPrometheusConfig(ctx, vmc)
	if err != nil {
		r.log.Infof("Failed deleting Prometheus configuration for managed cluster %s: %v", vmc.ClusterName, err)
		return nil
	}

	// Update Prometheus configuration to stop scraping for this VMC
	err = r.mutatePrometheusConfigMap(vmc, promConfigMap, nil)
	if err != nil {
		return err
	}
	err = r.Client.Update(ctx, promConfigMap)
	if err != nil {
		return err
	}
	return nil
}

// parseScrapeConfig returns an editable representation of the prometheus scrape configuration
func parseScrapeConfig(scrapeConfigStr string) (*gabs.Container, error) {
	scrapeConfigJSON, _ := yaml.YAMLToJSON([]byte(scrapeConfigStr))
	newScrapeConfig, err := gabs.ParseJSON(scrapeConfigJSON)
	if err != nil {
		return nil, err
	}
	return newScrapeConfig, nil
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
