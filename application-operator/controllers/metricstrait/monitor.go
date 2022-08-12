// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/internal/metrics"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// updateServiceMonitor creates or updates a service monitor given the trait and workload parameters
// A service monitor emulates a scrape config for Prometheus with the Prometheus Operator
func (r *Reconciler) updateServiceMonitor(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, log vzlog.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	var rel vzapi.QualifiedResourceRelation

	// If the metricsTrait is being disabled then return nil for the config
	if !isEnabled(trait) || workload == nil {
		return rel, controllerutil.OperationResultNone, nil
	}

	// Creating a service monitor with name and namespace
	pmName, err := createServiceMonitorName(trait, 0)
	if err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to create Service Monitor name: %v", err)
	}

	// Fetch the secret by name if it is provided in either the trait or the trait defaults.
	secret, err := fetchSourceCredentialsSecretIfRequired(ctx, trait, traitDefaults, workload, r.Client)
	if err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to fetch metrics source credentials: %v", err)
	}

	// Determine whether Istio is enabled for the workload
	useHTTPS, err := useHTTPSForScrapeTarget(ctx, r.Client, trait)
	if err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to determine if Istio was enabled for the target: %v", err)
	}

	wlsWorkload, err := isWLSWorkload(workload)
	if err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to determine if workload %s/&s was of type WLS: %v", workload.GetNamespace(), workload.GetName(), err)
	}
	vzPromLabels := !wlsWorkload

	log.Debugf("Creating or updating the Service Monitor for workload %s/%s", workload.GetNamespace(), workload.GetName())
	scrapeInfo := metrics.ScrapeInfo{
		Ports:              len(getPortSpecs(trait, traitDefaults)),
		BasicAuthSecret:    secret,
		IstioEnabled:       &useHTTPS,
		VZPrometheusLabels: &vzPromLabels,
		ClusterName:        clusters.GetClusterName(ctx, r.Client),
	}

	// Fill in the scrape info if it is populated in the trait
	if trait.Spec.Path != nil {
		scrapeInfo.Path = trait.Spec.Path
	}

	// Populate the keep labels to match the oam pod labels
	scrapeInfo.KeepLabels = map[string]string{
		"__meta_kubernetes_pod_label_app_oam_dev_name":      trait.Labels[appObjectMetaLabel],
		"__meta_kubernetes_pod_label_app_oam_dev_component": trait.Labels[compObjectMetaLabel],
	}

	serviceMonitor := promoperapi.ServiceMonitor{}
	serviceMonitor.SetName(pmName)
	serviceMonitor.SetNamespace(workload.GetNamespace())
	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &serviceMonitor, func() error {
		return metrics.PopulateServiceMonitor(scrapeInfo, &serviceMonitor, log)
	})
	if err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to create or update the service monitor for workload %s/%s: %v", workload.GetNamespace(), workload.GetName(), err)
	}

	rel = vzapi.QualifiedResourceRelation{APIVersion: promoperapi.SchemeGroupVersion.String(), Kind: promoperapi.ServiceMonitorsKind, Namespace: serviceMonitor.Namespace, Name: serviceMonitor.Name, Role: scraperRole}
	return rel, result, nil
}

// deleteServiceMonitor deletes the object responsible for transporting metrics from the source to Prometheus
func (r *Reconciler) deleteServiceMonitor(ctx context.Context, namespace string, name string, trait *vzapi.MetricsTrait, log vzlog.VerrazzanoLogger) (controllerutil.OperationResult, error) {
	if trait.DeletionTimestamp.IsZero() && isEnabled(trait) {
		log.Debugf("Maintaining Service Monitor name: %s namespace: %s because the trait is enabled and not in the deletion process", name, namespace)
		return controllerutil.OperationResultNone, nil
	}

	// If the trait is being deleted or is not enabled, delete the Service Monitor
	log.Debugf("Deleting Service Monitor name: %s namespace: %s from resource relation", name, namespace)
	serviceMonitor := promoperapi.ServiceMonitor{}
	serviceMonitor.SetName(name)
	serviceMonitor.SetNamespace(namespace)
	if err := r.Delete(ctx, &serviceMonitor); err != nil {
		return controllerutil.OperationResultNone, err
	}
	return controllerutil.OperationResultUpdated, nil
}
