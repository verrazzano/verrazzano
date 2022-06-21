// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/internal/metrics"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// updatePodMonitor creates or updates a Pod Monitor given the trait and workload parameters
// A pod monitor emulates a scrape config for Prometheus with the Prometheus Operator
func (r *Reconciler) updatePodMonitor(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, log vzlog.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	var rel vzapi.QualifiedResourceRelation

	// If the metricsTrait is being disabled then return nil for the config
	if !isEnabled(trait) || workload == nil {
		return rel, controllerutil.OperationResultNone, nil
	}

	// Creating a pod monitor with name and namespace
	// Replacing underscores with dashes in name to appease Kubernetes requirements
	pmName, err := createPodMonitorName(trait, 0)
	if err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to create Pod Monitor name: %v", err)
	}
	pmName = strings.Replace(pmName, "_", "-", -1)

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

	log.Debugf("Creating or updating the Pod Monitor for workload %s/%s", workload.GetNamespace(), workload.GetName())
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

	podMonitor := promoperapi.PodMonitor{}
	podMonitor.SetName(pmName)
	podMonitor.SetNamespace(workload.GetNamespace())
	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &podMonitor, func() error {
		return metrics.PopulatePodMonitor(scrapeInfo, &podMonitor, log)
	})
	if err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to create or update the service monitor for workload %s/%s: %v", workload.GetNamespace(), workload.GetName(), err)
	}

	rel = vzapi.QualifiedResourceRelation{APIVersion: promoperapi.SchemeGroupVersion.String(), Kind: promoperapi.ServiceMonitorsKind, Namespace: podMonitor.Namespace, Name: podMonitor.Name, Role: scraperRole}
	return rel, result, nil
}

// deletePodMonitor deletes the object responsible for transporting metrics from the source to Prometheus
func (r *Reconciler) deletePodMonitor(ctx context.Context, rel vzapi.QualifiedResourceRelation, trait *vzapi.MetricsTrait, log vzlog.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	if trait.DeletionTimestamp.IsZero() && isEnabled(trait) {
		log.Debugf("Maintaining Pod Monitor name: %s namespace: %s because the trait is enabled and not in the deletion process", rel.Name, rel.Namespace)
		return rel, controllerutil.OperationResultNone, nil
	}

	// Check if this is the last trait in the namespace
	// If so, delete the Istio certificate secret
	metricsTraitList := vzapi.MetricsTraitList{}
	err := r.List(ctx, &metricsTraitList, &client.ListOptions{Namespace: trait.GetNamespace()})
	if err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to list Metrics Trait in the namespace %s: %v", trait.GetNamespace(), err)
	}

	// It is the last trait if there is only one left since this one is being deleted
	if len(metricsTraitList.Items) == 1 {
		istioCertSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.IstioTLSSecretName,
				Namespace: trait.GetNamespace(),
			},
		}
		err := r.Delete(ctx, &istioCertSecret)
		if err != nil {
			return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to delete secret %s/%s: %v", trait.GetNamespace(), constants.IstioTLSSecretName, err)
		}
	}

	// If the trait is being deleted or is not enabled, delete the Pod Monitor
	log.Debugf("Deleting Pod Monitor name: %s/%s from resource relation", rel.Namespace, rel.Name)
	podMonitor := promoperapi.PodMonitor{}
	podMonitor.SetName(rel.Name)
	podMonitor.SetNamespace(rel.Namespace)
	if err := r.Delete(ctx, &podMonitor); err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to delete Pod Monitor %s/%s: %v", rel.Namespace, rel.Name, err)
	}
	return rel, controllerutil.OperationResultUpdated, nil
}
