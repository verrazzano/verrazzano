// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"
	"fmt"
	"strings"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	k8score "k8s.io/api/core/v1"
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
	// Replacing underscores with dashes in name to appease Kubernetes requirements
	serviceMonitor := promoperapi.ServiceMonitor{}
	pmName, err := createServiceMonitorName(trait, 0)
	if err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to create Service Monitor name: %v", err)
	}
	serviceMonitor.SetName(strings.Replace(pmName, "_", "-", -1))
	serviceMonitor.SetNamespace(workload.GetNamespace())

	log.Debugf("Creating or updating the Service Monitor name: %s namespace: %s", serviceMonitor.Name, serviceMonitor.Namespace)
	// Create or Update Service Monitor with valid scrape config for the target workload
	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &serviceMonitor, func() error {
		return r.mutateServiceMonitorFromTrait(ctx, &serviceMonitor, trait, workload, traitDefaults, log)
	})
	if err != nil {
		return rel, result, err
	}

	rel = vzapi.QualifiedResourceRelation{APIVersion: promoperapi.SchemeGroupVersion.String(), Kind: promoperapi.ServiceMonitorsKind, Namespace: serviceMonitor.Namespace, Name: serviceMonitor.Name, Role: scraperRole}
	return rel, result, nil
}

// deleteServiceMonitor deletes the object responsible for transporting metrics from the source to Prometheus
func (r *Reconciler) deleteServiceMonitor(ctx context.Context, rel vzapi.QualifiedResourceRelation, log vzlog.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	log.Debugf("Deleting Service Monitor name: %s namespace: %s from resource relation", rel.Namespace, rel.Name)
	serviceMonitor := promoperapi.ServiceMonitor{}
	serviceMonitor.SetName(rel.Name)
	serviceMonitor.SetNamespace(rel.Namespace)
	if err := r.Delete(ctx, &serviceMonitor); err != nil {
		return rel, controllerutil.OperationResultNone, err
	}
	return rel, controllerutil.OperationResultUpdated, nil
}

// mutateServiceMonitorFromTrait mutates the Service Monitor to prepare for a create or update
// the Service Monitor reflects the specifications of the trait and the trait defaults
func (r *Reconciler) mutateServiceMonitorFromTrait(ctx context.Context, serviceMonitor *promoperapi.ServiceMonitor, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, log vzlog.VerrazzanoLogger) error {
	// Create the Service Monitor name from the trait if the label exists
	// Create the Service Monitor selector from the trait label if it exists
	if serviceMonitor.ObjectMeta.Labels == nil {
		serviceMonitor.ObjectMeta.Labels = map[string]string{}
	}
	serviceMonitor.Labels["release"] = "prometheus-operator"
	serviceMonitor.Spec.NamespaceSelector = promoperapi.NamespaceSelector{
		MatchNames: []string{workload.GetNamespace()},
	}

	// Fetch the secret by name if it is provided in either the trait or the trait defaults.
	secret, err := fetchSourceCredentialsSecretIfRequired(ctx, trait, traitDefaults, workload, r.Client)
	if err != nil {
		return log.ErrorfNewErr("Failed to fetch metrics source credentials: %v", err)
	}

	// Clear the existing endpoints to avoid duplications
	serviceMonitor.Spec.Endpoints = nil

	// Loop through ports in the trait and create scrape targets for each
	ports := getPortSpecs(trait, traitDefaults)
	for i := range ports {
		endpoint, err := r.createServiceMonitorEndpoint(ctx, trait, secret, i)
		if err != nil {
			return log.ErrorfNewErr("Failed to create an endpoint for the Service Monitor: %v", err)
		}
		serviceMonitor.Spec.Endpoints = append(serviceMonitor.Spec.Endpoints, endpoint)
	}

	return nil
}

