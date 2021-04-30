// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
static_configs:
- targets:
  - ##HOST##
  labels:
    managed_cluster: '##CLUSTER_NAME##'
basic_auth:
  username: verrazzano-prom-internal
  password: ##PASSWORD##
`
)

// prometheusConfig contains the information required to create a scrape configuration
type prometheusConfig struct {
	Host  string `yaml:"host"`
	CaCrt string `yaml:"cacrt"`
}

// prometheusInfo wraps the prometheus configuration info
type prometheusInfo struct {
	Prometheus prometheusConfig `yaml:"prometheus"`
}

// syncPrometheusScraper will create a scrape configuration for the cluster and update the prometheus config map.  There will also be an
// entry for the cluster's CA cert added to the prometheus config map to allow for lookup of the CA cert by the scraper's HTTP client.
func (r *VerrazzanoManagedClusterReconciler) syncPrometheusScraper(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// read the configuration secret specified
	if len(vmc.Spec.PrometheusSecret) == 0 {
		return nil
	}

	var secret corev1.Secret
	secretNsn := types.NamespacedName{
		Namespace: vmc.Namespace,
		Name:      vmc.Spec.PrometheusSecret,
	}
	if err := r.Get(context.TODO(), secretNsn, &secret); err != nil {
		return fmt.Errorf("Failed to fetch the managed cluster prometheus secret %s/%s, %v", vmc.Namespace, vmc.Spec.PrometheusSecret, err)
	}

	// obtain the configuration data from the prometheus secret
	config, ok := secret.Data[getClusterYamlKey(vmc.Name)]
	if !ok {
		return fmt.Errorf("Managed cluster yaml configuration not found")
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
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      prometheusConfigMapName,
		}}
	controllerutil.CreateOrUpdate(ctx, r.Client, promConfigMap, func() error {
		err := r.mutatePrometheusConfigMap(vmc, promConfigMap, &prometheusConfig)
		if err != nil {
			return err
		}
		return nil
	})

	return nil
}

// mutatePrometheusConfigMap will add a scraper configuration and a CA cert entry to the prometheus config map
func (r *VerrazzanoManagedClusterReconciler) mutatePrometheusConfigMap(vmc *clustersv1alpha1.VerrazzanoManagedCluster, configMap *corev1.ConfigMap, info *prometheusInfo) error {
	prometheusConfig, err := parsePrometheusConfig(configMap.Data[prometheusYamlKey])
	if err != nil {
		return err
	}

	oldScrapeConfigs := prometheusConfig.Path(scrapeConfigsKey).Children()
	prometheusConfig.Array(scrapeConfigsKey) // zero out the array of scrape configs

	// create the new scrape config
	newScrapeConfig, err := r.newScrapeConfig(info, vmc)
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
				if len(info.Prometheus.CaCrt) > 0 {
					// cert configured for scraper - needs to be added to config map
					configMap.Data[getCAKey(vmc)] = info.Prometheus.CaCrt
				}
				existingReplaced = true
			}
		} else {
			prometheusConfig.ArrayAppendP(oldScrapeConfig.Data(), scrapeConfigsKey)
		}
	}
	if !existingReplaced && newScrapeConfig != nil {
		prometheusConfig.ArrayAppendP(newScrapeConfig.Data(), scrapeConfigsKey)
		configMap.Data[getCAKey(vmc)] = info.Prometheus.CaCrt
	}

	bytes, err := yaml.JSONToYAML(prometheusConfig.Bytes())
	if err != nil {
		return err
	}
	configMap.Data[prometheusYamlKey] = string(bytes)

	return nil
}

// newScrapeConfig will return a prometheus scraper configuration based on the entries in the prometheus info structure provided
func (r *VerrazzanoManagedClusterReconciler) newScrapeConfig(info *prometheusInfo, vmc *clustersv1alpha1.VerrazzanoManagedCluster) (*gabs.Container, error) {
	var newScrapeConfig *gabs.Container
	if info == nil {
		return newScrapeConfig, nil
	}

	vzPromSecret, err := r.getVzPromSecret()
	if err != nil {
		return nil, err
	}

	newScrapeConfigMappings := map[string]string{
		"##JOB_NAME##":     vmc.Name,
		"##HOST##":         info.Prometheus.Host,
		"##PASSWORD##":     string(vzPromSecret.Data[PasswordKey]),
		"##CLUSTER_NAME##": vmc.Name}
	configTemplate := scrapeConfigTemplate
	for key, value := range newScrapeConfigMappings {
		configTemplate = strings.ReplaceAll(configTemplate, key, value)
	}

	newScrapeConfig, err = parseScrapeConfig(configTemplate)
	if err != nil {
		return nil, err
	}
	if len(info.Prometheus.CaCrt) > 0 {
		newScrapeConfig.Set(prometheusConfigBasePath+getCAKey(vmc), "tls_config", "ca_file")
		newScrapeConfig.Set(false, "tls_config", "insecure_skip_verify")
	}
	return newScrapeConfig, nil
}

// deleteClusterPrometheusConfiguration deletes the managed cluster configuration from the prometheus configuration and updates the prometheus config
// map
func (r *VerrazzanoManagedClusterReconciler) deleteClusterPrometheusConfiguration(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// Get the Prometheus configuration.  The ConfigMap may not exist if this delete is being called
	// during an uninstall of Verrazzano.
	var promConfigMap = &corev1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: prometheusConfigMapName, Namespace: constants.VerrazzanoSystemNamespace}, promConfigMap)
	if err != nil {
		r.log.Warn(fmt.Sprintf("unable to delete Prometheus configuration for managed cluster %s: %v", vmc.ClusterName, err))
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

// getClusterYamlKey returns the key to the cluster information yaml from the configured prometheus secret
func getClusterYamlKey(name string) string {
	return fmt.Sprintf("%s.yaml", name)
}
