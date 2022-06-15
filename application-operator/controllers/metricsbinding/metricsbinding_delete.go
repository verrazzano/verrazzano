// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"
	"fmt"

	"github.com/Jeffail/gabs/v2"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/metricsutils"
	k8scorev1 "k8s.io/api/core/v1"
	k8scontroller "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// reconcileBindingDelete completes the reconcile process for an object that is being deleted
func (r *Reconciler) reconcileBindingDelete(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (k8scontroller.Result, error) {
	log.Debugw("Reconcile for deleted object", "resource", metricsBinding.GetName())

	// If a ConfigMap is populated, delete the existing scrape config from the ConfigMap
	var configMap = getPromConfigMap(metricsBinding)
	if configMap != nil {
		log.Debugf("ConfigMap %s/%s found in the MetricsBinding, deleting scrape config", configMap.GetName(), configMap.GetNamespace())
		if err := r.deleteFromPrometheusConfigMap(ctx, metricsBinding, configMap, log); err != nil {
			return k8scontroller.Result{Requeue: true}, err
		}
	}
	// If the Secret exists, delete the existing config from the Secret
	secret, key := getPromConfigSecret(metricsBinding)
	if secret != nil {
		log.Debugf("Secret %s/%s found in the MetricsBinding, deleting scrape config", secret.GetName(), secret.GetNamespace())
		if err := r.deleteFromPrometheusConfigSecret(ctx, metricsBinding, secret, key, log); err != nil {
			return k8scontroller.Result{Requeue: true}, err
		}
	}

	// Remove the finalizer if deletion was successful
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, metricsBinding, func() error {
		controllerutil.RemoveFinalizer(metricsBinding, finalizerName)
		return nil
	})
	if err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	return k8scontroller.Result{}, nil
}

// deleteFromPrometheusConfigMap deletes the scrape config from the Prometheus ConfigMap
func (r *Reconciler) deleteFromPrometheusConfigMap(ctx context.Context, metricsBinding *vzapi.MetricsBinding, configMap *k8scorev1.ConfigMap, log vzlog.VerrazzanoLogger) error {
	log.Debugw("Prometheus target ConfigMap is being altered", "resource", configMap.GetName())
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		// Get data from the configmap
		promConfig, err := getConfigData(configMap)
		if err != nil {
			return log.ErrorfNewErr("Failed to get Prometheus ConfigMap Data: %v", err)
		}
		// scrape configs would have been edited in-place, in the promConfig Container, so ignore
		// that return value
		_, err = r.deleteScrapeConfig(metricsBinding, promConfig, log, true)
		if err != nil {
			return log.ErrorfNewErr("Failed to delete scrape config from Prometheus ConfigMap: %v", err)
		}
		// scrape configs would have been edited in-place, in the promConfig Container, so serialize
		// the whole thing for the new data.
		newPromConfigData, err := yaml.JSONToYAML(promConfig.Bytes())
		if err != nil {
			return log.ErrorfNewErr("Failed to convert scrape config JSON to YAML: %v", err)
		}
		configMap.Data[prometheusConfigKey] = string(newPromConfigData)
		return nil
	})
	return err
}

// deleteFromPrometheusConfigSecret deletes the scrape config from the Prometheus config Secret
func (r *Reconciler) deleteFromPrometheusConfigSecret(ctx context.Context, metricsBinding *vzapi.MetricsBinding, secret *k8scorev1.Secret, key string, log vzlog.VerrazzanoLogger) error {
	log.Debugw("Prometheus target config Secret is being altered", "resource", secret.GetName())
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		scrapeConfigData, err := getConfigDataFromSecret(secret, key)
		if err != nil {
			return err
		}
		updatedScrapeConfigs, err := r.deleteScrapeConfig(metricsBinding, scrapeConfigData, log, false)
		if err != nil {
			return err
		}
		newPromConfigData, err := yaml.JSONToYAML(updatedScrapeConfigs.Bytes())
		if err != nil {
			return log.ErrorfNewErr("Failed to convert scrape config JSON to YAML: %v", err)
		}
		secret.Data[key] = newPromConfigData
		return nil
	})
	return err
}

// deleteScrapeConfig is a mutation function that deletes the scrape config data from the Prometheus ConfigMap
func (r *Reconciler) deleteScrapeConfig(metricsBinding *vzapi.MetricsBinding, configData *gabs.Container, log vzlog.VerrazzanoLogger, isPromConfigMap bool) (*gabs.Container, error) {
	log.Debugw("Scrape Config is being deleted from the Prometheus Config", "resource", metricsBinding.GetName())

	// Verify the Owner Reference exists
	if len(metricsBinding.OwnerReferences) < 1 {
		return nil, fmt.Errorf("Failed to find Owner Reference found in the MetricsBinding: %s", metricsBinding.GetName())
	}

	// Delete scrape config with job name matching resource
	// parse the scrape config so we can manipulate it
	jobNameToDelete := createJobName(metricsBinding)

	var updatedConfigData *gabs.Container
	var err error
	if isPromConfigMap {
		err = metricsutils.EditScrapeJobInPrometheusConfig(configData, prometheusScrapeConfigsLabel, jobNameToDelete, nil)
		updatedConfigData = configData
	} else {
		updatedConfigData, err = metricsutils.EditScrapeJob(configData, jobNameToDelete, nil)
	}
	if err != nil {
		return nil, err
	}
	return updatedConfigData, err
}