// createServiceMonitorEndpoint creates an endpoint for a given port and trait
// this function effectively creates a scrape config for the trait target through the Service Monitor API
func (r *Reconciler) createServiceMonitorEndpoint(ctx context.Context, trait *vzapi.MetricsTrait, secret *k8score.Secret, portIncrement int) (promoperapi.Endpoint, error) {
	var endpoint promoperapi.Endpoint

	// Add the secret username and password if basic auth is required for this endpoint
	// The secret has to exist in the workload and namespace
	if secret != nil {
		trueVal := true
		endpoint.BasicAuth = &promoperapi.BasicAuth{
			Username: k8score.SecretKeySelector{
				LocalObjectReference: k8score.LocalObjectReference{
					Name: secret.Name,
				},
				Key:      "username",
				Optional: &trueVal,
			},
			Password: k8score.SecretKeySelector{
				LocalObjectReference: k8score.LocalObjectReference{
					Name: secret.Name,
				},
				Key:      "password",
				Optional: &trueVal,
			},
		}
	}

	endpoint.Scheme = "http"
	endpoint.Path = "/metrics"
	if trait.Spec.Path != nil {
		endpoint.Path = *trait.Spec.Path
	}

	// Set up the port appendix if necessary
	var portString string
	if portIncrement > 0 {
		portString = fmt.Sprintf("_%d", portIncrement)
	}

	// If Istio is enabled, use the tls config
	useHTTPS, err := useHTTPSForScrapeTarget(ctx, r.Client, trait)
	if err != nil {
		return endpoint, err
	}
	if useHTTPS {
		// The Prometheus Pod contains Istio certificates from the installation process
		// These certs are generated by Istio and are mounted as a volume on the Prometheus pod
		// ServiceMonitors are used to take advantage of these existing files because it allows us to reference the files in the volume
		certPath := "/etc/istio-certs"
		endpoint.Scheme = "https"
		endpoint.TLSConfig = &promoperapi.TLSConfig{
			CAFile:   fmt.Sprintf("%s/root-cert.pem", certPath),
			CertFile: fmt.Sprintf("%s/cert-chain.pem", certPath),
			KeyFile:  fmt.Sprintf("%s/key.pem", certPath),
		}
		endpoint.TLSConfig.InsecureSkipVerify = true
	}

	// Relabel the cluster name
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action:      "replace",
		Replacement: "local",
		TargetLabel: prometheusClusterNameLabel,
	})

	// Relabel to match the expected labels
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action: "keep",
		Regex:  fmt.Sprintf("true;%s;%s", trait.Labels[appObjectMetaLabel], trait.Labels[compObjectMetaLabel]),
		SourceLabels: []promoperapi.LabelName{
			promoperapi.LabelName(fmt.Sprintf("__meta_kubernetes_pod_annotation_verrazzano_io_metricsEnabled%s", portString)),
			"__meta_kubernetes_pod_label_app_oam_dev_name",
			"__meta_kubernetes_pod_label_app_oam_dev_component",
		},
	})

	// Relabel the address of the metrics endpoint
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action:      "replace",
		Regex:       `([^:]+)(?::\d+)?;(\d+)`,
		Replacement: "$1:$2",
		SourceLabels: []promoperapi.LabelName{
			"__address__",
			promoperapi.LabelName(fmt.Sprintf("__meta_kubernetes_pod_annotation_verrazzano_io_metricsPort%s", portString)),
		},
		TargetLabel: "__address__",
	})

	// Relabel the namespace label
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action:      "replace",
		Regex:       `(.*)`,
		Replacement: "$1",
		SourceLabels: []promoperapi.LabelName{
			"__meta_kubernetes_namespace",
		},
		TargetLabel: "namespace",
	})

	// Relabel the pod label
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action: "labelmap",
		Regex:  `__meta_kubernetes_pod_label_(.+)`,
	})

	// Relabel the pod name label
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action: "replace",
		SourceLabels: []promoperapi.LabelName{
			"__meta_kubernetes_pod_name",
		},
		TargetLabel: "pod_name",
	})

	// Drop the controller revision hash label
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action: "labeldrop",
		Regex:  `(controller_revision_hash)`,
	})

	// Relabel the webapp label
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action:      "replace",
		Regex:       `.*/(.*)$`,
		Replacement: "$1",
		SourceLabels: []promoperapi.LabelName{
			"name",
		},
		TargetLabel: "webapp",
	})

	return endpoint, nil
}
